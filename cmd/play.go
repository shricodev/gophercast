package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/shricodev/gophercast/client"
)

var (
	host       string
	port       int
	clientName string
	outputFile string
)

// playCmd represents the play command.
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Connect to a GopherCast server and play audio",
	Long: `Connect to a running GopherCast server and play synchronized audio.
The client must connect during the lobby phase before playback starts.

Examples:
  gophercast play --host 192.168.1.10 --port 8080
  gophercast play --host 192.168.1.10 --port 8080 --name "Kitchen Speaker"
  gophercast play --host 192.168.1.10 --port 8080 --output recording.pcm`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := fmt.Sprintf("ws://%s:%d/ws", host, port)

		name := clientName
		if name == "" {
			hostname, err := os.Hostname()
			if err == nil {
				name = hostname
			} else {
				name = "gophercast-client"
			}
		}

		var sink client.AudioSink
		if outputFile != "" {
			sink = client.NewFileSink(outputFile)
			fmt.Printf("Writing audio to file: %s\n", outputFile)
		} else {
			sink = client.NewSystemAudioSink()
		}

		fmt.Printf("Connecting to %s as %q...\n", serverURL, name)

		c, err := client.NewAudioClient(serverURL, name, sink)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not connect to server at %s\n", serverURL)
			fmt.Fprintf(os.Stderr, "Make sure the server is running and the address is correct.\n")
			return nil
		}

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-sigCh
			fmt.Println("\nDisconnecting...")
			c.Close()
		}()

		fmt.Println("Connected! Waiting for playback to start...")

		err = c.Start()
		c.Close()

		if err == nil {
			fmt.Println("Playback finished.")
			return nil
		}

		// Handle specific error types with friendly messages
		if errors.Is(err, client.ErrRejected) {
			fmt.Fprintf(os.Stderr, "Server rejected the connection: %s\n", err)
			fmt.Fprintf(os.Stderr, "You can only join during the lobby phase before playback starts.\n")
			return nil
		}

		if errors.Is(err, client.ErrDisconnected) {
			fmt.Println("Disconnected from server.")
			return nil
		}

		fmt.Fprintf(os.Stderr, "Connection lost: %v\n", err)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(playCmd)

	playCmd.Flags().StringVar(&host, "host", "", "Host of the server")
	playCmd.Flags().IntVar(&port, "port", 8080, "Port of the server")
	playCmd.Flags().StringVarP(&clientName, "name", "n", "", "Client display name (default: hostname)")
	playCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write raw PCM to file instead of playing audio")

	playCmd.MarkFlagsRequiredTogether("host", "port")
}
