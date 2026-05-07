// Package main provides the gophercast command-line application.
//
// GopherCast streams synchronized audio across devices on the same local
// network. The server decodes audio, broadcasts PCM frames over WebSocket,
// and clients play those frames at a shared wall-clock start time.
package main
