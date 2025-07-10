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
