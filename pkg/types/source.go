package types

type Source string

const (
	SourceLocal           Source = "local"
	SourceYoutube         Source = "youtube"
	SourceYoutubePlaylist Source = "yt-playlist"
)

func (s Source) String() string {
	return string(s)
}

func (s Source) IsValid() bool {
	switch s {
	case SourceLocal, SourceYoutube, SourceYoutubePlaylist:
		return true
	default:
		return false
	}
}
