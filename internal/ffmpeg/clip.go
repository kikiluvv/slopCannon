package ffmpeg

import (
	"context"
	"fmt"
	"time"

	"github.com/keagan/slopcannon/pkg/util"
)

// ClipOptions defines clip extraction parameters
type ClipOptions struct {
	Start        time.Duration
	End          time.Duration
	Output       string
	CopyCodec    bool // If true, use -c copy for fast extraction
	VideoCodec   string
	AudioCodec   string
	CRF          int // Quality (0-51, lower = better)
	ProgressFunc ProgressFunc
}

// ExtractClip cuts a segment from a video
func (e *Executor) ExtractClip(ctx context.Context, input string, opts ClipOptions) error {
	duration := opts.End - opts.Start
	if duration <= 0 {
		return fmt.Errorf("invalid clip duration: end must be after start")
	}

	e.logger.Info().
		Str("input", input).
		Str("output", opts.Output).
		Dur("start", opts.Start).
		Dur("duration", duration).
		Bool("copy_codec", opts.CopyCodec).
		Msg("extracting clip")

	args := []string{
		"-i", input,
		"-ss", util.FormatDuration(opts.Start),
		"-t", util.FormatDuration(duration),
	}

	if opts.CopyCodec {
		args = append(args, "-c", "copy")
	} else {
		codec := opts.VideoCodec
		if codec == "" {
			codec = DefaultVideoCodec
		}
		args = append(args, "-c:v", codec)

		audioCodec := opts.AudioCodec
		if audioCodec == "" {
			audioCodec = DefaultAudioCodec
		}
		args = append(args, "-c:a", audioCodec)

		crf := opts.CRF
		if crf == 0 {
			crf = DefaultCRF
		}
		args = append(args, "-crf", fmt.Sprintf("%d", crf))
	}

	args = append(args, opts.Output)

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: opts.ProgressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("clip extraction")
		},
	}

	if err := e.Run(ctx, runOpts); err != nil {
		return fmt.Errorf("clip extraction failed: %w", err)
	}

	e.logger.Info().Str("output", opts.Output).Msg("clip extraction complete")
	return nil
}

// TrimOptions defines trimming parameters for in-place editing
type TrimOptions struct {
	Start        time.Duration
	End          time.Duration
	Output       string
	ProgressFunc ProgressFunc
}

// Trim creates a trimmed copy with re-encoding for precision
func (e *Executor) Trim(ctx context.Context, input string, opts TrimOptions) error {
	return e.ExtractClip(ctx, input, ClipOptions{
		Start:        opts.Start,
		End:          opts.End,
		Output:       opts.Output,
		CopyCodec:    false,
		ProgressFunc: opts.ProgressFunc,
	})
}
