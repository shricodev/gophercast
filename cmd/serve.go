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
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve subcommand serves the audio either with a local file, a directory of audio files or a yt URL",
	Long: `Serve is meant to serve an audio(mp3) file. Video files are not supported. You can
either provide a local mp3 file, a directory with list of mp3 files, or a
youtube URL or a youtube playlist URL.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		switch {
		case dirToMP3 != "":
			fmt.Println(dirToMP3)
		case ytURL != "":
			fmt.Println(ytURL)
		case ytPlaylist != "":
			fmt.Println(ytPlaylist)
		default:
			d, y, p := tui.Run()
			fmt.Println(d, y, p)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	sourceLocalDirStr := types.SourceLocalDir.String()
	sourceYoutubeStr := types.SourceYoutube.String()
	sourceYoutubePlaylistStr := types.SourceYoutubePlaylist.String()

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	serveCmd.Flags().StringVarP(&dirToMP3, sourceLocalDirStr, "d", "", "Path to the audio(mp3) directory")
	serveCmd.Flags().StringVarP(&ytURL, sourceYoutubeStr, "y", "", "Link to the youtube video")
	serveCmd.Flags().StringVarP(&ytPlaylist, sourceYoutubePlaylistStr, "p", "", "Link to the youtube playlist")

	// serveCmd.MarkFlagsOneRequired(sourceLocalDirStr, sourceYoutubeStr, sourceYoutubePlaylistStr)
	serveCmd.MarkFlagsMutuallyExclusive(sourceLocalDirStr, sourceYoutubeStr, sourceYoutubePlaylistStr)

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
