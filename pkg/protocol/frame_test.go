package protocol

import (
	"testing"

	"github.com/shricodev/gophercast/internal/testutil"
)

func TestAudioFrameRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		seqNum       uint32
		sampleOffset uint64
		payloadSize  int
	}{
		{
			name:         "typical frame",
			seqNum:       42,
			sampleOffset: 1024,
			payloadSize:  4096,
		},
		{
			name:         "first frame",
			seqNum:       0,
			sampleOffset: 0,
			payloadSize:  4096,
		},
		{
			name:         "small payload",
			seqNum:       1,
			sampleOffset: 100,
			payloadSize:  64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, tt.payloadSize)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			frame := &AudioFrame{
				SeqNum:       tt.seqNum,
				SampleOffset: tt.sampleOffset,
				Payload:      payload,
			}

			data := frame.MarshalBinary()
			testutil.AssertLen(t, len(data), AudioFrameHeaderSize+tt.payloadSize)

			parsed, err := UnmarshalAudioFrame(data)
			testutil.AssertNoErr(t, err)

			if parsed.SeqNum != tt.seqNum {
				t.Fatalf("expected SeqNum %d, got %d", tt.seqNum, parsed.SeqNum)
			}
			if parsed.SampleOffset != tt.sampleOffset {
				t.Fatalf("expected SampleOffset %d, got %d", tt.sampleOffset, parsed.SampleOffset)
			}
			testutil.AssertLen(t, len(parsed.Payload), tt.payloadSize)

			for i := range payload {
				if parsed.Payload[i] != payload[i] {
					t.Fatalf("payload mismatch at byte %d: expected %d, got %d", i, payload[i], parsed.Payload[i])
				}
			}
		})
	}
}

func TestUnmarshalAudioFrameTooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty",
			data: []byte{},
		},
		{
			name: "less than header",
			data: make([]byte, AudioFrameHeaderSize-1),
		},
		{
			name: "way too short",
			data: []byte{0x01, 0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalAudioFrame(tt.data)
			testutil.AssertErr(t, err)
		})
	}
}

