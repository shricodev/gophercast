package downloader

import (
	"testing"

	"github.com/shricodev/gophercast/internal/testutil"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	testutil.AssertEqual(t, config.Quality, "192")
	testutil.AssertEqual(t, config.Format, "mp3")

	if config.Workers != 5 {
		t.Fatalf("expected 5 workers, got %d", config.Workers)
	}

	testutil.AssertNotEmpty(t, config.OutputDir.String())
	testutil.AssertContains(t, config.OutputDir.String(), ".gophercast")
	testutil.AssertContains(t, config.OutputDir.String(), "downloads")
}

func TestDownloaderShutdown(t *testing.T) {
	config := DefaultConfig()
	d := NewDownloader(config)

	d.Shutdown()

	if !d.IsShutdown() {
		t.Fatal("downloader should be shutdown after Shutdown()")
	}
}

func TestDownloaderDoubleShutdown(t *testing.T) {
	config := DefaultConfig()
	d := NewDownloader(config)

	// Should not panic on double shutdown
	d.Shutdown()
	d.Shutdown()

	if !d.IsShutdown() {
		t.Fatal("downloader should still be shutdown")
	}
}

func TestDownloadVideoAfterShutdown(t *testing.T) {
	config := DefaultConfig()
	d := NewDownloader(config)
	d.Shutdown()

	_, err := d.DownloadVideo("https://youtube.com/watch?v=test")
	testutil.AssertErr(t, err)
}

func TestDownloadPlaylistAfterShutdown(t *testing.T) {
	config := DefaultConfig()
	d := NewDownloader(config)
	d.Shutdown()

	_, err := d.DownloadPlaylist("https://youtube.com/playlist?list=test")
	testutil.AssertErr(t, err)
}
