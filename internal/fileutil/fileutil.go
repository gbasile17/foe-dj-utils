// Package fileutil provides common file operations used across the application.
package fileutil

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CopyFile copies a file from src to dst, preserving permissions.
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, info.Mode())
}

// SanitizeFilename removes characters not allowed in filenames.
func SanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "'",
		"<", "",
		">", "",
		"|", "-",
	)
	return replacer.Replace(name)
}

// WriteM3U writes an M3U playlist file.
func WriteM3U(path string, entries []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("#EXTM3U\n")
	for _, entry := range entries {
		f.WriteString(entry + "\n")
	}
	return nil
}

// WalkAudioFiles walks a directory and sends audio file paths to a channel.
// It closes the channel when done.
func WalkAudioFiles(dir string, fileChan chan<- string, isAudioFile func(string) bool) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && isAudioFile(path) {
			fileChan <- path
		}
		return nil
	})
}

// WalkAllFiles walks a directory and sends all file paths to a channel.
func WalkAllFiles(dir string, fileChan chan<- string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileChan <- path
		}
		return nil
	})
}

// DirExists checks if a directory exists.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// FileExists checks if a file exists.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
