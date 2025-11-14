package ffmpeg

import (
	"fmt"
	"strings"
)

// FilterBuilder helps construct complex ffmpeg filter chains
type FilterBuilder struct {
	filters []string
}

// NewFilterBuilder creates a new filter builder
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filters: make([]string, 0),
	}
}

// Scale adds a scale filter
func (fb *FilterBuilder) Scale(width, height int) *FilterBuilder {
	if width <= 0 || height <= 0 {
		// Return self without adding filter - allows chaining to continue
		return fb
	}
	fb.filters = append(fb.filters, fmt.Sprintf("scale=%d:%d", width, height))
	return fb
}

// FPS adds an fps filter
func (fb *FilterBuilder) FPS(fps float64) *FilterBuilder {
	if fps <= 0 {
		return fb
	}
	fb.filters = append(fb.filters, fmt.Sprintf("fps=%f", fps))
	return fb
}

// Crop adds a crop filter
func (fb *FilterBuilder) Crop(width, height, x, y int) *FilterBuilder {
	if width <= 0 || height <= 0 {
		return fb
	}
	fb.filters = append(fb.filters, fmt.Sprintf("crop=%d:%d:%d:%d", width, height, x, y))
	return fb
}

// Fade adds a fade in/out filter
func (fb *FilterBuilder) Fade(fadeIn, fadeOut bool, duration int) *FilterBuilder {
	if fadeIn {
		fb.filters = append(fb.filters, fmt.Sprintf("fade=in:0:%d", duration))
	}
	if fadeOut {
		fb.filters = append(fb.filters, fmt.Sprintf("fade=out:0:%d", duration))
	}
	return fb
}

// AudioVolume adjusts audio volume
func (fb *FilterBuilder) AudioVolume(volumeDB float64) *FilterBuilder {
	fb.filters = append(fb.filters, fmt.Sprintf("volume=%fdB", volumeDB))
	return fb
}

// Custom adds a custom filter string
func (fb *FilterBuilder) Custom(filter string) *FilterBuilder {
	fb.filters = append(fb.filters, filter)
	return fb
}

// Build returns the complete filter string joined with commas
func (fb *FilterBuilder) Build() string {
	if len(fb.filters) == 0 {
		return ""
	}
	return strings.Join(fb.filters, ",")
}

// BuildAll returns all filters as a slice
func (fb *FilterBuilder) BuildAll() []string {
	return fb.filters
}
