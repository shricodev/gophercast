package types

import "testing"

func TestSourceIsValid(t *testing.T) {
	tests := []struct {
		name  string
		src   Source
		valid bool
	}{
		{"local dir", SourceLocalDir, true},
		{"local file", SourceLocalFile, true},
		{"youtube", SourceYoutube, true},
		{"youtube playlist", SourceYoutubePlaylist, true},
		{"empty", Source(""), false},
		{"unknown", Source("spotify"), false},
		{"typo", Source("youtube-playlist"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.src.IsValid() != tt.valid {
				t.Fatalf("expected IsValid=%v for source %q", tt.valid, tt.src)
			}
		})
	}
}
