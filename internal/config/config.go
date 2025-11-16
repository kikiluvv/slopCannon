package config

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type contextKey string

const configKey contextKey = "config"

// Config holds all application configuration
type Config struct {
	// Core settings
	WorkDir     string `yaml:"work_dir"`
	TempDir     string `yaml:"temp_dir"`
	Concurrency int    `yaml:"concurrency"`

	// AI settings
	AI AIConfig `yaml:"ai"`

	// FFmpeg settings
	FFmpeg FFmpegConfig `yaml:"ffmpeg"`

	// Subtitle settings
	Subtitles SubtitleConfig `yaml:"subtitles"`

	// Overlay settings
	Overlays OverlayConfig `yaml:"overlays"`
}

type AIConfig struct {
	ModelPath      string  `yaml:"model_path" env:"AI_MODEL_PATH"`
	UseModel       bool    `yaml:"use_model" env:"AI_USE_MODEL"`
	WhisperModel   string  `yaml:"whisper_model"`
	ScoreThreshold float64 `yaml:"score_threshold"`
}

type FFmpegConfig struct {
	BinaryPath string `yaml:"binary_path"`
	Threads    int    `yaml:"threads"`
	Preset     string `yaml:"preset"`
}

type SubtitleConfig struct {
	FontName     string `yaml:"font_name"`
	FontSize     int    `yaml:"font_size"`
	FontColor    string `yaml:"font_color"`
	OutlineWidth int    `yaml:"outline_width"`
}

type OverlayConfig struct {
	DefaultOverlay string            `yaml:"default_overlay"`
	Overlays       map[string]string `yaml:"overlays"`
}

// Load reads configuration from file or returns defaults
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if path == "" {
		path = findConfigFile()
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func defaultConfig() *Config {
	return &Config{
		WorkDir:     "./work",
		TempDir:     "./temp",
		Concurrency: 4,
		AI: AIConfig{
			ModelPath:      "./models/clip-vit-base.onnx",
			UseModel:       true,
			WhisperModel:   "base",
			ScoreThreshold: 0.7,
		},
		FFmpeg: FFmpegConfig{
			BinaryPath: "ffmpeg",
			Threads:    0,
			Preset:     "medium",
		},
		Subtitles: SubtitleConfig{
			FontName:     "Arial",
			FontSize:     24,
			FontColor:    "#FFFFFF",
			OutlineWidth: 2,
		},
		Overlays: OverlayConfig{
			DefaultOverlay: "none",
			Overlays:       make(map[string]string),
		},
	}
}

func findConfigFile() string {
	candidates := []string{
		"./config.yaml",
		"./config.yml",
		filepath.Join(os.Getenv("HOME"), ".slopcannon", "config.yaml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// WithConfig stores config in context
func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

// FromContext retrieves config from context
func FromContext(ctx context.Context) *Config {
	if cfg, ok := ctx.Value(configKey).(*Config); ok {
		return cfg
	}
	return defaultConfig()
}
