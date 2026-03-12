package utils

import (
	"testing"

	"github.com/shricodev/gophercast/internal/testutil"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "My Song",
			expected: "my_song",
		},
		{
			name:     "special characters",
			input:    "Song (feat Artist) [Official Video]",
			expected: "song_feat_artist_official_video",
		},
		{
			name:     "multiple spaces",
			input:    "Song   with   spaces",
			expected: "song_with_spaces",
		},
		{
			name:     "with extension",
			input:    "song.mp3",
			expected: "song",
		},
		{
			name:     "unicode characters",
			input:    "日本語の曲",
			expected: "",
		},
		{
			name:     "dashes preserved",
			input:    "my-song-name",
			expected: "my-song-name",
		},
		{
			name:     "leading trailing underscores",
			input:    "  hello world  ",
			expected: "hello_world",
		},
		{
			name:     "dots treated as extension",
			input:    "feat. someone",
			expected: "feat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			testutil.AssertEqual(t, result, tt.expected)
		})
	}
}

func TestUnsanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "underscores to spaces",
			input:    "my_song_name",
			contains: "MY SONG NAME",
		},
		{
			name:     "dashes to spaces",
			input:    "my-song-name",
			contains: "MY SONG NAME",
		},
		{
			name:     "with extension",
			input:    "my_song.mp3",
			contains: "MY SONG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UnsanitizeFilename(tt.input)
			testutil.AssertContains(t, result, tt.contains)
		})
	}
}
