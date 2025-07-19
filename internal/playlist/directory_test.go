// Package playlist provides functionality for managing playlists.
package playlist

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/shricodev/gophercast/internal/testutil"
	"github.com/shricodev/gophercast/pkg/types"
)

// TestBuildPlaylistFromFS tests the BuildPlaylistFromFS function.
func TestBuildPlaylistFromFS(t *testing.T) {
	tests := []struct {
		name              string
		fs                fstest.MapFS
		expectedTracksLen int
		wantErr           bool
	}{
		{
			name: "basic mp3 files",
			fs: fstest.MapFS{
				"song1.mp3": &fstest.MapFile{Data: []byte("fake mp3 data")},
				"song2.mp3": &fstest.MapFile{Data: []byte("fake mp3 data")},
			},
			expectedTracksLen: 2,
			wantErr:           false,
		},
		{
			name: "mixed file types",
			fs: fstest.MapFS{
				"song1.mp3":  &fstest.MapFile{Data: []byte("fake mp3 data")},
				"song2.mp4":  &fstest.MapFile{Data: []byte("fake mp4 data")},
				"readme.txt": &fstest.MapFile{Data: []byte("fake text data")},
				"image3.jpg": &fstest.MapFile{Data: []byte("fake image data")},
				"song3.MP3":  &fstest.MapFile{Data: []byte("uppercase fake mp3 data")},
			},
			expectedTracksLen: 1,
			wantErr:           false,
		},
		{
			name: "nested directories",
			fs: fstest.MapFS{
				"album1":                  &fstest.MapFile{Mode: fs.ModeDir},
				"album1/song1.mp3":        &fstest.MapFile{Data: []byte("fake mp3 data")},
				"album1/song2.mp3":        &fstest.MapFile{Data: []byte("fake mp3 data")},
				"album2":                  &fstest.MapFile{Mode: fs.ModeDir},
				"album2/song3.mp3":        &fstest.MapFile{Data: []byte("fake mp3 data")},
				"album2/subdir":           &fstest.MapFile{Mode: fs.ModeDir},
				"album2/subdir/song4.mp3": &fstest.MapFile{Data: []byte("fake mp3 data")},
			},
			expectedTracksLen: 4,
			wantErr:           false,
		},
		{
			name: "empty directory",
			fs: fstest.MapFS{
				"empty_dir": &fstest.MapFile{Mode: fs.ModeDir},
			},
			expectedTracksLen: 0,
			wantErr:           false,
		},
		{
			name:              "completely empty",
			fs:                fstest.MapFS{},
			expectedTracksLen: 0,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Don't use NewPath here, as the filesystem is mocked.
			rootPath := types.Path("/test/root")

			playlist, err := buildPlaylistFromFS(tt.fs, rootPath)

			if tt.wantErr {
				testutil.AssertErr(t, err)
				return
			}

			testutil.AssertNoErr(t, err)
			testutil.AssertLen(t, playlist.Len(), tt.expectedTracksLen)

			for _, track := range *playlist {
				testutil.AssertNotEmpty(t, track.Title)
				testutil.AssertEqual(t, string(track.Source), string(types.SourceLocalDir))
				testutil.AssertContains(t, track.Title, ".mp3")
				testutil.AssertNotNil(t, track.Path)
			}
		})
	}
}
