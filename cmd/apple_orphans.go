package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/spf13/cobra"
)

var appleOrphansCmd = &cobra.Command{
	Use:     "apple-orphans",
	Aliases: []string{"ao"},
	Short:   "Find and remove orphaned tracks from Apple Music library",
	Long: `Scans your Apple Music library for tracks where the underlying audio file
has been deleted or moved. Orphaned tracks can then be removed from the library.

This command communicates directly with the Music app via AppleScript.`,
	Run: runAppleOrphans,
}

var (
	appleOrphansAutoDelete bool
	appleOrphansQuiet      bool
)

func init() {
	rootCmd.AddCommand(appleOrphansCmd)
	appleOrphansCmd.Flags().BoolVarP(&appleOrphansAutoDelete, "delete", "d", false, "Automatically delete all orphaned tracks without confirmation")
	appleOrphansCmd.Flags().BoolVarP(&appleOrphansQuiet, "quiet", "q", false, "Only output count of orphaned tracks (useful for scripts)")
}

func runAppleOrphans(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	if !appleOrphansQuiet {
		Styles.Header.Println("Scanning Apple Music library for orphaned tracks...")
	}

	// Get total track count for progress
	total, err := applemusic.GetTrackCount()
	if err != nil {
		Styles.Error.Printf("Failed to get track count: %v\n", err)
		os.Exit(1)
	}

	if !appleOrphansQuiet {
		fmt.Printf("Library contains %d tracks\n\n", total)
	}

	// Find orphaned tracks with progress reporting
	orphans, err := applemusic.FindOrphanedTracks(func(checked, total int) {
		if !appleOrphansQuiet {
			fmt.Printf("\rChecking tracks: %d/%d", checked, total)
		}
	})
	if err != nil {
		Styles.Error.Printf("\nFailed to scan library: %v\n", err)
		os.Exit(1)
	}

	if !appleOrphansQuiet {
		fmt.Println() // New line after progress
	}

	if len(orphans) == 0 {
		if appleOrphansQuiet {
			fmt.Println("0")
		} else {
			Styles.Success.Println("\nNo orphaned tracks found!")
		}
		return
	}

	if appleOrphansQuiet {
		fmt.Println(len(orphans))
		return
	}

	// Display orphaned tracks
	Styles.Header.Printf("\nFound %d orphaned tracks:\n\n", len(orphans))

	for i, track := range orphans {
		fmt.Printf("%d. ", i+1)
		Styles.Title.Printf("%s", track.Name)
		if track.Artist != "" {
			fmt.Printf(" - ")
			Styles.Path.Printf("%s", track.Artist)
		}
		if track.Album != "" {
			fmt.Printf(" [%s]", track.Album)
		}
		fmt.Println()
	}

	// Ask for confirmation unless auto-delete is enabled
	if !appleOrphansAutoDelete {
		fmt.Println()
		Styles.Prompt.Print("Delete these tracks from Apple Music? [y/N]: ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return
		}
	}

	// Delete orphaned tracks
	fmt.Println("\nDeleting orphaned tracks...")

	var persistentIDs []string
	for _, t := range orphans {
		persistentIDs = append(persistentIDs, t.PersistentID)
	}

	err = applemusic.DeleteTracksByPersistentID(persistentIDs)
	if err != nil {
		Styles.Error.Printf("Error during deletion: %v\n", err)
		os.Exit(1)
	}

	Styles.Success.Printf("\nSuccessfully removed %d orphaned tracks from Apple Music library.\n", len(orphans))
}

// CleanupOrphanedTracks is a helper function that can be called from other commands
// (like detect-duplicates) to clean up orphans after file deletion.
// Returns the number of orphans cleaned up.
func CleanupOrphanedTracks(deletedPaths []string, autoConfirm bool) (int, error) {
	if !applemusic.IsAvailable() {
		return 0, fmt.Errorf("Apple Music app is not available")
	}

	// Find orphans (scan full library since deleted files won't match anymore)
	orphans, err := applemusic.FindOrphanedTracks(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to find orphaned tracks: %w", err)
	}

	if len(orphans) == 0 {
		return 0, nil
	}

	if !autoConfirm {
		Styles.Header.Printf("\nFound %d orphaned tracks in Apple Music:\n", len(orphans))
		for i, track := range orphans {
			fmt.Printf("  %d. %s - %s\n", i+1, track.Artist, track.Name)
		}

		Styles.Prompt.Print("\nRemove these from Apple Music library? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			return 0, nil
		}
	}

	var persistentIDs []string
	for _, t := range orphans {
		persistentIDs = append(persistentIDs, t.PersistentID)
	}

	err = applemusic.DeleteTracksByPersistentID(persistentIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to delete tracks: %w", err)
	}

	return len(orphans), nil
}
