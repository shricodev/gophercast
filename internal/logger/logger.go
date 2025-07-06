// Package logger handles logging with slog
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shricodev/gophercast/pkg/types"
)

type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

type Config struct {
	Level      LogLevel
	Output     string
	TimeFormat string
}

type Logger struct {
	*slog.Logger
	config Config
	writer io.Writer
}

func New(config Config) (*Logger, error) {
	var writer io.Writer

	if config.TimeFormat == "" {
		config.TimeFormat = time.RFC3339
	}

	if strings.ToLower(config.Output) == "stdout" {
		writer = os.Stdout
	} else {
		path, err := types.NewPath(config.Output)
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}

		writer, err = createFileWriter(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %w", err)
		}
	}

	slogLevel := convertLogLevel(config.Level)

	opts := &slog.HandlerOptions{
		Level: slogLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format(config.TimeFormat))
				}
			}
			return a
		},
	}

	handler := slog.NewJSONHandler(writer, opts)
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		config: config,
		writer: writer,
	}, nil
}

func createFileWriter(basePath types.Path) (io.Writer, error) {
	now := time.Now()

	dirPath := filepath.Join(basePath.String(), fmt.Sprintf("%d", now.Year()), fmt.Sprintf("%02d", now.Month()))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	fileName := fmt.Sprintf("%02d.json", now.Day())
	filePath := filepath.Join(dirPath, fileName)

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return file, nil
}

func convertLogLevel(level LogLevel) slog.Level {
	switch level {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.Logger.DebugContext(ctx, msg, args...)
}

func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.Logger.InfoContext(ctx, msg, args...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.Logger.WarnContext(ctx, msg, args...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.Logger.ErrorContext(ctx, msg, args...)
}

func (l *Logger) ClientConnected(clientID string, clientIP string) {
	l.Info("client connected", slog.String("event", "client_connected"),
		slog.String("client_id", clientID), slog.String("client_ip", clientIP),
		slog.Time("timestamp", time.Now()))
}

func (l *Logger) ClientDisconnected(clientID string, reason string) {
	l.Info("client disconnected", slog.String("event", "client_disconnected"),
		slog.String("client_id", clientID), slog.String("reason", reason),
		slog.Time("timestamp", time.Now()),
	)
}

func (l *Logger) PlaylistCreated(playlistID string, trackCount int) {
	l.Info("Playlist created", slog.String("event", "playlist_created"),
		slog.String("playlist_id", playlistID), slog.Int("track_count",
			trackCount), slog.Time("timestamp", time.Now()),
	)
}

func (l *Logger) TrackPlayed(trackID string, clientID string, duration time.Duration) {
	l.Info("track played", slog.String("event", "track_played"),
		slog.String("track_id", trackID), slog.String("client_id", clientID),
		slog.Duration("duration", duration), slog.Time("timestamp",
			time.Now()),
	)
}

func (l *Logger) ServerStarted(port int) {
	l.Info("server started", slog.String("event", "server_started"),
		slog.Int("port", port), slog.Time("timestamp", time.Now()),
	)
}

func (l *Logger) ServerStopped(reason string) {
	l.Info("server stopped", slog.String("event", "server_stopped"),
		slog.String("reason", reason), slog.Time("timestamp", time.Now()),
	)
}

func (l *Logger) ErrorOccurred(err error, context string) {
	l.Error("error occurred", slog.String("event", "error"),
		slog.String("error", err.Error()), slog.String("context", context),
		slog.Time("timestamp", time.Now()),
	)
}

func (l *Logger) Close() error {
	if closer, ok := l.writer.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

type RotatingFileLogger struct {
	*Logger
	basePath    string
	currentDate string
}

func NewRotatingFileLogger(config Config) (*RotatingFileLogger, error) {
	if strings.ToLower(config.Output) == "stdout" {
		return nil, fmt.Errorf("rotating logger requires a file path, got stdout")
	}

	logger, err := New(config)
	if err != nil {
		return nil, err
	}

	return &RotatingFileLogger{
		Logger:      logger,
		basePath:    config.Output,
		currentDate: time.Now().Format("2006-01-02"),
	}, nil
}

func (r *RotatingFileLogger) checkAndRotate() error {
	today := time.Now().Format("2006-01-02")
	if today != r.currentDate {
		if err := r.Close(); err != nil {
			return fmt.Errorf("failed to close current log file: %w", err)
		}

		config := r.config
		config.Output = r.basePath
		newLogger, err := New(config)
		if err != nil {
			return fmt.Errorf("failed to create new rotated logger: %w", err)
		}

		r.Logger = newLogger
		r.currentDate = today
	}
	return nil
}

// Info override logging methods to include rotation check
func (r *RotatingFileLogger) Info(msg string, args ...any) {
	r.checkAndRotate()
	r.Logger.Info(msg, args...)
}

func (r *RotatingFileLogger) Debug(msg string, args ...any) {
	r.checkAndRotate()
	r.Logger.Debug(msg, args...)
}

func (r *RotatingFileLogger) Warn(msg string, args ...any) {
	r.checkAndRotate()
	r.Logger.Warn(msg, args...)
}

func (r *RotatingFileLogger) Error(msg string, args ...any) {
	r.checkAndRotate()
	r.Logger.Error(msg, args...)
}
