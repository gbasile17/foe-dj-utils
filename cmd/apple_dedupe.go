package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/gbasile17/foe/dj-utils/internal/fileutil"
	"github.com/spf13/cobra"
)

var appleDedupeCmd = &cobra.Command{
	Use:     "apple-dedupe",
	Aliases: []string{"adp"},
	Short:   "Quarantine duplicate-flagged files outside the library",
	Long: `Finds every track in user playlists with cloud status "duplicate"
(iCloud has matched it against another copy and refuses to sync it), copies
the underlying file to ~/Music/duplicates (flat), removes the track from the
Apple Music library, and deletes the original file on disk.

Use this when you want to keep your lossless source files outside the library
while letting iCloud sync the canonical copy. You can re-add the quarantined
files later (after iCloud has finished propagating the deletion) if you want
them back in the library.

Skips _lib and _block by default.

Examples:
  foe apple-dedupe --dry-run
  foe apple-dedupe
  foe adp --dest ~/Music/dj-duplicates`,
	Run: runAppleDedupe,
}

var (
	dedupeDryRun bool
	dedupeDest   string
	dedupeSkip   []string
)

func init() {
	rootCmd.AddCommand(appleDedupeCmd)
	dedupeDest = filepath.Join(os.Getenv("HOME"), "Music", "duplicates")
	appleDedupeCmd.Flags().BoolVarP(&dedupeDryRun, "dry-run", "n", false, "Report what would be quarantined, don't change anything")
	appleDedupeCmd.Flags().StringVar(&dedupeDest, "dest", dedupeDest, "Destination directory for quarantined files")
	appleDedupeCmd.Flags().StringSliceVar(&dedupeSkip, "skip", []string{"_lib", "_block"}, "Comma-separated playlists to skip when collecting duplicates")
}

func runAppleDedupe(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	all, err := applemusic.GetUserPlaylists()
	if err != nil {
		Styles.Error.Printf("Failed to list playlists: %v\n", err)
		os.Exit(1)
	}

	skip := map[string]bool{}
	for _, s := range dedupeSkip {
		s = strings.TrimSpace(s)
		if s != "" {
			skip[s] = true
		}
	}

	// Collect duplicate-flagged tracks across all (non-skipped) playlists,
	// de-duped by persistent ID.
	type dupCandidate struct {
		applemusic.BlockingTrack
		Sources []string
	}
	dupsByPID := map[string]*dupCandidate{}
	for _, name := range all {
		if skip[name] {
			continue
		}
		blockers, err := applemusic.ListBlockingTracks(name)
		if err != nil {
			Styles.Error.Printf("Failed to list blockers in %q: %v\n", name, err)
			continue
		}
		for _, b := range blockers {
			if b.CloudStatus != "duplicate" {
				continue
			}
			if existing, ok := dupsByPID[b.PersistentID]; ok {
				existing.Sources = append(existing.Sources, name)
			} else {
				dupsByPID[b.PersistentID] = &dupCandidate{
					BlockingTrack: b,
					Sources:       []string{name},
				}
			}
		}
	}

	if len(dupsByPID) == 0 {
		Styles.Success.Println("No duplicate-flagged tracks found.")
		return
	}

	// Resolve file locations up front so we can show them in the preview and
	// fail loudly if any are unreachable.
	type plan struct {
		PID      string
		Name     string
		Artist   string
		Sources  []string
		Source   string // current file path on disk
		Dest     string // computed destination path
		Skip     string // non-empty = reason to skip this track
	}
	var plans []plan
	usedDest := map[string]bool{}
	for _, d := range dupsByPID {
		details, err := applemusic.GetTrackDetails(d.PersistentID)
		if err != nil {
			plans = append(plans, plan{
				PID: d.PersistentID, Name: d.Name, Artist: d.Artist,
				Sources: d.Sources,
				Skip:    fmt.Sprintf("could not fetch details: %v", err),
			})
			continue
		}
		if details.Location == "" {
			plans = append(plans, plan{
				PID: d.PersistentID, Name: d.Name, Artist: d.Artist,
				Sources: d.Sources,
				Skip:    "track has no on-disk location (cloud-only)",
			})
			continue
		}
		dest := uniqueDest(dedupeDest, filepath.Base(details.Location), usedDest)
		usedDest[dest] = true
		plans = append(plans, plan{
			PID: d.PersistentID, Name: d.Name, Artist: d.Artist,
			Sources: d.Sources,
			Source:  details.Location,
			Dest:    dest,
		})
	}

	// Preview tree.
	Styles.Header.Printf("Quarantining %d duplicate-flagged track(s):\n", len(plans))
	fmt.Printf("Destination: %s\n\n", dedupeDest)
	for _, p := range plans {
		Styles.Group.Printf("● %s — %s\n", p.Name, p.Artist)
		fmt.Printf("    in: %s\n", strings.Join(p.Sources, ", "))
		if p.Skip != "" {
			Styles.Error.Printf("    skip: %s\n", p.Skip)
			continue
		}
		fmt.Printf("    from: %s\n", p.Source)
		fmt.Printf("    to:   %s\n", p.Dest)
	}
	fmt.Println()

	if dedupeDryRun {
		Styles.Header.Println("Dry-run — no changes made.")
		return
	}

	Styles.Prompt.Print("This will copy + delete from library + delete original on disk. Continue? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Aborted.")
		return
	}
	fmt.Println()

	if err := os.MkdirAll(dedupeDest, 0755); err != nil {
		Styles.Error.Printf("Failed to create destination dir: %v\n", err)
		os.Exit(1)
	}

	var (
		quarantined int
		skipped     int
		failed      []string
	)
	for i, p := range plans {
		Styles.Header.Printf("[%d/%d] %s — %s\n", i+1, len(plans), p.Name, p.Artist)
		if p.Skip != "" {
			Styles.Error.Printf("    skipped: %s\n", p.Skip)
			skipped++
			continue
		}

		// Step 1: copy
		if err := fileutil.CopyFile(p.Source, p.Dest); err != nil {
			Styles.Error.Printf("    copy failed: %v\n", err)
			failed = append(failed, p.Name)
			continue
		}
		// Verify copy size matches.
		srcInfo, err1 := os.Stat(p.Source)
		dstInfo, err2 := os.Stat(p.Dest)
		if err1 != nil || err2 != nil || srcInfo.Size() != dstInfo.Size() {
			Styles.Error.Printf("    copy verification failed (sizes differ); leaving original in place\n")
			_ = os.Remove(p.Dest)
			failed = append(failed, p.Name)
			continue
		}
		Styles.Success.Printf("    copied (%s)\n", formatBytes(srcInfo.Size()))

		// Step 2: remove track from Music library
		if err := applemusic.DeleteTrackFromLibrary(p.PID); err != nil {
			Styles.Error.Printf("    library delete failed: %v\n", err)
			failed = append(failed, p.Name)
			continue
		}
		Styles.Success.Printf("    removed from library\n")

		// Step 3: ensure original file is gone from disk. Music may or may
		// not move it to Trash depending on user prefs; force-remove.
		if _, err := os.Stat(p.Source); err == nil {
			if err := os.Remove(p.Source); err != nil {
				Styles.Error.Printf("    on-disk delete failed: %v (file at %s)\n", err, p.Source)
				// Library entry is already gone; don't count as full failure.
			} else {
				Styles.Success.Printf("    deleted original on disk\n")
			}
		} else {
			Styles.Success.Printf("    original already gone from disk (Music moved it)\n")
		}

		quarantined++
	}

	fmt.Println()
	Styles.Header.Println("Dedupe Summary:")
	Styles.Success.Printf("  Quarantined: %d\n", quarantined)
	if skipped > 0 {
		Styles.Error.Printf("  Skipped:     %d\n", skipped)
	}
	if len(failed) > 0 {
		Styles.Error.Printf("  Failed:      %d\n", len(failed))
		for _, n := range failed {
			Styles.Error.Printf("    - %s\n", n)
		}
		os.Exit(1)
	}
}

// uniqueDest returns destDir/basename, or destDir/<stem>-N<ext> if the flat
// name already collides with another planned copy.
func uniqueDest(destDir, basename string, used map[string]bool) string {
	candidate := filepath.Join(destDir, basename)
	if !used[candidate] {
		return candidate
	}
	ext := filepath.Ext(basename)
	stem := strings.TrimSuffix(basename, ext)
	for i := 2; ; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if !used[candidate] {
			return candidate
		}
	}
}
