package protocol

import (
	"encoding/json"
	"testing"

	"github.com/shricodev/gophercast/internal/testutil"
)

func TestMarshalEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		data    any
	}{
		{
			name:    "hello message",
			msgType: MsgHello,
			data:    HelloMsg{Name: "test-client"},
		},
		{
			name:    "start playback",
			msgType: MsgStartPlayback,
			data: StartPlaybackMsg{
				TrackTitle: "song.mp3",
				SampleRate: 44100,
				Channels:   2,
				StartAtNs:  1234567890,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := MarshalEnvelope(tt.msgType, tt.data)
			testutil.AssertNoErr(t, err)

			env, err := ParseEnvelope(bytes)
			testutil.AssertNoErr(t, err)
			testutil.AssertEqual(t, string(env.Type), string(tt.msgType))

			// Verify the data round-trips correctly
			reMarshaled, err := json.Marshal(tt.data)
			testutil.AssertNoErr(t, err)

			var expected, actual any
			testutil.AssertNoErr(t, json.Unmarshal(reMarshaled, &expected))
			testutil.AssertNoErr(t, json.Unmarshal(env.Data, &actual))

			expectedJSON, _ := json.Marshal(expected)
			actualJSON, _ := json.Marshal(actual)
			testutil.AssertEqual(t, string(actualJSON), string(expectedJSON))
		})
	}
}

func TestParseEnvelopeInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty bytes",
			input: []byte{},
		},
		{
			name:  "invalid json",
			input: []byte("not json"),
		},
		{
			name:  "incomplete json",
			input: []byte(`{"type": "hello"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEnvelope(tt.input)
			testutil.AssertErr(t, err)
		})
	}
}

