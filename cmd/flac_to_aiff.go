package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/gbasile17/foe/dj-utils/pkg/audiotag"
	"github.com/spf13/cobra"
)

var recursive bool
var removeOriginal bool

// flacToAiffCmd represents the flac-to-aiff command
var flacToAiffCmd = &cobra.Command{
	Use:   "flac-to-aiff <directory>",
	Short: "Convert all FLAC files in a directory to AIFF using ffmpeg (preserves metadata)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := args[0]

		// Validate directory
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			fmt.Printf("Error: Directory %s does not exist\n", dir)
			os.Exit(1)
		}

		var files []string
		var err error

		// Use recursive or non-recursive file listing
		if recursive {
			err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Printf("Error accessing %s: %v\n", path, err)
					return nil
				}
				if !info.IsDir() && filepath.Ext(path) == ".flac" {
					files = append(files, path)
				}
				return nil
			})
		} else {
			files, err = filepath.Glob(filepath.Join(dir, "*.flac"))
		}

		if err != nil {
			fmt.Println("Error finding .flac files:", err)
			os.Exit(1)
		}

		if len(files) == 0 {
			fmt.Println("No FLAC files found.")
			return
		}

		fmt.Printf("Converting %d FLAC files to AIFF (preserving metadata)...\n", len(files))

		successCount := 0
		// Convert each file
		for i, flacFile := range files {
			fmt.Printf("\nProcessing %d/%d: %s\n", i+1, len(files), filepath.Base(flacFile))
			
			aiffFile := flacFile[:len(flacFile)-5] + ".aiff" // Replace .flac with .aiff

			// Read metadata from FLAC file before conversion
			fmt.Printf("  Reading metadata from FLAC file...")
			flacTags, err := audiotag.ReadTags(flacFile)
			if err != nil {
				color.Yellow("  Warning: Could not read metadata from %s: %v", flacFile, err)
				flacTags = &audiotag.AudioTags{} // Use empty tags as fallback
			} else {
				color.Green("  ✓ Metadata read successfully")
			}

			// Convert audio using ffmpeg (without metadata to avoid conflicts)
			fmt.Printf("  Converting audio to AIFF...")
			cmd := exec.Command("ffmpeg", "-i", flacFile, "-c:a", "pcm_s16be", "-y", aiffFile)
			cmd.Stdout = nil // Suppress ffmpeg output
			cmd.Stderr = nil

			err = cmd.Run()
			if err != nil {
				color.Red("  ✗ Error converting %s: %v", flacFile, err)
				continue
			}
			color.Green("  ✓ Audio conversion successful")

			// Write metadata to AIFF file using our audiotag library
			fmt.Printf("  Writing metadata to AIFF file...")
			err = audiotag.WriteTags(aiffFile, flacTags)
			if err != nil {
				color.Yellow("  Warning: Could not write metadata to %s: %v", aiffFile, err)
			} else {
				color.Green("  ✓ Metadata written successfully")
			}

			// Verify the metadata was written correctly
			fmt.Printf("  Verifying metadata...")
			aiffTags, err := audiotag.ReadTags(aiffFile)
			if err != nil {
				color.Yellow("  Warning: Could not verify metadata: %v", err)
			} else {
				if aiffTags.Title == flacTags.Title && aiffTags.Artist == flacTags.Artist {
					color.Green("  ✓ Metadata verification successful")
				} else {
					color.Yellow("  Warning: Metadata may not have transferred completely")
				}
			}

			color.Green("✓ Converted: %s -> %s", filepath.Base(flacFile), filepath.Base(aiffFile))
			successCount++

			// Delete original file if --remove flag is set
			if removeOriginal {
				err := os.Remove(flacFile)
				if err != nil {
					color.Red("  ✗ Error deleting %s: %v", flacFile, err)
				} else {
					color.Green("  ✓ Deleted original: %s", filepath.Base(flacFile))
				}
			}
		}

		// Summary
		fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
		if successCount == len(files) {
			color.Green("✅ Successfully converted all %d files!", successCount)
		} else {
			color.Yellow("⚠️  Converted %d/%d files", successCount, len(files))
		}
	},
}

func init() {
	flacToAiffCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Include subdirectories")
	flacToAiffCmd.Flags().BoolVarP(&removeOriginal, "remove", "d", false, "Delete original FLAC files after conversion")
	rootCmd.AddCommand(flacToAiffCmd)
}