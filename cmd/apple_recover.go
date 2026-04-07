package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/gbasile17/foe/dj-utils/internal/music"
	"github.com/spf13/cobra"
)

var appleRecoverCmd = &cobra.Command{
	Use:     "apple-recover [music-directory]",
	Aliases: []string{"ar"},
	Short:   "Find audio files not in Apple Music library and add them",
	Long: `Scans a directory for audio files that exist on disk but are not in your
Apple Music library. Found files can be added to the library and optionally
to a "Recovery" playlist.

This is useful for recovering files that were accidentally removed from
the library but still exist on disk.

If no directory is provided, defaults to ~/Music/Music/Media.

Example:
  foe apple-recover
  foe apple-recover ~/Music/Music/Media
  foe ar --playlist "Found Files"`,
	Run: runAppleRecover,
}

var (
	recoverPlaylistName string
	recoverDryRun       bool
)

func init() {
	rootCmd.AddCommand(appleRecoverCmd)
	appleRecoverCmd.Flags().StringVarP(&recoverPlaylistName, "playlist", "p", "Recovery", "Name of playlist to add recovered files to")
	appleRecoverCmd.Flags().BoolVarP(&recoverDryRun, "dry-run", "n", false, "Only show what would be recovered, don't add to library")
}

func runAppleRecover(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	// Default to ~/Music/Music/Media
	musicDir := filepath.Join(os.Getenv("HOME"), "Music", "Music", "Media")
	if len(args) > 0 {
		musicDir = args[0]
	}

	// Convert to absolute path for comparison with library locations
	musicDir, err := filepath.Abs(musicDir)
	if err != nil {
		Styles.Error.Printf("Failed to resolve path: %v\n", err)
		os.Exit(1)
	}

	// Verify directory exists
	if _, err := os.Stat(musicDir); os.IsNotExist(err) {
		Styles.Error.Printf("Directory does not exist: %s\n", musicDir)
		os.Exit(1)
	}

	Styles.Header.Printf("Scanning for files not in Apple Music library...\n")
	fmt.Printf("Directory: %s\n\n", musicDir)

	// Get all library locations
	fmt.Print("Loading library locations...")
	libraryLocations, err := applemusic.GetAllLibraryLocations(func(checked, total int) {
		fmt.Printf("\rLoading library locations: %d/%d", checked, total)
	})
	if err != nil {
		Styles.Error.Printf("\nFailed to get library locations: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\rLoaded %d tracks from library          \n\n", len(libraryLocations))

	// Scan directory for audio files not in library
	var missingFiles []string
	var scannedCount int

	err = filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}
		if info.IsDir() {
			return nil
		}

		if music.IsAudioFile(path) {
			scannedCount++
			if scannedCount%100 == 0 {
				fmt.Printf("\rScanning files: %d", scannedCount)
			}

			if !libraryLocations[path] {
				missingFiles = append(missingFiles, path)
			}
		}
		return nil
	})

	fmt.Printf("\rScanned %d audio files                \n", scannedCount)

	if err != nil {
		Styles.Error.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(missingFiles) == 0 {
		Styles.Success.Println("\nNo missing files found! All audio files are in the library.")
		return
	}

	Styles.Header.Printf("\nFound %d files not in library:\n\n", len(missingFiles))

	// Show up to 20 files
	displayCount := len(missingFiles)
	if displayCount > 20 {
		displayCount = 20
	}
	for i := 0; i < displayCount; i++ {
		// Show relative path for readability
		relPath, _ := filepath.Rel(musicDir, missingFiles[i])
		if relPath == "" {
			relPath = missingFiles[i]
		}
		fmt.Printf("  %d. %s\n", i+1, Styles.Path.Sprint(relPath))
	}
	if len(missingFiles) > 20 {
		fmt.Printf("  ... and %d more\n", len(missingFiles)-20)
	}

	if recoverDryRun {
		fmt.Println("\n[Dry run - no changes made]")
		return
	}

	// Confirm
	fmt.Println()
	if !promptYesNo(Styles.Prompt.Sprint("Add these files to the library?")) {
		fmt.Println("Cancelled.")
		return
	}

	// Add files to library
	fmt.Println("\nAdding files to library...")
	added, err := applemusic.AddFilesToLibrary(missingFiles, func(current, total int) {
		fmt.Printf("\rAdding: %d/%d", current, total)
	})
	fmt.Println()

	if err != nil {
		Styles.Error.Printf("Error adding files: %v\n", err)
	}

	Styles.Success.Printf("Added %d files to library\n", added)

	if added == 0 {
		return
	}

	// Create playlist and add files
	if recoverPlaylistName != "" {
		fmt.Printf("\nCreating playlist '%s'...\n", recoverPlaylistName)

		// Check if playlist exists
		exists, _ := applemusic.PlaylistExists(recoverPlaylistName)
		if !exists {
			if err := applemusic.CreatePlaylist(recoverPlaylistName); err != nil {
				// Playlist might exist with different casing
				if !strings.Contains(err.Error(), "already exists") {
					Styles.Error.Printf("Failed to create playlist: %v\n", err)
				}
			}
		}

		fmt.Println("Adding files to playlist...")
		playlistAdded, _ := applemusic.AddFilesToPlaylist(missingFiles, recoverPlaylistName, func(current, total int) {
			fmt.Printf("\rAdding to playlist: %d/%d", current, total)
		})
		fmt.Println()

		Styles.Success.Printf("Added %d files to playlist '%s'\n", playlistAdded, recoverPlaylistName)
	}
}
