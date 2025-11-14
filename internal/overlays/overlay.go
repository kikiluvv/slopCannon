package overlays

import (
	"context"
	"time"
)

// Renderer applies overlays to video
type Renderer interface {
	Render(ctx context.Context, input, output string, overlays []Overlay) error
}

// Overlay represents a video overlay
type Overlay struct {
	Type     string
	Path     string
	Start    time.Duration
	End      time.Duration
	Opacity  float64
	Position Position
}

// Position defines overlay placement
type Position struct {
	X int
	Y int
}

// Registry manages available overlays
type Registry struct {
	overlays map[string]string
}

// NewRegistry creates a new overlay registry
func NewRegistry() *Registry {
	return &Registry{
		overlays: make(map[string]string),
	}
}

// Register adds an overlay to the registry
func (r *Registry) Register(name, path string) {
	r.overlays[name] = path
}

// Get retrieves an overlay path by name
func (r *Registry) Get(name string) (string, bool) {
	path, ok := r.overlays[name]
	return path, ok
}

// List returns all registered overlays
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.overlays))
	for name := range r.overlays {
		names = append(names, name)
	}
	return names
}

// Presets for common overlays
var (
	MinecraftParkour = "minecraft_parkour"
	CSGOSurfing      = "csgo_surfing"
	SubwaySurfers    = "subway_surfers"
)
