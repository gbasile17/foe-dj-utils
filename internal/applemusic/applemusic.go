// Package applemusic provides functions to interact with the Apple Music app
// on macOS via AppleScript automation.
package applemusic

import (
	"fmt"
	"os/exec"
	"strings"
)

// Track represents a track in the Apple Music library
type Track struct {
	Name         string
	Artist       string
	Album        string
	PersistentID string
	DatabaseID   int
	Location     string // POSIX path, empty if orphaned
	IsOrphaned   bool
}

// Playlist represents a playlist in Apple Music
type Playlist struct {
	Name   string
	Tracks []Track
}

// runAppleScript executes an AppleScript and returns the output
func runAppleScript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AppleScript error: %v\nOutput: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// runAppleScriptFile executes a multi-line AppleScript via stdin
func runAppleScriptFile(script string) (string, error) {
	cmd := exec.Command("osascript")
	cmd.Stdin = strings.NewReader(script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("AppleScript error: %v\nOutput: %s", err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// IsAvailable checks if the Music app is available on this system
func IsAvailable() bool {
	_, err := runAppleScript(`tell application "System Events" to return exists application process "Music"`)
	return err == nil
}

// GetTrackCount returns the total number of file tracks in the library
func GetTrackCount() (int, error) {
	output, err := runAppleScript(`tell application "Music" to return count of file tracks of playlist "Library"`)
	if err != nil {
		return 0, err
	}
	var count int
	_, err = fmt.Sscanf(output, "%d", &count)
	return count, err
}

// GetPlaylistNames returns a list of all playlist names
func GetPlaylistNames() ([]string, error) {
	output, err := runAppleScript(`tell application "Music" to return name of every user playlist`)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	// AppleScript returns comma-separated list
	names := strings.Split(output, ", ")
	return names, nil
}