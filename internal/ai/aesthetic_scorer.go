package ai

import (
	"context"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/keagan/slopcannon/internal/clips"
	"github.com/keagan/slopcannon/internal/ffmpeg"
	"github.com/rs/zerolog"
)

// AestheticScorer uses simple image analysis heuristics
type AestheticScorer struct {
	logger zerolog.Logger
	ffmpeg *ffmpeg.Executor
}

// NewAestheticScorer creates a lightweight image-based scorer
func NewAestheticScorer(logger zerolog.Logger, exec *ffmpeg.Executor) *AestheticScorer {
	return &AestheticScorer{
		logger: logger.With().Str("scorer", "aesthetic").Logger(),
		ffmpeg: exec,
	}
}

// Score analyzes visual aesthetics of clip keyframe
func (a *AestheticScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
	// Extract keyframe from middle of clip
	keyframeTime := clip.Start + (clip.Duration / 2)
	keyframePath := filepath.Join(os.TempDir(), fmt.Sprintf("keyframe_%s_%d.jpg", clip.ID, time.Now().UnixNano()))
	defer os.Remove(keyframePath)

	err := a.ffmpeg.ExtractFrame(ctx, clip.SourceURL, keyframeTime, keyframePath)
	if err != nil {
		a.logger.Warn().Err(err).Str("clip", clip.ID).Msg("keyframe extraction failed")
		return 0.0, err
	}

	// Load image
	file, err := os.Open(keyframePath)
	if err != nil {
		return 0.0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0.0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Calculate aesthetic metrics
	colorfulness := a.calculateColorfulness(img)
	contrast := a.calculateContrast(img)
	brightness := a.calculateBrightness(img)

	// Weighted combination
	score := (0.4 * colorfulness) + (0.3 * contrast) + (0.3 * brightness)

	a.logger.Debug().
		Str("clip", clip.ID).
		Float64("colorfulness", colorfulness).
		Float64("contrast", contrast).
		Float64("brightness", brightness).
		Float64("score", score).
		Msg("aesthetic scoring complete")

	return math.Max(0, math.Min(1, score)), nil
}

// calculateColorfulness measures color variance
func (a *AestheticScorer) calculateColorfulness(img image.Image) float64 {
	bounds := img.Bounds()
	var rSum, gSum, bSum float64
	pixels := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rSum += float64(r >> 8)
			gSum += float64(g >> 8)
			bSum += float64(b >> 8)
		}
	}

	rMean := rSum / pixels
	gMean := gSum / pixels
	bMean := bSum / pixels

	// Higher RGB variance = more colorful
	variance := math.Abs(rMean-gMean) + math.Abs(gMean-bMean) + math.Abs(bMean-rMean)
	return math.Min(1.0, variance/255.0)
}

// calculateContrast measures luminance variance
func (a *AestheticScorer) calculateContrast(img image.Image) float64 {
	bounds := img.Bounds()
	var lumSum, lumSqSum float64
	pixels := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			// Luminance formula
			lum := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			lumSum += lum
			lumSqSum += lum * lum
		}
	}

	mean := lumSum / pixels
	variance := (lumSqSum / pixels) - (mean * mean)
	stdDev := math.Sqrt(variance)

	// Normalize to 0-1 (typical stddev 0-60)
	return math.Min(1.0, stdDev/60.0)
}

// calculateBrightness measures average luminance
func (a *AestheticScorer) calculateBrightness(img image.Image) float64 {
	bounds := img.Bounds()
	var lumSum float64
	pixels := float64(bounds.Dx() * bounds.Dy())

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			lum := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			lumSum += lum
		}
	}

	avgLum := lumSum / pixels
	// Prefer moderate brightness (not too dark, not blown out)
	// Optimal around 128
	deviation := math.Abs(avgLum - 128.0)
	return 1.0 - math.Min(1.0, deviation/128.0)
}

// Close is a no-op for aesthetic scorer
func (a *AestheticScorer) Close() error {
	return nil
}
