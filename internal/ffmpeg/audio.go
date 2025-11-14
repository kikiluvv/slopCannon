package ffmpeg

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// AudioFormat defines audio extraction format options
type AudioFormat struct {
	Codec      string
	SampleRate int
	Channels   int
	Bitrate    string
}

// DefaultWhisperFormat returns optimal format for Whisper transcription
func DefaultWhisperFormat() AudioFormat {
	return AudioFormat{
		Codec:      "pcm_s16le",
		SampleRate: 16000,
		Channels:   1, // mono
		Bitrate:    "",
	}
}

// ExtractAudio extracts audio stream to a separate file
func (e *Executor) ExtractAudio(ctx context.Context, input, output string, format AudioFormat, progressFunc ProgressFunc) error {
	e.logger.Info().
		Str("input", input).
		Str("output", output).
		Str("codec", format.Codec).
		Int("sample_rate", format.SampleRate).
		Msg("extracting audio")

	args := []string{
		"-i", input,
		"-vn", // no video
		"-acodec", format.Codec,
		"-ar", fmt.Sprintf("%d", format.SampleRate),
		"-ac", fmt.Sprintf("%d", format.Channels),
	}

	if format.Bitrate != "" {
		args = append(args, "-b:a", format.Bitrate)
	}

	args = append(args, output)

	opts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("audio extraction")
		},
	}

	return e.Run(ctx, opts)
}

// SilenceSegment represents a period of silence in audio
type SilenceSegment struct {
	Start    float64
	End      float64
	Duration float64
}

// DetectSilence finds silence segments in audio/video file
func (e *Executor) DetectSilence(ctx context.Context, input string, noiseThreshold float64, minDuration float64) ([]SilenceSegment, error) {
	e.logger.Info().
		Str("input", input).
		Float64("noise_threshold", noiseThreshold).
		Float64("min_duration", minDuration).
		Msg("detecting silence")

	var stderrBuf bytes.Buffer
	var mu sync.Mutex

	opts := RunOptions{
		Args: []string{
			"-i", input,
			"-af", fmt.Sprintf("silencedetect=noise=%.6fdB:d=%.6f", noiseThreshold, minDuration),
			"-f", "null",
			"-",
		},
		LogHandler: func(line string) {
			mu.Lock()
			stderrBuf.WriteString(line + "\n")
			mu.Unlock()
			// Also log it for debugging
			e.logger.Debug().Str("stderr", line).Msg("silence detection output")
		},
	}

	err := e.Run(ctx, opts)

	mu.Lock()
	output := stderrBuf.String()
	mu.Unlock()

	// Log the full output for debugging
	e.logger.Debug().Str("full_output", output).Msg("silence detection full stderr")

	if err != nil {
		// Check if it's a context cancellation - propagate that
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		// Only ignore the specific null output errors
		if !strings.Contains(err.Error(), "Conversion failed") &&
			!strings.Contains(err.Error(), "Invalid return value") &&
			!strings.Contains(err.Error(), "Output file is empty") {
			return nil, fmt.Errorf("silence detection failed: %w", err)
		}
	}

	if output == "" {
		return nil, fmt.Errorf("silence detection produced no output")
	}

	return parseSilenceOutput(output), nil
}

// parseSilenceOutput extracts silence segments from ffmpeg output
func parseSilenceOutput(output string) []SilenceSegment {
	var segments []SilenceSegment
	var currentStart float64

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "silence_start:") {
			parts := strings.Split(line, "silence_start:")
			if len(parts) == 2 {
				currentStart, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			}
		} else if strings.Contains(line, "silence_end:") {
			parts := strings.Split(line, "silence_end:")
			if len(parts) == 2 {
				endStr := strings.Fields(strings.TrimSpace(parts[1]))[0]
				end, _ := strconv.ParseFloat(endStr, 64)

				var duration float64
				if strings.Contains(line, "silence_duration:") {
					durParts := strings.Split(line, "silence_duration:")
					if len(durParts) == 2 {
						duration, _ = strconv.ParseFloat(strings.TrimSpace(durParts[1]), 64)
					}
				} else {
					duration = end - currentStart
				}

				segments = append(segments, SilenceSegment{
					Start:    currentStart,
					End:      end,
					Duration: duration,
				})
			}
		}
	}

	return segments
}

// VolumeStats holds volume analysis results
type VolumeStats struct {
	MeanVolume float64
	MaxVolume  float64
}

// AnalyzeVolume calculates volume statistics for audio/video file
func (e *Executor) AnalyzeVolume(ctx context.Context, input string) (*VolumeStats, error) {
	e.logger.Info().Str("input", input).Msg("analyzing volume")

	var stderrBuf bytes.Buffer
	var mu sync.Mutex

	opts := RunOptions{
		Args: []string{
			"-i", input,
			"-af", "volumedetect",
			"-f", "null",
			"-",
		},
		LogHandler: func(line string) {
			mu.Lock()
			stderrBuf.WriteString(line + "\n")
			mu.Unlock()
			e.logger.Debug().Str("stderr", line).Msg("volume detection output")
		},
	}

	err := e.Run(ctx, opts)

	mu.Lock()
	output := stderrBuf.String()
	mu.Unlock()

	e.logger.Debug().Str("full_output", output).Msg("volume detection full stderr")

	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !strings.Contains(err.Error(), "Conversion failed") &&
			!strings.Contains(err.Error(), "Invalid return value") &&
			!strings.Contains(err.Error(), "Output file is empty") {
			return nil, fmt.Errorf("volume analysis failed: %w", err)
		}
	}

	if output == "" {
		return nil, fmt.Errorf("volume analysis produced no output")
	}

	return e.parseVolumeOutput(output)
}

// parseVolumeOutput extracts volume stats from ffmpeg output
func (e *Executor) parseVolumeOutput(output string) (*VolumeStats, error) {
	stats := &VolumeStats{}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "mean_volume:") {
			parts := strings.Split(line, "mean_volume:")
			if len(parts) == 2 {
				valStr := strings.Fields(strings.TrimSpace(parts[1]))[0]
				stats.MeanVolume, _ = strconv.ParseFloat(valStr, 64)
			}
		} else if strings.Contains(line, "max_volume:") {
			parts := strings.Split(line, "max_volume:")
			if len(parts) == 2 {
				valStr := strings.Fields(strings.TrimSpace(parts[1]))[0]
				stats.MaxVolume, _ = strconv.ParseFloat(valStr, 64)
			}
		}
	}

	return stats, nil
}

// NormalizeAudio applies audio normalization to a file
func (e *Executor) NormalizeAudio(ctx context.Context, input, output string, targetLevel float64, progressFunc ProgressFunc) error {
	e.logger.Info().
		Str("input", input).
		Str("output", output).
		Float64("target_level", targetLevel).
		Msg("normalizing audio")

	filter := fmt.Sprintf("loudnorm=I=%f:TP=-1.5:LRA=11", targetLevel)

	args := []string{
		"-i", input,
		"-af", filter,
		"-c:v", "copy", // copy video stream
		output,
	}

	opts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("audio normalization")
		},
	}

	return e.Run(ctx, opts)
}
