// Package protocol defines the wire protocol for GopherCast audio streaming.
// Control messages are sent as JSON WebSocket TextMessages, while audio data
// is sent as binary WebSocket BinaryMessages with a 12-byte header.
package protocol
