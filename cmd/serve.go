package cmd

/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/

import (
	"fmt"

	"github.com/spf13/cobra"
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
	Long: `It is meant to serve an audio(mp3) file. Video files are not supported. You can
either provide a local mp3 file, a directory with list of mp3 files, or a
youtube URL or a youtube playlist URL.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("serve called")
		switch {
		case dirToMP3 != "":
			fmt.Println(dirToMP3)
		case ytURL != "":
			fmt.Println(ytURL)
		case ytPlaylist != "":
			fmt.Println(ytPlaylist)
		default:
			fmt.Println("Invalid arguments")
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	serveCmd.Flags().StringVarP(&dirToMP3, "dir-to-mp3", "d", "", "Path to the directory")
	serveCmd.Flags().StringVarP(&ytURL, "yt", "y", "", "Link to the youtube video")
	serveCmd.Flags().StringVarP(&ytPlaylist, "yt-playlist", "p", "", "Link to the youtube playlist")

	serveCmd.MarkFlagsOneRequired("dir-to-mp3", "yt", "yt-playlist")
	serveCmd.MarkFlagsMutuallyExclusive("dir-to-mp3", "yt", "yt-playlist")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
