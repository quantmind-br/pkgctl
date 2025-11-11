package logging

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config holds logger configuration
type Config struct {
	Level   string
	LogFile string
	NoColor bool
}

// NewLogger creates a new zerolog logger with dual output (console + file)
func NewLogger(cfg Config) *zerolog.Logger {
	// Enable stack trace marshaling
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Determine log level
	level := parseLevel(cfg.Level)

	// Console writer (colored output for TTY)
	consoleWriter := zerolog.ConsoleWriter{
		Out:        newProgressSafeWriter(os.Stderr),
		TimeFormat: "15:04:05",
		NoColor:    cfg.NoColor,
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	// File logger if path provided
	if cfg.LogFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(dir, 0755); err == nil {
			fileWriter := &lumberjack.Logger{
				Filename:   cfg.LogFile,
				MaxSize:    10, // MB
				MaxBackups: 3,
				MaxAge:     28, // days
				Compress:   true,
			}
			writers = append(writers, fileWriter)
		}
	}

	// Create multi-writer
	multi := zerolog.MultiLevelWriter(writers...)

	// Create logger
	logger := zerolog.New(multi).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &logger
}

// progressSafeWriter ensures console logs don't overlap with progress bars/spinners.
// It clears the current terminal line before writing and tracks line boundaries
// so the clear code runs only once per log entry.
type progressSafeWriter struct {
	out       io.Writer
	lineStart bool
	mu        sync.Mutex
	clearSeq  []byte
}

func newProgressSafeWriter(out io.Writer) *progressSafeWriter {
	return &progressSafeWriter{
		out:       out,
		lineStart: true,
		clearSeq:  []byte("\r\033[2K"),
	}
}

func (w *progressSafeWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.lineStart {
		if _, err := w.out.Write(w.clearSeq); err != nil {
			return 0, err
		}
		w.lineStart = false
	}

	n, err := w.out.Write(p)

	if n > 0 && bytes.LastIndexByte(p[:n], '\n') == n-1 {
		w.lineStart = true
	}

	return n, err
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

// NewTestLogger creates a logger for testing that writes to a buffer
func NewTestLogger(w io.Writer) *zerolog.Logger {
	logger := zerolog.New(w).With().Timestamp().Logger()
	return &logger
}
