package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/keagan/slopcannon/pkg/util"
)

// DetectScenes finds scene changes in video using ffmpeg scene detection
func (e *Executor) DetectScenes(ctx context.Context, input string, threshold float64) ([]time.Duration, error) {
	e.logger.Info().
		Str("input", input).
		Float64("threshold", threshold).
		Msg("detecting scene changes")

	var stderrBuf bytes.Buffer
	var mu sync.Mutex

	opts := RunOptions{
		Args: []string{
			"-i", input,
			"-vf", fmt.Sprintf("select='gt(scene,%f)',showinfo", threshold),
			"-f", "null",
			"-",
		},
		LogHandler: func(line string) {
			mu.Lock()
			stderrBuf.WriteString(line + "\n")
			mu.Unlock()
			e.logger.Debug().Str("stderr", line).Msg("scene detection output")
		},
	}

	err := e.Run(ctx, opts)

	mu.Lock()
	output := stderrBuf.String()
	mu.Unlock()

	e.logger.Debug().Str("full_output", output).Msg("scene detection full stderr")

	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !strings.Contains(err.Error(), "Conversion failed") &&
			!strings.Contains(err.Error(), "Invalid return value") &&
			!strings.Contains(err.Error(), "Output file is empty") {
			return nil, fmt.Errorf("scene detection failed: %w", err)
		}
	}

	scenes := parseSceneOutput(output)
	e.logger.Info().Int("scenes", len(scenes)).Msg("scene detection complete")
	return scenes, nil
}

// parseSceneOutput extracts scene change timestamps from ffmpeg output
func parseSceneOutput(output string) []time.Duration {
	var scenes []time.Duration

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "pts_time:") {
			parts := strings.Split(line, "pts_time:")
			if len(parts) == 2 {
				timeStr := strings.Fields(strings.TrimSpace(parts[1]))[0]
				if seconds, err := strconv.ParseFloat(timeStr, 64); err == nil {
					scenes = append(scenes, time.Duration(seconds*float64(time.Second)))
				}
			}
		}
	}

	return scenes
}

// GenerateThumbnail creates a thumbnail image at a specific timestamp
func (e *Executor) GenerateThumbnail(ctx context.Context, input, output string, timestamp time.Duration, progressFunc ProgressFunc) error {
	if input == "" {
		return fmt.Errorf("input path is required")
	}
	if output == "" {
		return fmt.Errorf("output path is required")
	}

	e.logger.Info().
		Str("input", input).
		Str("output", output).
		Dur("timestamp", timestamp).
		Msg("generating thumbnail")

	args := []string{
		"-ss", util.FormatDuration(timestamp),
		"-i", input,
		"-vframes", "1",
		"-q:v", "2", // high quality JPEG
		output,
	}

	opts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("thumbnail generation")
		},
	}

	return e.Run(ctx, opts)
}

// GenerateThumbnails creates multiple thumbnails at specified intervals
func (e *Executor) GenerateThumbnails(ctx context.Context, input, outputPattern string, interval time.Duration, progressFunc ProgressFunc) error {
	if input == "" {
		return fmt.Errorf("input path is required")
	}
	if outputPattern == "" {
		return fmt.Errorf("output pattern is required")
	}

	e.logger.Info().
		Str("input", input).
		Str("pattern", outputPattern).
		Dur("interval", interval).
		Msg("generating thumbnails")

	args := []string{
		"-i", input,
		"-vf", fmt.Sprintf("fps=1/%d", int(interval.Seconds())),
		"-q:v", "2",
		outputPattern,
	}

	opts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("thumbnails generation")
		},
	}

	return e.Run(ctx, opts)
}
