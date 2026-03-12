package cmd

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/shricodev/gophercast/pkg/types"
	"github.com/shricodev/gophercast/tui"
)

var (
	dirToMP3   string
	ytURL      string
	ytPlaylist string

	random bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve subcommand serves the audio either with a local file, a directory of audio files or a yt URL",
	Long: `Serve is meant to serve an audio(mp3) file. Video files are not supported. You can
either provide a local mp3 file, a directory with list of mp3 files, or a
youtube URL or a youtube playlist URL.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if random && ytURL != "" {
			return fmt.Errorf("--random cannot be used with --youtube (YouTube video link)")
		}

		if random && dirToMP3 == "" && ytPlaylist == "" {
			return fmt.Errorf("--random must be used with either --dir-to-mp3 or --yt-playlist")
		}

		switch {
		case dirToMP3 != "":
			fmt.Println(dirToMP3)
		case ytURL != "":
			fmt.Println(ytURL)
		case ytPlaylist != "":
			fmt.Println(ytPlaylist)
		default:
			_, _, _ = tui.Start()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	sourceLocalDirStr := types.SourceLocalDir.String()
	sourceYoutubeStr := types.SourceYoutube.String()
	sourceYoutubePlaylistStr := types.SourceYoutubePlaylist.String()

	serveCmd.Flags().
		StringVarP(&dirToMP3, sourceLocalDirStr, "d", "", "Path to the audio(mp3) directory")
	serveCmd.Flags().StringVarP(&ytURL, sourceYoutubeStr, "y", "", "Link to the youtube video")
	serveCmd.Flags().
		StringVarP(&ytPlaylist, sourceYoutubePlaylistStr, "p", "", "Link to the youtube playlist")

	serveCmd.Flags().BoolVar(&random, "random", false, "Select and run the mp3 files in random")

	// serveCmd.MarkFlagsOneRequired(sourceLocalDirStr, sourceYoutubeStr, sourceYoutubePlaylistStr)
	serveCmd.MarkFlagsMutuallyExclusive(
		sourceLocalDirStr,
		sourceYoutubeStr,
		sourceYoutubePlaylistStr,
	)

	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
