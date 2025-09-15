package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// RenderClip trims input video and overlays Minecraft background in portrait mode
func RenderClip(inputVideo string, clipStart, clipEnd float64, mcOverlay string, clipIndex int) error {
	// locate exe dir
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate executable: %v", err)
	}
	exeDir := filepath.Dir(exePath)

	// path to ffmpeg binary relative to exe dir
	var ffmpegPath string
	if runtime.GOOS == "windows" {
		ffmpegPath = filepath.Join(exeDir, "assets", "ffmpeg.exe")
	} else {
		ffmpegPath = filepath.Join(exeDir, "assets", "ffmpeg")
	}

	// resolve input + overlay paths
	inputVideoAbs, err := filepath.Abs(inputVideo)
	if err != nil {
		return fmt.Errorf("failed to resolve input video path: %v", err)
	}
	mcOverlayAbs := filepath.Join(exeDir, mcOverlay)

	// ensure output dir inside exe dir
	outputDir := filepath.Join(exeDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %v", err)
	}
	outputFileAbs := filepath.Join(outputDir, fmt.Sprintf("clip_%d.mp4", clipIndex))

	/*
	   filter breakdown:
	   - input scaled/cropped to 1080x960
	   - mc scaled to 1080x960
	   - vstack → final 1080x1920
	*/
	filter := "[0:v]scale=-1:960,crop=1080:960:(in_w-1080)/2:0[vid];" +
		"[1:v]scale=1080:960[mc];" +
		"[vid][mc]vstack=inputs=2[outv]"

	// FFmpeg command
	cmd := exec.Command(ffmpegPath,
		"-y",
		"-i", inputVideoAbs,
		"-i", mcOverlayAbs,
		"-filter_complex", filter,
		"-map", "[outv]", "-map", "0:a?",
		"-c:v", "libx264", "-c:a", "aac",
		"-movflags", "+faststart",
		"-preset", "veryfast",
		"-ss", fmt.Sprintf("%.3f", clipStart),
		"-to", fmt.Sprintf("%.3f", clipEnd),
		outputFileAbs,
	)

	// show ffmpeg logs in terminal
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running FFmpeg command:")
	fmt.Println(cmd.String())

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %v", err)
	}

	fmt.Println("Clip rendered successfully:", outputFileAbs)
	return nil
}
