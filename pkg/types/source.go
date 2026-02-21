package types

// Source represents the source of a track.
type Source string

const (
	SourceLocalDir        Source = "dir-to-mp3"
	SourceLocalFile       Source = "local-file"
	SourceYoutube         Source = "youtube"
	SourceYoutubePlaylist Source = "yt-playlist"
)

func (s Source) String() string {
	return string(s)
}

func (s Source) IsValid() bool {
	switch s {
	case SourceLocalDir, SourceLocalFile, SourceYoutube, SourceYoutubePlaylist:
		return true
	default:
		return false
	}
}
