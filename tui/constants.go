// Package tui provides the terminal user interface for Gophercast.
// It is responsible for rendering the UI and handling user input.
package tui

const (
	directory       = "Directory"
	youtube         = "YouTube"
	youtubePlaylist = "Playlist"
)

type screen int

const (
	screenMenu screen = iota
	screenPickDir
	screenInputYoutube
	screenInputPlaylist
	screenAppStarting
	screenAppRunning
)
