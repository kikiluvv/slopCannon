package pipeline

import (
	"time"

	"github.com/keagan/slopcannon/internal/clips"
)

// Project represents a slopCannon project
type Project struct {
	Name      string
	InputPath string
	Clips     []*clips.Clip
	Timeline  *Timeline
	Metadata  map[string]interface{}
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Timeline holds the editing timeline
type Timeline struct {
	Clips    []*clips.Clip
	Overlays []Overlay
	SFX      []SoundEffect
}

// Overlay represents a video overlay
type Overlay struct {
	Type      string
	Path      string
	StartTime time.Duration
	EndTime   time.Duration
	Opacity   float64
	X         int
	Y         int
}

// SoundEffect represents a sound effect placement
type SoundEffect struct {
	Path      string
	Timestamp time.Duration
	Volume    float64
}

// AnalyzeOptions configures analysis behavior
type AnalyzeOptions struct {
	Model      string
	Overlay    string
	MinClipLen time.Duration
	MaxClips   int
	UseAI      bool
}

// RenderOptions configures render behavior
type RenderOptions struct {
	OutputPath string
	Format     string
	Quality    int // CRF value
	Preset     string
	Width      int
	Height     int
	FPS        float64
}

// Config holds pipeline-specific configuration
type Config struct {
	Workers     int
	ChunkSize   int
	EnableCache bool
	// New: where ONNX models live (directory with clip_image_encoder.onnx, virality_head.onnx)
	ModelPath string

	// Other per-pipeline knobs you might have
	MinClipLength time.Duration
	MaxClipLength time.Duration
}
