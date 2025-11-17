package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/keagan/slopcannon/internal/ai"
	"github.com/keagan/slopcannon/internal/clips"
	"github.com/keagan/slopcannon/internal/config"
	"github.com/keagan/slopcannon/internal/ffmpeg"
	"github.com/rs/zerolog"
)

// Pipeline orchestrates the entire video processing workflow
type Pipeline struct {
	logger   zerolog.Logger
	config   *Config
	ffmpeg   *ffmpeg.Executor
	detector *ai.ClipDetector
}

// New creates a new pipeline instance
func New(logger zerolog.Logger, cfg *Config, appCfg *config.Config) (*Pipeline, error) {
	if cfg == nil {
		cfg = &Config{
			Workers:   4,
			ChunkSize: 10,
			ModelPath: appCfg.AI.ModelPath,
		}
	} else if cfg.ModelPath == "" {
		cfg.ModelPath = appCfg.AI.ModelPath
	}

	ffmpegExec, err := ffmpeg.New(logger, appCfg.FFmpeg.Threads)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ffmpeg: %w", err)
	}

	p := &Pipeline{
		logger: logger.With().Str("component", "pipeline").Logger(),
		config: cfg,
		ffmpeg: ffmpegExec,
		// detector will be created per detectClips call
	}

	return p, nil
}

// Close releases pipeline resources
func (p *Pipeline) Close() error {
	if p.detector != nil {
		return p.detector.Close()
	}
	return nil
}

// Analyze runs the full analysis pipeline on input video
func (p *Pipeline) Analyze(ctx context.Context, input string, opts AnalyzeOptions) (*Project, error) {
	p.logger.Info().
		Str("input", input).
		Str("model", opts.Model).
		Msg("starting analysis pipeline")

	// Validate input
	if input == "" {
		return nil, fmt.Errorf("input path cannot be empty")
	}

	// Stage 1: Extract video metadata
	videoInfo, err := p.ffmpeg.ProbeVideo(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	p.logger.Info().
		Dur("duration", videoInfo.Duration).
		Int("width", videoInfo.Width).
		Int("height", videoInfo.Height).
		Float64("fps", videoInfo.FPS).
		Msg("video metadata extracted")

	// Stage 2: AI-powered clip detection
	detectedClips, err := p.detectClips(ctx, input, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to detect clips: %w", err)
	}

	p.logger.Info().
		Int("clips_detected", len(detectedClips)).
		Msg("clip detection complete")

		// Clips are already scored and ranked by detector
	// Just limit to max clips if specified
	if opts.MaxClips > 0 && len(detectedClips) > opts.MaxClips {
		detectedClips = detectedClips[:opts.MaxClips]
	}

	// Stage 3: Create project
	project := &Project{
		Name:      fmt.Sprintf("project_%d", time.Now().Unix()),
		InputPath: input,
		Clips:     detectedClips,
		Timeline:  &Timeline{Clips: detectedClips},
		Metadata: map[string]interface{}{
			"duration":    videoInfo.Duration.Seconds(),
			"width":       videoInfo.Width,
			"height":      videoInfo.Height,
			"fps":         videoInfo.FPS,
			"video_codec": videoInfo.VideoCodec,
			"has_audio":   videoInfo.HasAudio,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	p.logger.Info().
		Str("project", project.Name).
		Int("clips", len(project.Clips)).
		Msg("analysis pipeline complete")

	return project, nil
}

// Render executes the rendering pipeline for a project
func (p *Pipeline) Render(ctx context.Context, project *Project, opts RenderOptions) (string, error) {
	// Validate project
	if project == nil {
		return "", fmt.Errorf("project cannot be nil")
	}

	p.logger.Info().
		Str("project", project.Name).
		Str("output", opts.OutputPath).
		Msg("starting render pipeline")
	if len(project.Clips) == 0 {
		return "", fmt.Errorf("project has no clips to render")
	}
	if opts.OutputPath == "" {
		return "", fmt.Errorf("output path cannot be empty")
	}

	// TODO: Implement render stages:
	// 1. Extract clips from source video
	// 2. Generate subtitles (if enabled)
	// 3. Apply overlays
	// 4. Concatenate clips
	// 5. Final render with effects

	p.logger.Info().
		Str("output", opts.OutputPath).
		Msg("render pipeline complete")

	return opts.OutputPath, nil
}

// detectClips performs AI-powered clip detection with composite scoring
func (p *Pipeline) detectClips(ctx context.Context, videoPath string, opts AnalyzeOptions) ([]*clips.Clip, error) {
	p.logger.Debug().Msg("detecting clips with AI")

	// Create detector config
	detectorCfg := ai.DefaultDetectorConfig()
	if opts.MinClipLen > 0 {
		detectorCfg.MinClipLength = opts.MinClipLen
	}
	if opts.MaxClips > 0 {
		detectorCfg.TopN = opts.MaxClips
	}

	// Build scorer based on model availability
	scorer := p.buildScorer()
	defer scorer.Close()

	// Create detector with custom scorer
	detector := ai.NewClipDetector(p.logger, p.ffmpeg, scorer, detectorCfg)
	defer detector.Close()

	return detector.Detect(ctx, videoPath)
}

// buildScorer creates appropriate scorer based on pipeline config.
func (p *Pipeline) buildScorer() ai.Scorer {
	// Always have heuristic scoring
	heuristic := ai.NewHeuristicScorer()
	aesthetic := ai.NewAestheticScorer(p.logger, p.ffmpeg)

	modelDir := p.config.ModelPath
	if modelDir == "" {
		// No model configured → heuristic + aesthetic only
		p.logger.Info().Msg("no model path configured; using heuristic + aesthetic scoring")
		return ai.NewCompositeScorer(
			[]ai.Scorer{heuristic, aesthetic},
			[]float64{0.6, 0.4},
		)
	}

	encoderPath := filepath.Join(modelDir, "clip_image_encoder.onnx")
	headPath := filepath.Join(modelDir, "virality_head.onnx")

	// Sanity check: files exist
	if _, err := os.Stat(encoderPath); err != nil {
		p.logger.Warn().Err(err).
			Str("encoder", encoderPath).
			Msg("encoder model not found; falling back to heuristic + aesthetic scoring")
		return ai.NewCompositeScorer(
			[]ai.Scorer{heuristic, aesthetic},
			[]float64{0.6, 0.4},
		)
	}
	if _, err := os.Stat(headPath); err != nil {
		p.logger.Warn().Err(err).
			Str("head", headPath).
			Msg("virality head model not found; falling back to heuristic + aesthetic scoring")
		return ai.NewCompositeScorer(
			[]ai.Scorer{heuristic, aesthetic},
			[]float64{0.6, 0.4},
		)
	}

	clipScorer, err := ai.NewCLIPScorer(p.logger, p.ffmpeg, encoderPath, headPath)
	if err != nil {
		p.logger.Warn().Err(err).
			Str("encoder", encoderPath).
			Str("head", headPath).
			Msg("failed to initialize CLIP scorer; using heuristic + aesthetic scoring")
		return ai.NewCompositeScorer(
			[]ai.Scorer{heuristic, aesthetic},
			[]float64{0.6, 0.4},
		)
	}

	p.logger.Info().
		Str("encoder_model", encoderPath).
		Str("head_model", headPath).
		Msg("using heuristic + aesthetic + CLIP scoring")

	return ai.NewCompositeScorer(
		[]ai.Scorer{heuristic, aesthetic, clipScorer},
		[]float64{0.3, 0.2, 0.5}, // adjust weights as you like
	)
}
