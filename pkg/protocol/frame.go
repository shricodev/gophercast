package protocol

import (
	"encoding/binary"
	"fmt"
)

// AudioFrameHeaderSize is the size of the binary audio frame header in bytes.
// 4 bytes for sequence number + 8 bytes for sample offset.
const AudioFrameHeaderSize = 12

// AudioFrame represents a single audio frame with metadata.
type AudioFrame struct {
	SeqNum       uint32
	SampleOffset uint64
	Payload      []byte
}

// MarshalBinary serializes an AudioFrame into bytes.
func (f *AudioFrame) MarshalBinary() []byte {
	buf := make([]byte, AudioFrameHeaderSize+len(f.Payload))
	binary.BigEndian.PutUint32(buf[0:4], f.SeqNum)
	binary.BigEndian.PutUint64(buf[4:12], f.SampleOffset)
	copy(buf[AudioFrameHeaderSize:], f.Payload)
	return buf
}

// UnmarshalAudioFrame deserializes bytes into an AudioFrame.
func UnmarshalAudioFrame(data []byte) (*AudioFrame, error) {
	if len(data) < AudioFrameHeaderSize {
		return nil, fmt.Errorf("audio frame too short: %d bytes", len(data))
	}

	return &AudioFrame{
		SeqNum:       binary.BigEndian.Uint32(data[0:4]),
		SampleOffset: binary.BigEndian.Uint64(data[4:12]),
		Payload:      data[AudioFrameHeaderSize:],
	}, nil
}
