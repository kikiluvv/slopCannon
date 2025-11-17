package ffmpeg_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/keagan/slopcannon/internal/config"
	"github.com/keagan/slopcannon/internal/ffmpeg"
	"github.com/keagan/slopcannon/internal/pipeline"
	"github.com/rs/zerolog"
)

// local helper (cannot use unexported ones from ffmpeg package)
func skipIfNoFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found in PATH - install with: brew install ffmpeg")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found in PATH - install with: brew install ffmpeg")
	}
}

func TestIntegration_FFmpegAndAIScoring(t *testing.T) {
	skipIfNoFFmpeg(t)

	// testdata path from repo root: ../.. from internal/ffmpeg
	testVideoPath := filepath.Join("..", "..", "testdata", "test.mp4")
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		t.Skipf("test video not found at %s", testVideoPath)
	}

	// CLIP model path (same as CLI/config)
	modelPath := filepath.Join("..", "..", "models", "clip-vit-base.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("CLIP model not found at %s (download sayantan47/clip-vit-b32-onnx first)", modelPath)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}).With().Str("test", "integration_ffmpeg_ai").Logger()

	// Minimal config
	appCfg := &config.Config{}
	appCfg.Concurrency = 2
	appCfg.AI.ModelPath = modelPath
	appCfg.AI.UseModel = true

	// Ensure ffmpeg executor will be constructed correctly by pipeline.New
	pipeCfg := &pipeline.Config{
		Workers:     appCfg.Concurrency,
		EnableCache: false,
	}

	p, err := pipeline.New(logger, pipeCfg, appCfg)
	if err != nil {
		t.Fatalf("failed to create pipeline: %v", err)
	}
	defer p.Close()

	ctx := context.Background()
	opts := pipeline.AnalyzeOptions{
		MinClipLen: 5 * time.Second,
		MaxClips:   5,
		Model:      modelPath,
	}

	start := time.Now()
	project, err := p.Analyze(ctx, testVideoPath, opts)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(project.Clips) == 0 {
		t.Fatalf("expected at least one clip, got 0")
	}

	t.Logf("AI+FFmpeg integration analyze completed: project=%s clips=%d (in %v)",
		project.Name, len(project.Clips), elapsed)

	// Per‑clip scoring info
	for i, c := range project.Clips {
		var clipScore float64
		if v, ok := c.Metadata["clip_score"]; ok {
			if f, ok2 := v.(float64); ok2 {
				clipScore = f
			}
		}
		logger.Info().
			Int("idx", i).
			Str("clip_id", c.ID).
			Dur("start", c.Start).
			Dur("end", c.End).
			Float64("score_total", c.Score).
			Float64("score_clip", clipScore).
			Msg("integration ranked clip")
	}

	// Basic assertion: CLIP model was actually loaded & used
	if len(project.Clips) > 0 {
		if _, ok := project.Clips[0].Metadata["clip_score"]; !ok {
			t.Fatalf("expected clip_score in metadata for first clip")
		}
	}

	// Dummy reference to ffmpeg package to ensure it’s linked (avoid unused import).
	_ = ffmpeg.NewFilterBuilder()
}
