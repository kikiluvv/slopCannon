package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/keagan/slopcannon/internal/clips"
	"github.com/keagan/slopcannon/internal/ffmpeg"
	"github.com/rs/zerolog"
)

// DetectorConfig configures clip detection behavior
type DetectorConfig struct {
	MinClipLength      time.Duration
	MaxClipLength      time.Duration
	SceneThreshold     float64
	SilenceThreshold   float64
	MinSilenceDuration float64
	OverlapSeconds     float64
	TopN               int
}

func DefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		MinClipLength:      10 * time.Second,
		MaxClipLength:      90 * time.Second,
		SceneThreshold:     0.4,
		SilenceThreshold:   -30.0,
		MinSilenceDuration: 1.0,
		OverlapSeconds:     2.0,
		TopN:               10,
	}
}

// ClipDetector finds viral-worthy clips
type ClipDetector struct {
	logger    zerolog.Logger
	ffmpeg    *ffmpeg.Executor
	scorer    Scorer
	extractor *FeatureExtractor
	config    DetectorConfig
}

// NewClipDetector creates a detector with a custom scorer
func NewClipDetector(logger zerolog.Logger, exec *ffmpeg.Executor, scorer Scorer, cfg DetectorConfig) *ClipDetector {
	return &ClipDetector{
		logger:    logger.With().Str("component", "clip-detector").Logger(),
		ffmpeg:    exec,
		scorer:    scorer,
		extractor: NewFeatureExtractor(exec),
		config:    cfg,
	}
}

// NewDefaultClipDetector creates a detector with heuristic scoring
func NewDefaultClipDetector(logger zerolog.Logger, exec *ffmpeg.Executor, cfg DetectorConfig) *ClipDetector {
	return NewClipDetector(logger, exec, NewHeuristicScorer(), cfg)
}

// Detect finds and scores clips
func (d *ClipDetector) Detect(ctx context.Context, videoPath string) ([]*clips.Clip, error) {
	d.logger.Info().Str("video", videoPath).Msg("starting clip detection")

	// Step 1: Probe video
	info, err := d.ffmpeg.ProbeVideo(ctx, videoPath)
	if err != nil {
		return nil, fmt.Errorf("probe failed: %w", err)
	}

	// Step 2: Detect scene changes
	scenes, err := d.ffmpeg.DetectScenes(ctx, videoPath, d.config.SceneThreshold)
	if err != nil {
		return nil, fmt.Errorf("scene detection failed: %w", err)
	}

	// Step 3: Detect silence periods
	silences, err := d.ffmpeg.DetectSilence(ctx, videoPath,
		d.config.SilenceThreshold, d.config.MinSilenceDuration)
	if err != nil {
		return nil, fmt.Errorf("silence detection failed: %w", err)
	}

	// Step 4: Analyze volume
	volumeStats, err := d.ffmpeg.AnalyzeVolume(ctx, videoPath)
	if err != nil {
		return nil, fmt.Errorf("volume analysis failed: %w", err)
	}

	// Step 5: Generate candidate clips
	candidates := d.generateCandidates(scenes, silences, info.Duration)

	// Step 6: Score each candidate using the Scorer interface
	scoredClips := make([]*clips.Clip, 0, len(candidates))
	for i, candidate := range candidates {
		features := d.extractFeatures(candidate, scenes, silences, volumeStats)

		clip := &clips.Clip{
			ID:        fmt.Sprintf("clip_%d", i),
			Start:     candidate.Start,
			End:       candidate.End,
			Duration:  candidate.End - candidate.Start,
			SourceURL: videoPath,
			Metadata: map[string]interface{}{
				"scene_changes":  features.SceneChangeCount,
				"silence_ratio":  features.SilenceRatio,
				"peak_volume":    features.PeakVolume,
				"mean_volume":    features.MeanVolume,
				"audio_dynamics": features.AudioDynamics,
			},
		}

		// Use the scorer interface
		score, err := d.scorer.Score(ctx, clip)
		if err != nil {
			d.logger.Warn().Err(err).Str("clip_id", clip.ID).Msg("scoring failed, using 0")
			score = 0.0
		}
		clip.Score = score

		d.logger.Debug().
			Str("clip", clip.ID).
			Float64("score_total", clip.Score).
			Float64("score_clip", clip.Metadata["clip_score"].(float64)).
			Msg("ranked clip")

		scoredClips = append(scoredClips, clip)
	}

	// Step 7: Sort and return top N
	topClips := d.rankAndFilter(scoredClips)

	d.logger.Info().
		Int("candidates", len(candidates)).
		Int("top_clips", len(topClips)).
		Msg("clip detection complete")

	return topClips, nil
}

// Close releases scorer resources
func (d *ClipDetector) Close() error {
	return d.scorer.Close()
}

// candidateSegment represents a potential clip
type candidateSegment struct {
	Start time.Duration
	End   time.Duration
}

// generateCandidates creates candidate clips from scene boundaries
func (d *ClipDetector) generateCandidates(scenes []time.Duration, silences []ffmpeg.SilenceSegment, totalDuration time.Duration) []candidateSegment {
	var candidates []candidateSegment

	// Start from beginning
	lastBoundary := time.Duration(0)

	for _, sceneTime := range scenes {
		// Check if segment is long enough
		if sceneTime-lastBoundary >= d.config.MinClipLength {
			candidates = append(candidates, candidateSegment{
				Start: lastBoundary,
				End:   sceneTime,
			})
		}
		lastBoundary = sceneTime
	}

	// Add final segment
	if totalDuration-lastBoundary >= d.config.MinClipLength {
		candidates = append(candidates, candidateSegment{
			Start: lastBoundary,
			End:   totalDuration,
		})
	}

	// Merge adjacent short segments
	return d.mergeShortSegments(candidates)
}

func (d *ClipDetector) mergeShortSegments(segments []candidateSegment) []candidateSegment {
	merged := make([]candidateSegment, 0)

	for i := 0; i < len(segments); i++ {
		current := segments[i]

		// If too long, split it
		if current.End-current.Start > d.config.MaxClipLength {
			// Split into smaller chunks
			splitPoints := int((current.End - current.Start) / d.config.MaxClipLength)
			chunkSize := (current.End - current.Start) / time.Duration(splitPoints+1)

			for j := 0; j <= splitPoints; j++ {
				start := current.Start + time.Duration(j)*chunkSize
				end := start + chunkSize
				if end > current.End {
					end = current.End
				}
				merged = append(merged, candidateSegment{Start: start, End: end})
			}
		} else {
			merged = append(merged, current)
		}
	}

	return merged
}

// extractFeatures calculates features for a clip candidate
func (d *ClipDetector) extractFeatures(segment candidateSegment, scenes []time.Duration, silences []ffmpeg.SilenceSegment, volumeStats *ffmpeg.VolumeStats) ClipFeatures {
	// Count scene changes in this segment
	sceneCount := 0
	for _, scene := range scenes {
		if scene >= segment.Start && scene <= segment.End {
			sceneCount++
		}
	}

	// Calculate silence ratio
	silenceDuration := time.Duration(0)
	for _, silence := range silences {
		silStart := time.Duration(silence.Start * float64(time.Second))
		silEnd := time.Duration(silence.End * float64(time.Second))

		if silStart >= segment.Start && silEnd <= segment.End {
			silenceDuration += silEnd - silStart
		}
	}

	clipDuration := segment.End - segment.Start
	silenceRatio := 0.0
	if clipDuration > 0 {
		silenceRatio = float64(silenceDuration) / float64(clipDuration)
	}

	return ClipFeatures{
		Duration:         clipDuration,
		SceneChangeCount: sceneCount,
		SilenceRatio:     silenceRatio,
		MeanVolume:       volumeStats.MeanVolume,
		PeakVolume:       volumeStats.MaxVolume,
		AudioDynamics:    volumeStats.MaxVolume - volumeStats.MeanVolume,
	}
}

// rankAndFilter sorts clips by score and returns top N
func (d *ClipDetector) rankAndFilter(clips []*clips.Clip) []*clips.Clip {
	// Sort by score descending
	for i := 0; i < len(clips); i++ {
		for j := i + 1; j < len(clips); j++ {
			if clips[j].Score > clips[i].Score {
				clips[i], clips[j] = clips[j], clips[i]
			}
		}
	}

	// Return top N
	if len(clips) > d.config.TopN {
		return clips[:d.config.TopN]
	}

	return clips
}
