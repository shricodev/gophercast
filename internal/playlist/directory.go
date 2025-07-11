// Package playlist is the playlist builder
package playlist

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/shricodev/gophercast/pkg/types"
)

func BuildPlaylistFromDir(root types.Path) (*types.Playlist, error) {
	return buildPlaylistFromFS(os.DirFS(root.String()), root)
}

func buildPlaylistFromFS(fsys fs.FS, root types.Path) (*types.Playlist, error) {
	var tracks types.Playlist

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(d.Name()) != ".mp3" {
			return nil
		}

		p, err := types.NewPathInFS(fsys, path, root)
		if err != nil {
			return err
		}

		tracks = append(tracks, types.Track{
			Title:  d.Name(),
			Path:   p,
			Source: types.SourceLocalDir,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &tracks, nil
}
