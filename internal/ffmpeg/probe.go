package ffmpeg

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/keagan/slopcannon/pkg/util"
)

// ProbeVideo extracts metadata from a video file
func (e *Executor) ProbeVideo(ctx context.Context, filePath string) (*VideoInfo, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is required")
	}

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.CommandContext(ctx, e.ffprobePath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe probeResult
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{
		FilePath: filePath,
	}

	// Parse duration
	if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		info.Duration = time.Duration(dur * float64(time.Second))
	}

	// Parse bitrate
	if br, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
		info.Bitrate = br
	}

	// Extract video stream info
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			info.Width = stream.Width
			info.Height = stream.Height
			info.VideoCodec = stream.CodecName

			// Calculate FPS from r_frame_rate (e.g., "30/1")
			if stream.RFrameRate != "" {
				info.FPS = util.ParseFrameRate(stream.RFrameRate)
			}
		} else if stream.CodecType == "audio" {
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
			if br, err := strconv.ParseInt(stream.BitRate, 10, 64); err == nil {
				info.AudioBitrate = br
			}
		}
	}

	return info, nil
}

// probeResult matches ffprobe JSON output structure
type probeResult struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
}
