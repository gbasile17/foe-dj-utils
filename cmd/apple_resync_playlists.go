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

	type pending struct {
		name string
		pid  string
	}
	var (
		succeeded     []string
		failed        []string
		needsRename   []pending
	)

	// Phase 1: clone + delete-original for every target. Leaves each clone
	// named "<name> (resync)".
	for i, name := range targets {
		Styles.Header.Printf("[%d/%d] %s\n", i+1, len(targets), name)
		newPID, err := applemusic.CloneAndDeletePlaylist(name)
		if err != nil {
			Styles.Error.Printf("    failed: %v\n", err)
			failed = append(failed, name)
			continue
		}
		Styles.Success.Printf("    cloned + deleted original (new pid: %s)\n", newPID)
		needsRename = append(needsRename, pending{name: name, pid: newPID})
	}

	// Phase 2: drive the renames from Go. Music's name index lags after each
	// delete and `delay` inside AppleScript blocks the very event loop that
	// needs to drain that lag — so the retry has to happen across separate
	// osascript invocations with sleeps in between.
	if len(needsRename) > 0 {
		fmt.Println()
		Styles.Header.Printf("Renaming %d clone(s) to original names...\n", len(needsRename))

		const renameTimeout = 5 * time.Minute
		const renameInterval = 2 * time.Second
		deadline := time.Now().Add(renameTimeout)

		for len(needsRename) > 0 && time.Now().Before(deadline) {
			var stillPending []pending
			for _, p := range needsRename {
				ok, err := applemusic.TryRenamePlaylist(p.pid, p.name)
				if err != nil {
					Styles.Error.Printf("    %s: rename error: %v\n", p.name, err)
					failed = append(failed, p.name)
					continue
				}
				if ok {
					Styles.Success.Printf("    renamed: %s\n", p.name)
					succeeded = append(succeeded, p.name)
				} else {
					stillPending = append(stillPending, p)
				}
			}
			needsRename = stillPending
			if len(needsRename) > 0 {
				time.Sleep(renameInterval)
			}
		}

		// Anything still pending after the timeout is a failure — but the
		// "(resync)" leftover is recoverable: just rerun the command, the
		// rename phase will pick them up again on a clean slate.
		for _, p := range needsRename {
			Styles.Error.Printf("    %s: rename never took effect (clone still named %q)\n", p.name, p.name+" (resync)")
			failed = append(failed, p.name)
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
