package types

type Source string

const (
	SourceLocalDir        Source = "dir-to-mp3"
	SourceYoutube         Source = "youtube"
	SourceYoutubePlaylist Source = "yt-playlist"
)

func (s Source) String() string {
	return string(s)
}

func (s Source) IsValid() bool {
	switch s {
	case SourceLocalDir, SourceYoutube, SourceYoutubePlaylist:
		return true
	default:
		return false
	}
}
