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
	// pcmChunkSize is the number of decoded PCM bytes per audio frame.
	// 4096 bytes = 1024 samples of stereo 16-bit = ~23ms at 44.1kHz.
	pcmChunkSize = 4096

	// playbackLeadTime is the delay before playback starts so clients can buffer.
	playbackLeadTime = 750 * time.Millisecond

	// trackChangeLeadTime is the delay before the next track starts.
	trackChangeLeadTime = 500 * time.Millisecond
)

// streamPlaylist iterates through all tracks in the playlist and streams them.
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

		// Send per-client start times adjusted for audio pipeline latency.
		// broadcastPlaybackStart adds maxLatency to the lead time so the
		// highest-latency client still has enough buffering time.
		s.broadcastPlaybackStart(msgType, track.Title, sampleRate, channels, leadTime)

		// Compute the effective wait: leadTime + max client latency.
		// This matches the target time calculation in broadcastPlaybackStart.
		effectiveWait := leadTime + s.maxClientLatency()

		s.mu.Lock()
		s.playbackStart = time.Now().Add(effectiveWait)
		s.mu.Unlock()

		// Wait for all clients to be ready before streaming.
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
}

// streamTrack decodes an MP3 file to PCM and streams it at the correct bitrate.
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

	// bytesPerSample = channels * 2 (16-bit)
	bytesPerSample := channels * 2

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

			// Pace to real time using wall-clock drift correction
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

// maxClientLatency returns the maximum audio pipeline latency reported by
// any connected client. Used to ensure the lead time accounts for the
// slowest audio pipeline so all clients can start on time.
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

// determinePCMParams opens an MP3 file and returns its sample rate and channel count.
// go-mp3 always decodes to stereo (2 channels) 16-bit signed LE.
func determinePCMParams(trackPath types.Path) (sampleRate int, channels int) {
	f, err := os.Open(trackPath.String())
	if err != nil {
		// Fallback to CD-quality defaults
		return 44100, 2
	}
	defer f.Close()

	decoder, err := gomp3.NewDecoder(f)
	if err != nil {
		return 44100, 2
	}

	return decoder.SampleRate(), 2
}
