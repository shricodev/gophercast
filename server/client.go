package server

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shricodev/gophercast/pkg/protocol"
)

// Client represents a connected WebSocket client.
type Client struct {
	id             string
	name           string
	addr           string
	conn           *websocket.Conn
	sendCtrl       chan []byte // buffered, for JSON control messages
	sendAudio      chan []byte // buffered, for binary audio frames
	server         *AudioServer
	audioLatencyNs int64 // client-reported audio pipeline latency
}

// readMessages reads messages from the WebSocket connection.
func (c *Client) readMessages() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		if msgType == websocket.TextMessage {
			c.handleTextMessage(data)
		}
	}
}

// handleTextMessage processes a JSON control message from the client.
func (c *Client) handleTextMessage(data []byte) {
	env, err := protocol.ParseEnvelope(data)
	if err != nil {
		c.server.logger.Error("failed to parse client message", "client", c.id, "error", err)
		return
	}

	switch env.Type {
	case protocol.MsgHello:
		var hello protocol.HelloMsg
		if err := json.Unmarshal(env.Data, &hello); err != nil {
			return
		}
		c.name = hello.Name
		c.audioLatencyNs = hello.AudioLatencyNs
		c.server.broadcastClientList()
		c.server.pushClientChange()
	}
}

// writeMessages writes messages to the WebSocket connection.
func (c *Client) writeMessages() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendCtrl:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case msg, ok := <-c.sendAudio:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Info returns the client's public info.
func (c *Client) Info() protocol.ClientInfo {
	return protocol.ClientInfo{
		ID:   c.id,
		Name: c.name,
		Addr: c.addr,
	}
}
