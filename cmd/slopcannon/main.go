package main

import (
	"context"
	"os"
	"time"

	"github.com/keagan/slopcannon/internal/config"
	"github.com/keagan/slopcannon/internal/logging"
	"github.com/keagan/slopcannon/internal/pipeline"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

func main() {
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "slopcannon",
	Short: "slopCannon - viral clip generation toolkit",
	Long:  "A modular Go-powered viral-clip generation toolkit that slices, scores, edits, and exports.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize logging
		logging.Init(verbose)

		// Load config
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		// Store config in context
		ctx := config.WithConfig(cmd.Context(), cfg)
		cmd.SetContext(ctx)

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(renderCmd)
	rootCmd.AddCommand(clipCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(listCmd)
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [input video]",
	Short: "Analyze video and detect clips",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.FromContext(cmd.Context())

		// Create pipeline
		pipeCfg := &pipeline.Config{
			Workers:     cfg.Concurrency,
			EnableCache: true,
		}
		pipe, err := pipeline.New(log.Logger, pipeCfg, cfg)
		if err != nil {
			return err
		}

		// Run analysis
		opts := pipeline.AnalyzeOptions{
			MinClipLen: 5 * time.Second,
			MaxClips:   10,
		}

		project, err := pipe.Analyze(cmd.Context(), args[0], opts)
		if err != nil {
			return err
		}

		log.Info().
			Str("project", project.Name).
			Int("clips", len(project.Clips)).
			Msg("analysis complete")

		return nil
	},
}

var renderCmd = &cobra.Command{
	Use:   "render [project file]",
	Short: "Render final video from project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Str("project", args[0]).Msg("rendering project")
		// TODO: wire up pipeline.Render()
		return nil
	},
}

var clipCmd = &cobra.Command{
	Use:   "clip",
	Short: "Clip editing commands",
}

var clipTrimCmd = &cobra.Command{
	Use:   "trim",
	Short: "Trim a clip",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("trimming clip")
		// TODO: wire up clip editor
		return nil
	},
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Config management commands",
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Msg("editing config")
		// TODO: open config in $EDITOR
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list [plugins|overlays|models]",
	Short: "List available resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info().Str("resource", args[0]).Msg("listing resources")
		// TODO: wire up registry
		return nil
	},
}

func init() {
	clipCmd.AddCommand(clipTrimCmd)
	configCmd.AddCommand(configEditCmd)
}
