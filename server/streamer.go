package server

import (
	"fmt"
	"io"
	"os"
	"time"

	gomp3 "github.com/hajimehoshi/go-mp3"

	"github.com/shricodev/gophercast/pkg/protocol"
	"github.com/shricodev/gophercast/pkg/types"
)

const (
	// 4096 bytes = 1024 stereo 16-bit samples = ~23ms at 44.1kHz
	pcmChunkSize = 4096

	// how much lead time to give clients before first track starts
	playbackLeadTime = 750 * time.Millisecond

	// shorter lead time for subsequent tracks since sink is already warm
	trackChangeLeadTime = 500 * time.Millisecond
)

func (s *AudioServer) streamPlaylist() {
	for i := 0; i < s.playlist.Len(); i++ {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		s.mu.Lock()
		s.trackIndex = i
		s.playlist.MarkIsPlaying(i)
		track := (*s.playlist)[i]
		s.currentTrackTitle = track.Title
		s.mu.Unlock()

		sampleRate, channels := determinePCMParams(track.Path)

		leadTime := playbackLeadTime
		if i > 0 {
			leadTime = trackChangeLeadTime
		}

		msgType := protocol.MsgStartPlayback
		if i > 0 {
			msgType = protocol.MsgTrackChange
		}

		// each client gets a personalized start_at_ns adjusted for their latency
		s.broadcastPlaybackStart(msgType, track.Title, sampleRate, channels, leadTime)

		// wait the same duration we told clients to wait before sending audio
		effectiveWait := leadTime + s.maxClientLatency()

		s.mu.Lock()
		s.playbackStart = time.Now().Add(effectiveWait)
		s.mu.Unlock()

		time.Sleep(effectiveWait)

		if err := s.streamTrack(&track, sampleRate, channels); err != nil {
			s.logger.Error("error streaming track", "track", track.Title, "error", err)
		}
	}

	s.mu.Lock()
	s.state = protocol.StateStopped
	s.mu.Unlock()

	s.broadcastControl(protocol.MsgStopPlayback, protocol.StopPlaybackMsg{
		Reason: "playlist_ended",
	})

	close(s.playbackDoneCh)
}

func (s *AudioServer) streamTrack(track *types.Track, sampleRate, channels int) error {
	f, err := os.Open(track.Path.String())
	if err != nil {
		return fmt.Errorf("open track file: %w", err)
	}
	defer f.Close()

	decoder, err := gomp3.NewDecoder(f)
	if err != nil {
		return fmt.Errorf("create mp3 decoder: %w", err)
	}

	buf := make([]byte, pcmChunkSize)
	var seqNum uint32
	var sampleOffset uint64
	startTime := time.Now()

	bytesPerSample := channels * 2 // 16-bit samples

	for {
		select {
		case <-s.ctx.Done():
			return nil
		default:
		}

		n, err := io.ReadFull(decoder, buf)
		if n > 0 {
			frame := &protocol.AudioFrame{
				SeqNum:       seqNum,
				SampleOffset: sampleOffset,
				Payload:      buf[:n],
			}
			s.broadcastAudioFrame(frame)

			seqNum++
			sampleOffset += uint64(n / bytesPerSample)

			// pace output to real time so clients don't buffer too far ahead
			expectedElapsed := time.Duration(float64(sampleOffset) / float64(sampleRate) * float64(time.Second))
			actualElapsed := time.Since(startTime)
			if expectedElapsed > actualElapsed {
				time.Sleep(expectedElapsed - actualElapsed)
			}
		}

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read decoded pcm: %w", err)
		}
	}

	return nil
}

// maxClientLatency finds the worst-case latency among connected clients.
// We use this to set a lead time that works for everyone, including the slow ones.
func (s *AudioServer) maxClientLatency() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var maxNs int64
	for _, client := range s.clients {
		if client.audioLatencyNs > maxNs {
			maxNs = client.audioLatencyNs
		}
	}
	return time.Duration(maxNs)
}

// determinePCMParams reads the sample rate from the MP3 header.
// Note: go-mp3 always decodes to stereo 16-bit LE regardless of the source.
func determinePCMParams(trackPath types.Path) (sampleRate int, channels int) {
	f, err := os.Open(trackPath.String())
	if err != nil {
		return 44100, 2 // fallback to CD quality
	}
	defer f.Close()

	decoder, err := gomp3.NewDecoder(f)
	if err != nil {
		return 44100, 2
	}

	return decoder.SampleRate(), 2
}
