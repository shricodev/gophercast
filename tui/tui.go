// Package tui launches the tui with BubbleTea
package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/shricodev/gophercast/internal/downloader"
	"github.com/shricodev/gophercast/pkg/types"
)

var (
	docStyle  = lipgloss.NewStyle().Margin(1, 1)
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(1, 0)
)

func run() tea.Msg {
	// perform the actual work
	time.Sleep(2 * time.Second)
	return appStartedMsg{}
}

func downloadYouTubeVideo(url string, d *downloader.Downloader) tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan downloader.DownloadProgress, 1)
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case progressChan <- progress:
			default:
			}
		})

		resultChan := make(chan downloadResult, 1)
		go func() {
			track, err := d.DownloadVideo(url)
			resultChan <- downloadResult{
				track: track,
				err:   err,
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			select {
			case progress := <-progressChan:
				fmt.Println(progress)
			case result := <-resultChan:
				if result.err != nil {
					return errMsg{err: result.err}
				}
				return downloadCompleteMsg{
					tracks: types.Playlist{result.track},
					path:   result.track.Path,
				}
			case <-ctx.Done():
				return errMsg{ctx.Err()}
			}
		}
	}
}

func downloadYouTubePlaylist(url string, d *downloader.Downloader) tea.Cmd {
	return func() tea.Msg {
		progressChan := make(chan downloader.DownloadProgress, 1)
		d.SetProgressCallback(func(progress downloader.DownloadProgress) {
			select {
			case progressChan <- progress:
			default:
			}
		})

		resultChan := make(chan playlistResult, 1)
		go func() {
			tracks, err := d.DownloadPlaylist(url)
			resultChan <- playlistResult{
				tracks: tracks,
				err:    err,
			}
		}()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			select {
			case progress := <-progressChan:
				_ = progress
			case result := <-resultChan:
				if result.err != nil {
					return errMsg{result.err}
				}

				var dirPath types.Path
				var tracks types.Playlist
				if result.tracks != nil {
					tracks = *result.tracks
					if tracks.Len() > 0 {
						dirPath = tracks[0].Path
					}
				}

				return downloadCompleteMsg{
					tracks: tracks,
					path:   dirPath,
				}
			case <-ctx.Done():
				return errMsg{ctx.Err()}
			}
		}
	}
}

func Start() (types.Path, string, string) {
	m := InitialModel()
	p := tea.NewProgram(m)

	go func() {
		quitCh := make(chan os.Signal, 1)
		signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)

		<-quitCh

		fmt.Println("shutdown requested")
		p.Quit()
	}()

	finalModel, err := p.Run()
	if err != nil {
		log.Fatal(err)
	}

	mo := finalModel.(model)
	if mo.downloader != nil {
		mo.downloader.Shutdown()
	}

	return mo.dirToMp3, mo.youtubePlaylistURL, mo.youtubeURL
}

type downloadResult struct {
	track types.Track
	err   error
}

type playlistResult struct {
	tracks *types.Playlist
	err    error
}
