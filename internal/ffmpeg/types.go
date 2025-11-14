package ffmpeg

import "time"

// VideoInfo contains metadata about a video file
type VideoInfo struct {
	FilePath     string
	Duration     time.Duration
	Width        int
	Height       int
	FPS          float64
	Bitrate      int64
	VideoCodec   string
	HasAudio     bool
	AudioCodec   string
	AudioBitrate int64
}

// OverlayOptions configures overlay compositing
type OverlayOptions struct {
	X       int
	Y       int
	Opacity float64
	Start   time.Duration
	End     time.Duration
}

// Progress represents ffmpeg progress data
type Progress struct {
	Frame      int
	FPS        float64
	Bitrate    string
	Time       string
	Speed      string
	Percentage float64
}

// RunOptions configures ffmpeg execution
type RunOptions struct {
	Args            []string
	ProgressHandler func(*Progress)
	LogHandler      func(line string)
}

// Default encoding settings
const (
	DefaultCRF        = 23
	DefaultPreset     = "medium"
	DefaultVideoCodec = "libx264"
	DefaultAudioCodec = "aac"
)

// RenderOptions configures video rendering operations
type RenderOptions struct {
	Input        string
	Output       string
	Overlay      *OverlayOptions
	Subtitles    string
	Filters      []string
	VideoCodec   string
	AudioCodec   string
	CRF          int
	Preset       string
	Width        int
	Height       int
	FPS          float64
	Scale        string
	ProgressFunc ProgressFunc
	CustomArgs   []string
}

// ProgressFunc is a callback for progress updates during ffmpeg operations.
// Called periodically with progress information as the operation executes.
type ProgressFunc func(*Progress)

// FilterChain represents a complex filter graph
type FilterChain struct {
	Filters []string
}
