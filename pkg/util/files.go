package util

import (
	"os"
	"path/filepath"
)

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TempFile creates a temporary file with a specific extension
func TempFile(dir, pattern, ext string) (*os.File, error) {
	return os.CreateTemp(dir, pattern+"*"+ext)
}

// CleanupFiles removes multiple files, ignoring errors
func CleanupFiles(paths ...string) {
	for _, path := range paths {
		_ = os.Remove(path)
	}
}

// GetExtension returns the file extension
func GetExtension(path string) string {
	return filepath.Ext(path)
}
