package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// ConcatOptions defines concatenation parameters
type ConcatOptions struct {
	Inputs       []string
	Output       string
	ReEncode     bool
	VideoCodec   string
	AudioCodec   string
	CRF          int
	ProgressFunc ProgressFunc
}

// Concat merges multiple video files into one
func (e *Executor) Concat(ctx context.Context, opts ConcatOptions) error {
	if len(opts.Inputs) == 0 {
		return fmt.Errorf("no input files provided")
	}
	if opts.Output == "" {
		return fmt.Errorf("output path is required")
	}

	e.logger.Info().
		Int("inputs", len(opts.Inputs)).
		Str("output", opts.Output).
		Msg("concatenating videos")

	// Create temporary concat file list
	concatFile, err := e.createConcatFile(opts.Inputs)
	if err != nil {
		return fmt.Errorf("failed to create concat file: %w", err)
	}
	defer os.Remove(concatFile)

	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", concatFile,
	}

	if opts.ReEncode {
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
	} else {
		args = append(args, "-c", "copy")
	}

	args = append(args, opts.Output)

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: opts.ProgressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("concatenating")
		},
	}

	return e.Run(ctx, runOpts)
}

// createConcatFile generates a temporary file list for ffmpeg concat
func (e *Executor) createConcatFile(inputs []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "slopcannon-concat-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	for _, input := range inputs {
		absPath, err := filepath.Abs(input)
		if err != nil {
			return "", err
		}
		if _, err := fmt.Fprintf(tmpFile, "file '%s'\n", absPath); err != nil {
			return "", err
		}
	}

	return tmpFile.Name(), nil
}
