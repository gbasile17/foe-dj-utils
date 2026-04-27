package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/spf13/cobra"
)

var appleQuarantineBlockersCmd = &cobra.Command{
	Use:     "apple-quarantine-blockers",
	Aliases: []string{"aqb"},
	Short:   "Move sync-blocking tracks out of all playlists into _block",
	Long: `Scans every user playlist for tracks whose cloud status prevents
iCloud Music Library sync and is unlikely to resolve on its own (ineligible,
error, removed, duplicate, no longer available) and moves them into a single
quarantine playlist named "_block". Tracks with status "not uploaded" are
left alone — Apple's matcher may not have run yet for them.

The tracks remain in your library — only their membership in the source
playlist is removed. The same track appearing in multiple source playlists
is added to "_block" once.

By default skips "_lib" (master library mirror) and the "_block" playlist
itself. Use --skip to skip additional playlists.

Examples:
  foe apple-quarantine-blockers
  foe aqb --dry-run
  foe aqb --skip "Master,Archive"`,
	Run: runAppleQuarantineBlockers,
}

var (
	quarantineDryRun bool
	quarantineDest   string
	quarantineSkip   []string
)

// Tree-drawing helpers.
func playlistBranch(isLast bool) string {
	if isLast {
		return "└─"
	}
	return "├─"
}

func trackBranch(playlistIsLast, trackIsLast bool) string {
	var pipe string
	if playlistIsLast {
		pipe = "   "
	} else {
		pipe = "│  "
	}
	if trackIsLast {
		return pipe + "└─"
	}
	return pipe + "├─"
}

// formatStatus pads cloud status to a fixed width so track titles align.
func formatStatus(status string) string {
	const w = 22
	tag := "[" + status + "]"
	for len(tag) < w {
		tag += " "
	}
	return tag
}

func init() {
	rootCmd.AddCommand(appleQuarantineBlockersCmd)
	appleQuarantineBlockersCmd.Flags().BoolVarP(&quarantineDryRun, "dry-run", "n", false, "Report what would be moved, don't change anything")
	appleQuarantineBlockersCmd.Flags().StringVar(&quarantineDest, "dest", "_block", "Destination playlist name")
	appleQuarantineBlockersCmd.Flags().StringSliceVar(&quarantineSkip, "skip", []string{"_lib"}, "Comma-separated playlist names to skip (in addition to dest)")
}

func runAppleQuarantineBlockers(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	all, err := applemusic.GetUserPlaylists()
	if err != nil {
		Styles.Error.Printf("Failed to list playlists: %v\n", err)
		os.Exit(1)
	}

	skip := map[string]bool{quarantineDest: true}
	for _, s := range quarantineSkip {
		s = strings.TrimSpace(s)
		if s != "" {
			skip[s] = true
		}
	}

	type candidate struct {
		name   string
		tracks []applemusic.BlockingTrack
	}
	var candidates []candidate
	totalToMove := 0
	for _, name := range all {
		if skip[name] {
			continue
		}
		tracks, err := applemusic.ListBlockingTracks(name)
		if err != nil {
			Styles.Error.Printf("Failed to list blockers in %q: %v\n", name, err)
			continue
		}
		if len(tracks) > 0 {
			candidates = append(candidates, candidate{name: name, tracks: tracks})
			totalToMove += len(tracks)
		}
	}

	if len(candidates) == 0 {
		Styles.Success.Println("No blocking tracks found in any playlist.")
		return
	}

	Styles.Header.Printf("Found %d blocking track(s) across %d playlist(s):\n\n", totalToMove, len(candidates))
	for i, c := range candidates {
		isLast := i == len(candidates)-1
		Styles.Group.Printf("%s %s ", playlistBranch(isLast), c.name)
		fmt.Printf("(%d)\n", len(c.tracks))
		for j, t := range c.tracks {
			isLastTrack := j == len(c.tracks)-1
			fmt.Printf("%s %s ", trackBranch(isLast, isLastTrack), formatStatus(t.CloudStatus))
			Styles.Title.Printf("%s", t.Name)
			fmt.Printf(" — %s\n", t.Artist)
		}
	}
	fmt.Println()
	fmt.Printf("Destination playlist: %s\n", quarantineDest)
	fmt.Println("(de-duped by persistent ID; tracks stay in your library)")
	fmt.Println()

	if quarantineDryRun {
		Styles.Header.Println("Dry-run — no changes made.")
		return
	}

	Styles.Prompt.Print("Move these tracks now? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}
	fmt.Println()

	movedTotal := 0
	uniqueTracks := map[string]bool{}
	var failed []string
	for i, c := range candidates {
		Styles.Header.Printf("[%d/%d] %s (%d blocker(s))\n", i+1, len(candidates), c.name, len(c.tracks))
		moved, err := applemusic.QuarantineBlockingTracks(c.name, quarantineDest)
		if err != nil {
			Styles.Error.Printf("    failed: %v\n", err)
			failed = append(failed, c.name)
			continue
		}
		for _, t := range moved {
			Styles.Title.Printf("    %s — %s ", t.Artist, t.Name)
			fmt.Printf("[%s]\n", t.CloudStatus)
			uniqueTracks[t.PersistentID] = true
		}
		movedTotal += len(moved)
		Styles.Success.Printf("    moved %d\n", len(moved))
	}

	fmt.Println()
	Styles.Header.Println("Quarantine Summary:")
	Styles.Success.Printf("  Moved:  %d removals across source playlists\n", movedTotal)
	Styles.Success.Printf("  Unique: %d distinct tracks in %s\n", len(uniqueTracks), quarantineDest)
	if len(failed) > 0 {
		Styles.Error.Printf("  Failed playlists: %d\n", len(failed))
		for _, name := range failed {
			Styles.Error.Printf("    - %s\n", name)
		}
		os.Exit(1)
	}
}
