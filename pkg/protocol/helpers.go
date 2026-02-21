package protocol

import (
	"encoding/json"
	"fmt"
)

// MarshalEnvelope creates a JSON envelope for a control message.
func MarshalEnvelope(msgType MessageType, data any) ([]byte, error) {
	rawData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope data: %w", err)
	}

	env := Envelope{
		Type: msgType,
		Data: rawData,
	}

	return json.Marshal(env)
}

// ParseEnvelope parses a JSON envelope from raw bytes.
func ParseEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	return &env, nil
}
