package ai

import (
	"time"

	"github.com/keagan/slopcannon/internal/ffmpeg"
)

// ClipFeatures represents extracted features for scoring
type ClipFeatures struct {
	Duration         time.Duration
	SceneChangeCount int
	SilenceRatio     float64
	MeanVolume       float64
	PeakVolume       float64
	MotionIntensity  float64 // derived from scene changes
	AudioDynamics    float64 // peak - mean volume
}

// FeatureExtractor pulls features from video segments
type FeatureExtractor struct {
	ffmpeg *ffmpeg.Executor
}

func NewFeatureExtractor(exec *ffmpeg.Executor) *FeatureExtractor {
	return &FeatureExtractor{ffmpeg: exec}
}
