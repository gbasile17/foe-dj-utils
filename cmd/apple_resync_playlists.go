package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/spf13/cobra"
)

var appleResyncPlaylistsCmd = &cobra.Command{
	Use:     "apple-resync-playlists",
	Aliases: []string{"arp"},
	Short:   "Recreate Apple Music playlists to fix iCloud sync issues",
	Long: `Recreates user playlists in Apple Music to force iCloud Music Library
to re-upload them. This is the documented workaround for playlists that
have desynced from the cloud.

For each target playlist:
  1. Clone its tracks into a new playlist named "<name> (resync)"
  2. Verify the clone has the same track count
  3. Delete the original
  4. Rename the clone to the original name

Smart playlists, folder playlists, and special-kind playlists are skipped.
The verify-then-delete order means the original is preserved if anything
goes wrong before step 3.

Examples:
  foe apple-resync-playlists -a
  foe apple-resync-playlists -p "Tech House" -p "Afro"
  foe arp -p "foo"`,
	Run: runAppleResyncPlaylists,
}

var (
	resyncAll       bool
	resyncPlaylists []string
)

func init() {
	rootCmd.AddCommand(appleResyncPlaylistsCmd)
	appleResyncPlaylistsCmd.Flags().BoolVarP(&resyncAll, "all", "a", false, "Resync all user playlists")
	appleResyncPlaylistsCmd.Flags().StringArrayVarP(&resyncPlaylists, "playlist", "p", nil, "Specific playlist name to resync (repeatable)")
}

func runAppleResyncPlaylists(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	if resyncAll == (len(resyncPlaylists) > 0) {
		Styles.Error.Println("Error: specify exactly one of -a/--all or -p/--playlist")
		os.Exit(1)
	}

	var targets []string
	if resyncAll {
		all, err := applemusic.GetUserPlaylists()
		if err != nil {
			Styles.Error.Printf("Failed to list playlists: %v\n", err)
			os.Exit(1)
		}
		targets = all
	} else {
		targets = resyncPlaylists
		for _, name := range targets {
			exists, err := applemusic.PlaylistExists(name)
			if err != nil {
				Styles.Error.Printf("Failed to check playlist %q: %v\n", name, err)
				os.Exit(1)
			}
			if !exists {
				Styles.Error.Printf("Playlist not found: %q\n", name)
				os.Exit(1)
			}
		}
	}

	if len(targets) == 0 {
		Styles.Error.Println("No playlists to resync.")
		return
	}

	Styles.Header.Printf("Resyncing %d playlist(s):\n", len(targets))
	for _, name := range targets {
		fmt.Printf("  - %s\n", name)
	}
	fmt.Println()

	Styles.Prompt.Print("This will delete and recreate each playlist. Continue? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}
	fmt.Println()

	var succeeded, failed []string
	for i, name := range targets {
		Styles.Header.Printf("[%d/%d] %s\n", i+1, len(targets), name)
		newPID, err := applemusic.RecreatePlaylist(name)
		if err != nil {
			Styles.Error.Printf("    failed: %v\n", err)
			failed = append(failed, name)
			continue
		}
		Styles.Success.Printf("    recreated (new persistent ID: %s)\n", newPID)
		succeeded = append(succeeded, name)

		// Give Music's name index time to release the deleted name before the
		// next playlist's rename. The lag grows when recreating playlists
		// back-to-back, and skipping this is what caused (resync) leftovers.
		if i < len(targets)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Println()
	Styles.Header.Println("Resync Summary:")
	Styles.Success.Printf("  Recreated: %d\n", len(succeeded))
	if len(failed) > 0 {
		Styles.Error.Printf("  Failed: %d\n", len(failed))
		for _, name := range failed {
			Styles.Error.Printf("    - %s\n", name)
		}
		os.Exit(1)
	}
}
