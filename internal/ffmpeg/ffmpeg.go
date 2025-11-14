package ffmpeg

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

// Executor handles all ffmpeg operations with progress streaming
type Executor struct {
	logger      zerolog.Logger
	ffmpegPath  string
	ffprobePath string
	threads     int
}

// New creates a new ffmpeg executor
func New(logger zerolog.Logger, threads int) (*Executor, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	return &Executor{
		logger:      logger.With().Str("component", "ffmpeg").Logger(),
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		threads:     threads,
	}, nil
}

// Run executes ffmpeg with the given arguments and streams progress
func (e *Executor) Run(ctx context.Context, opts RunOptions) error {
	if len(opts.Args) == 0 {
		return fmt.Errorf("no arguments provided")
	}

	// Build args with threads BEFORE other arguments
	baseArgs := []string{"-y", "-hide_banner", "-loglevel", "info"}

	if e.threads > 0 {
		baseArgs = append(baseArgs, "-threads", fmt.Sprintf("%d", e.threads))
	}

	baseArgs = append(baseArgs, "-progress", "pipe:2")
	args := append(baseArgs, opts.Args...)

	e.logger.Debug().
		Str("cmd", "ffmpeg").
		Strs("args", args).
		Msg("executing ffmpeg")

	cmd := exec.CommandContext(ctx, e.ffmpegPath, args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stderr (progress + logs)
	go func() {
		defer wg.Done()
		e.streamOutput(stderr, opts.ProgressHandler, opts.LogHandler)
	}()

	// Stream stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if opts.LogHandler != nil {
				opts.LogHandler(scanner.Text())
			}
		}
	}()

	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.Canceled {
			return ctx.Err()
		}
		return fmt.Errorf("ffmpeg execution failed: %w", err)
	}

	e.logger.Debug().Msg("ffmpeg execution completed")
	return nil
}

// streamOutput parses ffmpeg output and calls handlers
func (e *Executor) streamOutput(r io.Reader, progressHandler func(*Progress), logHandler func(string)) {
	scanner := bufio.NewScanner(r)
	progressData := &Progress{}

	for scanner.Scan() {
		line := scanner.Text()

		if logHandler != nil {
			logHandler(line)
		}

		// Parse progress lines
		if strings.HasPrefix(line, "frame=") {
			fmt.Sscanf(line, "frame=%d", &progressData.Frame)
		} else if strings.HasPrefix(line, "fps=") {
			fmt.Sscanf(line, "fps=%f", &progressData.FPS)
		} else if strings.HasPrefix(line, "bitrate=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				progressData.Bitrate = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "time=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				progressData.Time = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "speed=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				progressData.Speed = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "progress=") {
			// End of progress block
			if progressHandler != nil && progressData.Frame > 0 {
				progressHandler(progressData)
			}
			progressData = &Progress{}
		}
	}
}
