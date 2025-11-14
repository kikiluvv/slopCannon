package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the global logger
func Init(verbose bool) {
	zerolog.TimeFieldFormat = time.RFC3339

	level := zerolog.InfoLevel
	if verbose {
		level = zerolog.DebugLevel
	}

	zerolog.SetGlobalLevel(level)

	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
		NoColor:    false,
	}

	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}

// NewLogger creates a new logger with optional writers
func NewLogger(writers ...io.Writer) zerolog.Logger {
	if len(writers) == 0 {
		return log.Logger
	}

	if len(writers) == 1 {
		return zerolog.New(writers[0]).With().Timestamp().Logger()
	}

	multi := zerolog.MultiLevelWriter(writers...)
	return zerolog.New(multi).With().Timestamp().Logger()
}

// WithComponent creates a logger with a component field
func WithComponent(component string) zerolog.Logger {
	return log.Logger.With().Str("component", component).Logger()
}
