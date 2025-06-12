package cmd

import "github.com/fatih/color"

// Styles provides reusable color configurations for the CLI outputs
var Styles = struct {
	Header  *color.Color // For headers or main sections
	Group   *color.Color // For duplicate groups (e.g., hash or title group)
	Title   *color.Color // For titles of the audio files
	Path    *color.Color // For file paths
	Prompt  *color.Color // For interactive prompts
	Error   *color.Color // For error messages
	Success *color.Color // For success messages
}{
	Header:  color.New(color.FgBlue, color.Bold),
	Group:   color.New(color.FgMagenta, color.Bold),
	Title:   color.New(color.FgYellow),
	Path:    color.New(color.FgCyan),
	Prompt:  color.New(color.FgGreen, color.Bold),
	Error:   color.New(color.FgRed),
	Success: color.New(color.FgGreen),
}
