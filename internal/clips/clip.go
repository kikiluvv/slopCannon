package clips

import (
	"context"
	"time"
)

// Clip represents a video segment with metadata
type Clip struct {
	ID        string
	Start     time.Duration
	End       time.Duration
	Duration  time.Duration
	Score     float64
	SourceURL string
	Metadata  map[string]interface{}
}

// Detector finds clips within a video
type Detector interface {
	Detect(ctx context.Context, videoPath string) ([]*Clip, error)
}

// Editor provides clip editing operations
type Editor interface {
	Trim(clip *Clip, start, end time.Duration) (*Clip, error)
	Split(clip *Clip, at time.Duration) ([]*Clip, error)
	Merge(clips []*Clip) (*Clip, error)
}

// Manager handles clip operations
type Manager struct {
	clips []*Clip
}

// NewManager creates a new clip manager
func NewManager() *Manager {
	return &Manager{
		clips: make([]*Clip, 0),
	}
}

// Add adds a clip to the manager
func (m *Manager) Add(clip *Clip) {
	m.clips = append(m.clips, clip)
}

// Get retrieves a clip by ID
func (m *Manager) Get(id string) *Clip {
	for _, clip := range m.clips {
		if clip.ID == id {
			return clip
		}
	}
	return nil
}

// All returns all clips
func (m *Manager) All() []*Clip {
	return m.clips
}
