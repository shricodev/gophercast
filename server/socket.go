package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

type AudioServer struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	audioFile  string
}

func NewAudioServer(audioFile string) *AudioServer {
	return &AudioServer{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		audioFile:  audioFile,
	}
}

type Client struct {
	server *AudioServer
	conn   *websocket.Conn
	send   chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (a *AudioServer) Run() {
	go a.streamAudio()

	for {
		select {
		case client := <-a.register:
			a.clients[client] = true
			fmt.Printf("client connected, total clients: %d\n", len(a.clients))

		case client := <-a.unregister:
			delete(a.clients, client)
			close(client.send)
			fmt.Printf("client disconnected, total clients: %d\n", len(a.clients))

		case audioData := <-a.broadcast:
			for client := range a.clients {
				select {
				case client.send <- audioData:
				default:
					close(client.send)
					delete(a.clients, client)
				}
			}
		}
	}
}

func (a *AudioServer) streamAudio() {
	fmt.Printf("starting to stream audio file: %s\n", a.audioFile)

	for {
		audioData, err := os.ReadFile(a.audioFile)
		if err != nil {
			fmt.Printf("error reading audio file: %v\n", err)
			fmt.Printf("sleeping for %d seconds\n", 5)
			time.Sleep(5 * time.Second)
			continue
		}

		chunkSize := 4096
		for i := 0; i < len(audioData); i += chunkSize {
			end := i + chunkSize
			end = min(end, len(audioData))

			chunk := audioData[i:end]

			select {
			case a.broadcast <- chunk:
			default:
				fmt.Printf("dropping chunk\n")
			}

			time.Sleep(10 * time.Millisecond)
		}

		fmt.Println("audio file finished, looping...")
		time.Sleep(1 * time.Second)
	}
}

func (a *AudioServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("websocket upgrade failed: %v\n", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte),
		server: a,
	}

	a.register <- client

	go client.readMessages()
	go client.writeMessages()
}

func (c *Client) readMessages() {
	defer func() {
		c.server.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Printf("error reading message: %v\n", err)
			break
		}
	}
}

func (c *Client) writeMessages() {
	defer c.conn.Close()

	for {
		select {
		case audioData, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
				fmt.Printf("error writing message: %v\n", err)
				return
			}
		}
	}
}
