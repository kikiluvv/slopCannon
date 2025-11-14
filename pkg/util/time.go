package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatDuration converts time.Duration to ffmpeg timestamp format
func FormatDuration(d time.Duration) string {
	seconds := d.Seconds()
	hours := int(seconds / 3600)
	minutes := int((seconds - float64(hours*3600)) / 60)
	secs := seconds - float64(hours*3600) - float64(minutes*60)
	return fmt.Sprintf("%02d:%02d:%06.3f", hours, minutes, secs)
}

// ParseTimestamp parses a timestamp string (HH:MM:SS.mmm or SS.mmm or MM:SS)
func ParseTimestamp(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	// Handle different formats
	parts := strings.Split(s, ":")

	var hours, minutes, seconds float64
	var err error

	switch len(parts) {
	case 1:
		// Just seconds (e.g., "45.5")
		seconds, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}

	case 2:
		// MM:SS format
		minutes, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}
		seconds, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}

	case 3:
		// HH:MM:SS format
		hours, err = strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}
		minutes, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}
		seconds, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timestamp format: %s", s)
		}

	default:
		return 0, fmt.Errorf("invalid timestamp format: %s", s)
	}

	totalSeconds := hours*3600 + minutes*60 + seconds
	return time.Duration(totalSeconds * float64(time.Second)), nil
}

// FormatTimestamp formats a duration as a simple timestamp string
func FormatTimestamp(d time.Duration) string {
	return FormatDuration(d)
}

// ParseFrameRate parses frame rate from ffprobe format (e.g., "30/1")
func ParseFrameRate(s string) float64 {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return 0
	}
	num, err1 := strconv.ParseFloat(parts[0], 64)
	den, err2 := strconv.ParseFloat(parts[1], 64)
	if err1 != nil || err2 != nil || den == 0 {
		return 0
	}
	return num / den
}
