package video

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ProcessVideo is for core processing
func ProcessVideo(inputVideo string, overlayVideo string, start float64, end float64, output string) error {
	// resolve binary dir
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)

	// ffmpeg path
	var ffmpegPath string
	if runtime.GOOS == "windows" {
		ffmpegPath = filepath.Join(exeDir, "assets", "ffmpeg.exe")
	} else {
		ffmpegPath = filepath.Join(exeDir, "assets", "ffmpeg")
	}

	// make input paths absolute
	inputAbs, err := filepath.Abs(inputVideo)
	if err != nil {
		return err
	}
	overlayAbs, err := filepath.Abs(overlayVideo)
	if err != nil {
		return err
	}
	outputAbs, err := filepath.Abs(output)
	if err != nil {
		return err
	}

	// build ffmpeg args
	args := []string{
		"-y",
		"-i", inputAbs,
		"-i", overlayAbs,
		"-filter_complex", "[0:v][1:v]overlay=0:0:shortest=1[outv]",
		"-map", "[outv]", "-map", "0:a?",
		"-c:v", "libx264", "-c:a", "aac",
		"-ss", fmt.Sprintf("%.3f", start),
		"-to", fmt.Sprintf("%.3f", end),
		outputAbs,
	}

	fmt.Println("Running FFmpeg command:")
	fmt.Println(ffmpegPath, args)

	cmd := exec.Command(ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %v\n%s", err, stderr.String())
	}

	return nil
}

// GetVideoDuration returns duration in seconds
func GetVideoDuration(videoPath string) (float64, error) {
	// get directory of the running binary
	exePath, err := os.Executable()
	if err != nil {
		return 0, err
	}
	exeDir := filepath.Dir(exePath)

	// ffprobe path relative to binary
	var ffprobePath string
	if runtime.GOOS == "windows" {
		ffprobePath = filepath.Join(exeDir, "assets", "ffprobe.exe")
	} else {
		ffprobePath = filepath.Join(exeDir, "assets", "ffprobe")
	}

	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return 0, fmt.Errorf("failed to run ffprobe: %v", err)
	}

	durationStr := strings.TrimSpace(out.String())
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}
