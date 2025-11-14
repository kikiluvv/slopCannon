package ffmpeg

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// Render performs a full video render with all specified options
func (e *Executor) Render(ctx context.Context, opts RenderOptions) error {
	if err := validateRenderOptions(opts); err != nil {
		return fmt.Errorf("invalid render options: %w", err)
	}

	e.logger.Info().
		Str("input", opts.Input).
		Str("output", opts.Output).
		Msg("starting render")

	args := []string{"-i", opts.Input}

	// Apply overlay if specified (requires second input)
	if opts.Overlay != nil {
		return fmt.Errorf("overlay must be applied using MergeWithOverlay() or via Filters field")
	}

	// Build filter chain
	filters := buildFilterChain(opts)
	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	// Video codec settings
	videoCodec := opts.VideoCodec
	if videoCodec == "" {
		videoCodec = DefaultVideoCodec
	}
	args = append(args, "-c:v", videoCodec)

	// Quality settings
	crf := opts.CRF
	if crf == 0 {
		crf = DefaultCRF
	}
	args = append(args, "-crf", fmt.Sprintf("%d", crf))

	// Preset
	preset := opts.Preset
	if preset == "" {
		preset = DefaultPreset
	}
	args = append(args, "-preset", preset)

	// Audio codec settings
	audioCodec := opts.AudioCodec
	if audioCodec == "" {
		audioCodec = DefaultAudioCodec
	}
	args = append(args, "-c:a", audioCodec)

	// FPS conversion
	if opts.FPS > 0 {
		args = append(args, "-r", fmt.Sprintf("%.2f", opts.FPS))
	}

	// Custom arguments
	if len(opts.CustomArgs) > 0 {
		args = append(args, opts.CustomArgs...)
	}

	// Output file
	args = append(args, opts.Output)

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: opts.ProgressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("render output")
		},
	}

	if err := e.Run(ctx, runOpts); err != nil {
		return fmt.Errorf("render failed: %w", err)
	}

	e.logger.Info().Str("output", opts.Output).Msg("render completed")
	return nil
}

// RenderClip is an alias for Render for consistent naming
func (e *Executor) RenderClip(ctx context.Context, opts RenderOptions) error {
	return e.Render(ctx, opts)
}

// MergeWithOverlay merges a video with an overlay using OverlayOptions
func (e *Executor) MergeWithOverlay(ctx context.Context, input, overlay, output string, overlayOpts OverlayOptions, progressFunc ProgressFunc) error {
	if input == "" {
		return fmt.Errorf("input path is required")
	}
	if overlay == "" {
		return fmt.Errorf("overlay path is required")
	}
	if output == "" {
		return fmt.Errorf("output path is required")
	}

	e.logger.Info().
		Str("input", input).
		Str("overlay", overlay).
		Str("output", output).
		Msg("merging with overlay")

	args := []string{
		"-i", input,
		"-i", overlay,
	}

	// Build overlay filter
	overlayFilter := fmt.Sprintf("overlay=%d:%d", overlayOpts.X, overlayOpts.Y)

	// Add opacity if specified
	if overlayOpts.Opacity > 0 && overlayOpts.Opacity < 1.0 {
		overlayFilter = fmt.Sprintf("[1]format=rgba,colorchannelmixer=aa=%.2f[ovr];[0][ovr]%s", overlayOpts.Opacity, overlayFilter)
	}

	// Add time constraints if specified
	if overlayOpts.Start > 0 || overlayOpts.End > 0 {
		if overlayOpts.Start > 0 {
			overlayFilter += fmt.Sprintf(":enable='gte(t,%.2f)", overlayOpts.Start.Seconds())
		}
		if overlayOpts.End > 0 {
			if overlayOpts.Start > 0 {
				overlayFilter += fmt.Sprintf("*lte(t,%.2f)", overlayOpts.End.Seconds())
			} else {
				overlayFilter += fmt.Sprintf(":enable='lte(t,%.2f)", overlayOpts.End.Seconds())
			}
			overlayFilter += "'"
		} else if overlayOpts.Start > 0 {
			overlayFilter += "'"
		}
	}

	args = append(args,
		"-filter_complex", overlayFilter,
		"-c:v", DefaultVideoCodec,
		"-crf", fmt.Sprintf("%d", DefaultCRF),
		"-preset", DefaultPreset,
		"-c:a", "copy",
		output,
	)

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("overlay output")
		},
	}

	if err := e.Run(ctx, runOpts); err != nil {
		return fmt.Errorf("overlay merge failed: %w", err)
	}

	e.logger.Info().Str("output", output).Msg("overlay merge completed")
	return nil
}

// ApplySubtitles burns subtitles into the video
func (e *Executor) ApplySubtitles(ctx context.Context, input, subtitles, output string, progressFunc ProgressFunc) error {
	if input == "" {
		return fmt.Errorf("input path is required")
	}
	if subtitles == "" {
		return fmt.Errorf("subtitles path is required")
	}
	if output == "" {
		return fmt.Errorf("output path is required")
	}

	e.logger.Info().
		Str("input", input).
		Str("subtitles", subtitles).
		Str("output", output).
		Msg("applying subtitles")

	// Escape the subtitle path for ffmpeg filter
	escapedPath := escapeSubtitlePath(subtitles)

	args := []string{
		"-i", input,
		"-vf", fmt.Sprintf("subtitles=%s", escapedPath),
		"-c:v", DefaultVideoCodec,
		"-crf", fmt.Sprintf("%d", DefaultCRF),
		"-preset", DefaultPreset,
		"-c:a", "copy",
		output,
	}

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("subtitle output")
		},
	}

	if err := e.Run(ctx, runOpts); err != nil {
		return fmt.Errorf("subtitle application failed: %w", err)
	}

	e.logger.Info().Str("output", output).Msg("subtitles applied")
	return nil
}

// RenderWithFilterBuilder renders using a FilterChain for complex operations
func (e *Executor) RenderWithFilterBuilder(ctx context.Context, input, output string, filterChain FilterChain, progressFunc ProgressFunc) error {
	if input == "" {
		return fmt.Errorf("input path is required")
	}
	if output == "" {
		return fmt.Errorf("output path is required")
	}
	if len(filterChain.Filters) == 0 {
		return fmt.Errorf("filter chain cannot be empty")
	}

	e.logger.Info().
		Str("input", input).
		Str("output", output).
		Int("filters", len(filterChain.Filters)).
		Msg("rendering with filter builder")

	args := []string{
		"-i", input,
		"-vf", strings.Join(filterChain.Filters, ","),
		"-c:v", DefaultVideoCodec,
		"-crf", fmt.Sprintf("%d", DefaultCRF),
		"-preset", DefaultPreset,
		"-c:a", DefaultAudioCodec,
		output,
	}

	runOpts := RunOptions{
		Args:            args,
		ProgressHandler: progressFunc,
		LogHandler: func(line string) {
			e.logger.Debug().Str("ffmpeg", line).Msg("filter builder output")
		},
	}

	if err := e.Run(ctx, runOpts); err != nil {
		return fmt.Errorf("filter builder render failed: %w", err)
	}

	e.logger.Info().Str("output", output).Msg("filter builder render completed")
	return nil
}

// validateRenderOptions validates the render options
func validateRenderOptions(opts RenderOptions) error {
	if opts.Input == "" {
		return fmt.Errorf("input path is required")
	}
	if opts.Output == "" {
		return fmt.Errorf("output path is required")
	}
	if opts.CRF < 0 || opts.CRF > 51 {
		if opts.CRF != 0 {
			return fmt.Errorf("CRF must be between 0 and 51")
		}
	}
	if opts.FPS < 0 {
		return fmt.Errorf("FPS cannot be negative")
	}
	return nil
}

// buildFilterChain constructs the filter chain from render options
func buildFilterChain(opts RenderOptions) []string {
	var filters []string

	// Scaling
	if opts.Width > 0 && opts.Height > 0 {
		filters = append(filters, fmt.Sprintf("scale=%d:%d", opts.Width, opts.Height))
	} else if opts.Scale != "" {
		filters = append(filters, fmt.Sprintf("scale=%s", opts.Scale))
	}

	// Subtitles
	if opts.Subtitles != "" {
		escapedPath := escapeSubtitlePath(opts.Subtitles)
		filters = append(filters, fmt.Sprintf("subtitles=%s", escapedPath))
	}

	// Custom filters
	filters = append(filters, opts.Filters...)

	return filters
}

// escapeSubtitlePath escapes the subtitle file path for ffmpeg filters
func escapeSubtitlePath(path string) string {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Windows: Convert backslashes to forward slashes
	if runtime.GOOS == "windows" {
		absPath = strings.ReplaceAll(absPath, "\\", "/")
		// Escape drive letter colon (C: -> C\:)
		if len(absPath) >= 2 && absPath[1] == ':' {
			absPath = absPath[0:1] + "\\:" + absPath[2:]
		}
	}

	// Escape special characters for ffmpeg filter
	escaped := strings.ReplaceAll(absPath, ":", "\\:")
	escaped = strings.ReplaceAll(escaped, "'", "\\'")

	return escaped
}
