package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestResults stores results from all tests for final summary
type TestResults struct {
	ExecutorPath  string
	ProbeResults  *VideoInfo
	ClipCreated   bool
	ScenesFound   int
	SilencesFound int
	VolumeStats   *VolumeStats
	Errors        []string
	TestDuration  time.Duration
}

var globalResults = &TestResults{
	Errors: make([]string, 0),
}

// getTestDataPath returns the path to testdata from project root
func getTestDataPath(filename string) string {
	return filepath.Join("..", "..", "testdata", filename)
}

// skipIfNoFFmpeg skips the test if ffmpeg is not available
func skipIfNoFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found in PATH - install with: brew install ffmpeg")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not found in PATH - install with: brew install ffmpeg")
	}
}

func TestExecutorCreation(t *testing.T) {
	skipIfNoFFmpeg(t)

	logger := zerolog.New(os.Stderr)
	exec, err := New(logger, 4)
	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("Executor creation failed: %v", err))
		t.Fatalf("failed to create executor: %v", err)
	}
	if exec.ffmpegPath == "" {
		t.Error("ffmpeg path is empty")
	}
	if exec.ffprobePath == "" {
		t.Error("ffprobe path is empty")
	}

	globalResults.ExecutorPath = exec.ffmpegPath
	t.Logf("ffmpeg: %s", exec.ffmpegPath)
	t.Logf("ffprobe: %s", exec.ffprobePath)
}

func TestProbeVideo(t *testing.T) {
	skipIfNoFFmpeg(t)

	testVideoPath := getTestDataPath("test.mp4")
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		globalResults.Errors = append(globalResults.Errors, "Test video not found")
		t.Skipf("test video not found at %s", testVideoPath)
	}

	logger := zerolog.New(os.Stderr)
	exec, err := New(logger, 2)
	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("Probe failed: %v", err))
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	info, err := exec.ProbeVideo(ctx, testVideoPath)
	elapsed := time.Since(start)

	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("ProbeVideo failed: %v", err))
		t.Fatalf("ProbeVideo failed: %v", err)
	}

	globalResults.ProbeResults = info
	globalResults.TestDuration = elapsed

	if info.Width != 320 {
		t.Errorf("expected width 320, got %d", info.Width)
	}
	if info.Height != 240 {
		t.Errorf("expected height 240, got %d", info.Height)
	}
	if info.Duration == 0 {
		t.Error("duration is zero")
	}

	t.Logf("Video info: %dx%d, %.2f fps, duration: %v (probed in %v)",
		info.Width, info.Height, info.FPS, info.Duration, elapsed)
}

func TestExtractClip(t *testing.T) {
	skipIfNoFFmpeg(t)

	testVideoPath := getTestDataPath("test.mp4")
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		t.Skip("test video not found")
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	outputPath := getTestDataPath("clip_output.mp4")
	defer func() {
		if _, err := os.Stat(outputPath); err == nil {
			_ = os.Remove(outputPath)
		}
	}()

	opts := ClipOptions{
		Start:     0,
		End:       500 * time.Millisecond,
		Output:    outputPath,
		CopyCodec: true,
	}

	start := time.Now()
	err = exec.ExtractClip(ctx, testVideoPath, opts)
	elapsed := time.Since(start)

	if err != nil {
		globalResults.Errors = append(globalResults.Errors,
			fmt.Sprintf("ExtractClip failed: %v", err))
		t.Fatalf("ExtractClip failed: %v", err)
	}

	// Verify output exists
	stat, err := os.Stat(outputPath)
	if err != nil {
		globalResults.ClipCreated = false
		t.Fatalf("output file was not created: %v", err)
	}

	globalResults.ClipCreated = true
	t.Logf("Clip created: %s (size: %d bytes, took %v)",
		outputPath, stat.Size(), elapsed)
}
func TestFilterBuilder(t *testing.T) {
	fb := NewFilterBuilder()
	filter := fb.Scale(1920, 1080).FPS(30).Build()

	expected := "scale=1920:1080,fps=30.000000"
	if filter != expected {
		t.Errorf("expected %q, got %q", expected, filter)
	}
}

func TestFilterBuilderEmpty(t *testing.T) {
	fb := NewFilterBuilder()
	filter := fb.Build()

	if filter != "" {
		t.Errorf("expected empty string, got %q", filter)
	}
}

func TestFilterBuilderChaining(t *testing.T) {
	fb := NewFilterBuilder()
	filter := fb.Scale(1920, 1080).FPS(60).Build()

	expected := "scale=1920:1080,fps=60.000000"
	if filter != expected {
		t.Errorf("expected %q, got %q", expected, filter)
	}
}

func TestDetectScenes(t *testing.T) {
	skipIfNoFFmpeg(t)

	testVideoPath := getTestDataPath("test.mp4")
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		t.Skip("test video not found")
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	scenes, err := exec.DetectScenes(ctx, testVideoPath, 0.3)
	elapsed := time.Since(start)

	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("DetectScenes failed: %v", err))
		t.Fatalf("DetectScenes failed: %v", err)
	}

	globalResults.ScenesFound = len(scenes)

	t.Logf("Found %d scene changes in %v", len(scenes), elapsed)
	for i, scene := range scenes {
		if i >= 5 {
			t.Logf("  ... and %d more", len(scenes)-5)
			break
		}
		t.Logf("  Scene %d: %v", i+1, scene)
	}
}

func TestDetectSilence(t *testing.T) {
	skipIfNoFFmpeg(t)

	testVideoPath := getTestDataPath("test_with_audio.mp4")

	// Generate a 2-second video with audio (sine wave)
	cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
		"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=30",
		"-pix_fmt", "yuv420p", "-y", testVideoPath)
	if err := cmd.Run(); err != nil {
		t.Skipf("Could not generate test video with audio: %v", err)
	}
	defer os.Remove(testVideoPath)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	silences, err := exec.DetectSilence(ctx, testVideoPath, -30, 0.5)
	elapsed := time.Since(start)

	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("DetectSilence failed: %v", err))
		t.Fatalf("DetectSilence failed: %v", err)
	}

	globalResults.SilencesFound = len(silences)

	t.Logf("Found %d silence periods in %v", len(silences), elapsed)
	for i, silence := range silences {
		t.Logf("  Silence %d: %.2fs - %.2fs (%.2fs)",
			i+1, silence.Start, silence.End, silence.Duration)
	}
}

func TestAnalyzeVolume(t *testing.T) {
	skipIfNoFFmpeg(t)

	testVideoPath := getTestDataPath("test_with_audio.mp4")

	// Generate if it doesn't exist
	if _, err := os.Stat(testVideoPath); os.IsNotExist(err) {
		cmd := exec.Command("ffmpeg", "-f", "lavfi", "-i", "sine=frequency=1000:duration=2",
			"-f", "lavfi", "-i", "testsrc=duration=2:size=320x240:rate=30",
			"-pix_fmt", "yuv420p", "-y", testVideoPath)
		if err := cmd.Run(); err != nil {
			t.Skipf("Could not generate test video with audio: %v", err)
		}
	}
	defer os.Remove(testVideoPath)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	stats, err := exec.AnalyzeVolume(ctx, testVideoPath)
	elapsed := time.Since(start)

	if err != nil {
		globalResults.Errors = append(globalResults.Errors, fmt.Sprintf("AnalyzeVolume failed: %v", err))
		t.Fatalf("AnalyzeVolume failed: %v", err)
	}

	globalResults.VolumeStats = stats

	t.Logf("Volume analysis completed in %v:", elapsed)
	t.Logf("  Mean: %.2f dB", stats.MeanVolume)
	t.Logf("  Max: %.2f dB", stats.MaxVolume)

	if stats.MeanVolume < -100 {
		t.Error("Mean volume suspiciously low")
	}
}

func TestConcatValidation(t *testing.T) {
	skipIfNoFFmpeg(t)

	logger := zerolog.New(os.Stderr)
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()

	opts := ConcatOptions{
		Inputs: []string{"nonexistent1.mp4", "nonexistent2.mp4"},
		Output: "output.mp4",
	}

	err = exec.Concat(ctx, opts)
	t.Logf("Concat with non-existent files returned: %v", err)
}

func TestProbeVideoInvalidFile(t *testing.T) {
	skipIfNoFFmpeg(t)

	logger := zerolog.New(os.Stderr)
	exec, err := New(logger, 2)
	if err != nil {
		t.Fatalf("failed to create executor: %v", err)
	}

	ctx := context.Background()

	_, err = exec.ProbeVideo(ctx, "nonexistent.mp4")
	if err == nil {
		t.Error("ProbeVideo should fail for non-existent file")
	}
	t.Logf("Error (expected): %v", err)

	invalidPath := getTestDataPath("invalid.txt")
	os.WriteFile(invalidPath, []byte("not a video"), 0644)
	defer os.Remove(invalidPath)

	_, err = exec.ProbeVideo(ctx, invalidPath)
	if err == nil {
		t.Error("ProbeVideo should fail for invalid video file")
	}
	t.Logf("Error (expected): %v", err)
}

// TestMain runs after all tests and prints summary
func TestMain(m *testing.M) {
	code := m.Run()

	// Print summary
	printTestSummary()

	os.Exit(code)
}

func printTestSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎬 TEST SUMMARY - FFmpeg Layer")
	fmt.Println(strings.Repeat("=", 80))

	if globalResults.ExecutorPath != "" {
		fmt.Printf("\n✓ FFmpeg Binary: %s\n", globalResults.ExecutorPath)
	}

	if globalResults.ProbeResults != nil {
		fmt.Println("\n📹 VIDEO PROBE RESULTS:")
		fmt.Printf("  Resolution:    %dx%d @ %.2f fps\n",
			globalResults.ProbeResults.Width,
			globalResults.ProbeResults.Height,
			globalResults.ProbeResults.FPS)
		fmt.Printf("  Duration:      %v\n", globalResults.ProbeResults.Duration)
		fmt.Printf("  Video Codec:   %s\n", globalResults.ProbeResults.VideoCodec)
		fmt.Printf("  Audio Codec:   %s\n", globalResults.ProbeResults.AudioCodec)
		fmt.Printf("  Probe Time:    %v\n", globalResults.TestDuration)
	}

	fmt.Println("\n🎬 PROCESSING RESULTS:")
	if globalResults.ClipCreated {
		fmt.Println("  ✓ Clip Extraction:  SUCCESS")
	} else {
		fmt.Println("  ✗ Clip Extraction:  FAILED")
	}

	fmt.Printf("  🎞️  Scene Changes:    %d detected\n", globalResults.ScenesFound)

	if globalResults.SilencesFound > 0 {
		fmt.Printf("  🔇 Silence Periods:  %d detected\n", globalResults.SilencesFound)
	} else {
		fmt.Printf("  🔇 Silence Periods:  0 (continuous audio)\n")
	}

	if globalResults.VolumeStats != nil {
		fmt.Println("\n🔊 AUDIO ANALYSIS:")
		fmt.Printf("  Mean Volume:   %6.2f dB\n", globalResults.VolumeStats.MeanVolume)
		fmt.Printf("  Peak Volume:   %6.2f dB\n", globalResults.VolumeStats.MaxVolume)

		// Audio quality assessment
		if globalResults.VolumeStats.MeanVolume > -12 {
			fmt.Println("  Quality:       ⚠️  May be too loud (risk of clipping)")
		} else if globalResults.VolumeStats.MeanVolume < -30 {
			fmt.Println("  Quality:       ⚠️  Low volume (may need normalization)")
		} else {
			fmt.Println("  Quality:       ✓ Good levels")
		}
	}

	if len(globalResults.Errors) > 0 {
		fmt.Println("\n❌ ERRORS ENCOUNTERED:")
		for i, err := range globalResults.Errors {
			fmt.Printf("  %d. %s\n", i+1, err)
		}
	} else {
		fmt.Println("\n✅ ALL TESTS PASSED - No critical errors")
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()
}
