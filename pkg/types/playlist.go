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

func (p *Playlist) Previous() *Track {
	for i := range *p {
		track := (*p)[i]
		if track.IsPlaying && i-1 >= 0 {
			return &(*p)[i-1]
		}
	}

	if len(*p) > 0 {
		return &(*p)[len(*p)-1]
	}

	return nil
}

func (p *Playlist) MarkIsPlaying(idx int) {
	for i := range *p {
		(*p)[i].IsPlaying = (i == idx)
	}
}
