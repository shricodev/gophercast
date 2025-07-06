package types

type Playlist []Track

func (p *Playlist) Len() int {
	return len(*p)
}

func (p *Playlist) Current() *Track {
	for i := range *p {
		track := (*p)[i]
		if track.IsPlaying {
			return &track
		}
	}

	return nil
}

func (p *Playlist) Next() *Track {
	for i := range *p {
		track := (*p)[i]
		if track.IsPlaying && i+1 < len(*p) {
			return &(*p)[i+1]
		}
	}

	if len(*p) > 0 {
		return &(*p)[0]
	}

	return nil
}
