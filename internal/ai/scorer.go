package ai

import (
	"context"
	"math"

	"github.com/keagan/slopcannon/internal/clips"
)

// Scorer evaluates clips for viral potential
type Scorer interface {
	Score(ctx context.Context, clip *clips.Clip) (float64, error)
	Close() error
}

// HeuristicScorer uses rule-based heuristics
type HeuristicScorer struct {
	weights Weights
}

// Weights for different heuristic factors
type Weights struct {
	Duration      float64
	ShotChanges   float64
	AudioPeaks    float64
	DialogDensity float64
}

// NewHeuristicScorer creates a new heuristic scorer
func NewHeuristicScorer() *HeuristicScorer {
	return &HeuristicScorer{
		weights: Weights{
			Duration:      0.2,
			ShotChanges:   0.3,
			AudioPeaks:    0.3,
			DialogDensity: 0.2,
		},
	}
}

// Score calculates a heuristic score
func (h *HeuristicScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
	var totalScore float64

	// Duration scoring (optimal 15-60s for viral clips)
	durationScore := h.scoreDuration(clip.Duration.Seconds())
	totalScore += h.weights.Duration * durationScore

	// Shot changes scoring (from metadata)
	if sceneChanges, ok := clip.Metadata["scene_changes"].(int); ok {
		shotScore := h.scoreShotChanges(sceneChanges, clip.Duration.Seconds())
		totalScore += h.weights.ShotChanges * shotScore
	}

	// Audio peaks scoring (from metadata)
	if peakVolume, ok := clip.Metadata["peak_volume"].(float64); ok {
		audioScore := h.scoreAudioPeaks(peakVolume)
		totalScore += h.weights.AudioPeaks * audioScore
	}

	// Dialog density scoring (inverse of silence ratio)
	if silenceRatio, ok := clip.Metadata["silence_ratio"].(float64); ok {
		dialogScore := 1.0 - math.Min(1.0, silenceRatio)
		totalScore += h.weights.DialogDensity * dialogScore
	}

	return math.Max(0.0, math.Min(1.0, totalScore)), nil
}

// scoreDuration uses a bell curve around optimal length
func (h *HeuristicScorer) scoreDuration(seconds float64) float64 {
	// Optimal viral clip: 30 seconds
	// Acceptable range: 15-60 seconds
	optimal := 30.0
	return math.Exp(-math.Pow(seconds-optimal, 2) / 400.0)
}

// scoreShotChanges normalizes scene changes per second
func (h *HeuristicScorer) scoreShotChanges(changes int, durationSeconds float64) float64 {
	if durationSeconds == 0 {
		return 0
	}
	// Optimal: 1 scene change per 3-5 seconds
	changesPerSecond := float64(changes) / durationSeconds
	optimal := 0.25 // ~1 change per 4 seconds

	// Bell curve around optimal
	return math.Exp(-math.Pow(changesPerSecond-optimal, 2) / 0.05)
}

// scoreAudioPeaks normalizes volume peaks (dB)
func (h *HeuristicScorer) scoreAudioPeaks(peakDB float64) float64 {
	// Normalize from typical dB range (-60 to 0)
	// Higher peaks = more engaging
	normalized := (peakDB + 60.0) / 60.0
	return math.Max(0.0, math.Min(1.0, normalized))
}

// Close is a no-op for heuristic scorer
func (h *HeuristicScorer) Close() error {
	return nil
}

// ModelScorer uses AI models for scoring
type ModelScorer struct {
	modelPath string
	// TODO: add model-specific fields
}

// NewModelScorer creates a new AI model scorer
func NewModelScorer(modelPath string) (*ModelScorer, error) {
	// TODO: load model
	return &ModelScorer{
		modelPath: modelPath,
	}, nil
}

// Score calculates an AI-based score
func (m *ModelScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
	// TODO: implement model inference
	return 0.0, nil
}

// Close releases model resources
func (m *ModelScorer) Close() error {
	// TODO: cleanup model
	return nil
}

// CompositeScorer combines multiple scorers
type CompositeScorer struct {
	scorers []Scorer
	weights []float64
}

// NewCompositeScorer creates a scorer that combines multiple scorers
func NewCompositeScorer(scorers []Scorer, weights []float64) *CompositeScorer {
	return &CompositeScorer{
		scorers: scorers,
		weights: weights,
	}
}

// Score calculates a weighted average of all scorers
func (c *CompositeScorer) Score(ctx context.Context, clip *clips.Clip) (float64, error) {
	if len(c.scorers) == 0 {
		return 0.0, nil
	}

	var totalScore float64
	var totalWeight float64

	for i, scorer := range c.scorers {
		score, err := scorer.Score(ctx, clip)
		if err != nil {
			return 0.0, err
		}

		weight := 1.0
		if i < len(c.weights) {
			weight = c.weights[i]
		}

		totalScore += score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0, nil
	}

	return totalScore / totalWeight, nil
}

// Close closes all underlying scorers
func (c *CompositeScorer) Close() error {
	for _, scorer := range c.scorers {
		if err := scorer.Close(); err != nil {
			return err
		}
	}
	return nil
}
