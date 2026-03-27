package types

import "testing"

func newTestPlaylist(n int) *Playlist {
	p := make(Playlist, n)
	for i := range p {
		p[i] = Track{
			Title:  "track-" + string(rune('A'+i)),
			Source: SourceLocalDir,
		}
	}
	return &p
}

func TestPlaylistMarkIsPlaying(t *testing.T) {
	p := newTestPlaylist(3)

	p.MarkIsPlaying(1)

	if (*p)[0].IsPlaying {
		t.Fatal("track 0 should not be playing")
	}
	if !(*p)[1].IsPlaying {
		t.Fatal("track 1 should be playing")
	}
	if (*p)[2].IsPlaying {
		t.Fatal("track 2 should not be playing")
	}

	// Move to a different track
	p.MarkIsPlaying(0)
	if !(*p)[0].IsPlaying {
		t.Fatal("track 0 should be playing after mark")
	}
	if (*p)[1].IsPlaying {
		t.Fatal("track 1 should not be playing after mark")
	}
}

func TestPlaylistCurrent(t *testing.T) {
	p := newTestPlaylist(3)

	// No track playing
	if p.Current() != nil {
		t.Fatal("expected nil when nothing is playing")
	}

	// Mark track 2 as playing
	p.MarkIsPlaying(2)
	cur := p.Current()
	if cur == nil {
		t.Fatal("expected non-nil current track")
	}
	if cur.Title != (*p)[2].Title {
		t.Fatalf("expected %s, got %s", (*p)[2].Title, cur.Title)
	}
}

func TestPlaylistNext(t *testing.T) {
	p := newTestPlaylist(3)

	// No track playing
	next := p.Next()
	if next == nil || next.Title != (*p)[0].Title {
		t.Fatal("expected first track when nothing is playing")
	}

	// Playing first
	p.MarkIsPlaying(0)
	next = p.Next()
	if next == nil || next.Title != (*p)[1].Title {
		t.Fatal("expected second track")
	}

	// Playing last
	p.MarkIsPlaying(2)
	next = p.Next()
	if next == nil || next.Title != (*p)[0].Title {
		t.Fatal("expected wrap to first track when at last")
	}
}

func TestPlaylistPrevious(t *testing.T) {
	p := newTestPlaylist(3)

	// No track playing
	prev := p.Previous()
	if prev == nil || prev.Title != (*p)[2].Title {
		t.Fatal("expected last track when nothing is playing")
	}

	// Playing last
	p.MarkIsPlaying(2)
	prev = p.Previous()
	if prev == nil || prev.Title != (*p)[1].Title {
		t.Fatal("expected second track")
	}

	// Playing first
	p.MarkIsPlaying(0)
	prev = p.Previous()
	if prev == nil || prev.Title != (*p)[2].Title {
		t.Fatal("expected wrap to last track when at first")
	}
}

func TestPlaylistNextPreviousEmpty(t *testing.T) {
	p := newTestPlaylist(0)

	if p.Next() != nil {
		t.Fatal("expected nil next on empty playlist")
	}
	if p.Previous() != nil {
		t.Fatal("expected nil previous on empty playlist")
	}
	if p.Current() != nil {
		t.Fatal("expected nil current on empty playlist")
	}
}

func TestPlaylistSingleTrack(t *testing.T) {
	p := newTestPlaylist(1)

	p.MarkIsPlaying(0)

	// Single track: Next() and Previous() wrap to the same track (first = last)
	next := p.Next()
	if next == nil || next.Title != (*p)[0].Title {
		t.Fatal("expected wrap to self on single track next")
	}
	prev := p.Previous()
	if prev == nil || prev.Title != (*p)[0].Title {
		t.Fatal("expected wrap to self on single track previous")
	}
	if p.Current() == nil {
		t.Fatal("expected current track")
	}
}
