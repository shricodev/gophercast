// Emulate a client that connects to a WebSocket server and receives audio chunks
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

type AudioClient struct {
	conn       *websocket.Conn
	outputFile *os.File
}

func NewAudioClient(serverURL, outputFileName string) (*AudioClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(serverURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	return &AudioClient{
		conn:       conn,
		outputFile: outputFile,
	}, nil
}

func (c *AudioClient) Start() {
	fmt.Println("connected to audio server, listening for audio chunks...")
	fmt.Printf("writing audio to file: %s\n", c.outputFile.Name())

	for {
		_, audioData, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Printf("error reading message: %v\n", err)
			break
		}

		_, err = c.outputFile.Write(audioData)
		if err != nil {
			fmt.Printf("error writing audio data to file: %v\n", err)
			break
		}

		fmt.Printf("wrote %d bytes to file\n", len(audioData))
	}
}

func (c *AudioClient) Close() {
	defer c.conn.Close()
	c.outputFile.Close()
}

func main() {
	serverURL := "ws://localhost:8080/ws"
	outputFile := "received_audio.wav"

	fmt.Printf("connecting to %s\n", serverURL)

	client, err := NewAudioClient(serverURL, outputFile)
	if err != nil {
		fmt.Printf("error creating audio client: %v\n", err)
		return
	}
	defer client.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("shutting down")
		os.Exit(0)
	}()

	client.Start()
}
