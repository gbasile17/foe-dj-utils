package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/spf13/cobra"
)

var appleResolveDuplicatesCmd = &cobra.Command{
	Use:     "apple-resolve-duplicates",
	Aliases: []string{"ard"},
	Short:   "Inspect tracks flagged as duplicate by iCloud and find the canonical copy",
	Long: `Walks every track in user playlists with cloud status "duplicate"
and searches the library for the canonical copy iCloud is matching against.

For each duplicate, the cleaned title (DJ key/BPM prefix and "(... Mix)"
suffix stripped) is matched against the library by artist + title substring,
then filtered by duration tolerance.

Read-only — does not change anything. Use the output to decide whether to
keep the prefixed/duplicated copy or the canonical copy.

Examples:
  foe apple-resolve-duplicates --inspect
  foe ard --inspect --tolerance 5`,
	Run: runAppleResolveDuplicates,
}

var (
	resolveInspect   bool
	resolveTolerance float64
	resolveSkip      []string
)

func init() {
	rootCmd.AddCommand(appleResolveDuplicatesCmd)
	appleResolveDuplicatesCmd.Flags().BoolVar(&resolveInspect, "inspect", false, "Print candidate canonical for each duplicate (required for now)")
	appleResolveDuplicatesCmd.Flags().Float64Var(&resolveTolerance, "tolerance", 3.0, "Duration tolerance in seconds when matching candidates")
	appleResolveDuplicatesCmd.Flags().StringSliceVar(&resolveSkip, "skip", []string{"_lib", "_block"}, "Comma-separated playlists to skip when collecting duplicates")
}

// djPrefixRe matches "<key> - <bpm> - " at the start of a title:
//   3A - 122 - You & Me
//   12B - 125 - Promised Land
var djPrefixRe = regexp.MustCompile(`^\s*\d{1,2}[AB]\s*-\s*\d{2,3}\s*-\s*`)

// mixSuffixRe matches a trailing "(... Mix)" or "(Extended)" / "(Original)" etc.,
// optionally with feature info inside. We strip these because iCloud's matcher
// often normalizes them away.
var mixSuffixRe = regexp.MustCompile(`(?i)\s*\((?:[^()]*?(?:mix|extended|original|edit|version|remix)[^()]*?)\)\s*$`)

// cleanTitle strips DJ prefix and "(... Mix)" suffix.
func cleanTitle(title string) string {
	t := djPrefixRe.ReplaceAllString(title, "")
	t = mixSuffixRe.ReplaceAllString(t, "")
	return strings.TrimSpace(t)
}

func runAppleResolveDuplicates(cmd *cobra.Command, args []string) {
	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	if !resolveInspect {
		Styles.Error.Println("Error: --inspect is required (no resolution actions implemented yet)")
		os.Exit(1)
	}

	all, err := applemusic.GetUserPlaylists()
	if err != nil {
		Styles.Error.Printf("Failed to list playlists: %v\n", err)
		os.Exit(1)
	}

	skip := map[string]bool{}
	for _, s := range resolveSkip {
		s = strings.TrimSpace(s)
		if s != "" {
			skip[s] = true
		}
	}

	// Collect duplicate-flagged tracks across all (non-skipped) playlists,
	// de-duped by persistent ID.
	type dupTrack struct {
		applemusic.BlockingTrack
		Sources []string // playlist names this duplicate appeared in
	}
	dupsByPID := map[string]*dupTrack{}
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
				dupsByPID[b.PersistentID] = &dupTrack{
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

	Styles.Header.Printf("Inspecting %d unique duplicate-flagged track(s) (tolerance ±%.1fs):\n\n", len(dupsByPID), resolveTolerance)

	for _, d := range dupsByPID {
		// Get the duplicate's own duration so we can filter candidates.
		details, err := applemusic.GetTrackDetails(d.PersistentID)
		if err != nil {
			Styles.Error.Printf("Failed to fetch details for %q: %v\n", d.Name, err)
			continue
		}

		cleanedTitle := cleanTitle(d.Name)

		Styles.Group.Printf("● %s — %s\n", d.Name, d.Artist)
		fmt.Printf("    in: %s\n", strings.Join(d.Sources, ", "))
		fmt.Printf("    cleaned title: %q\n", cleanedTitle)
		fmt.Printf("    %s, %s, %s, %s\n",
			formatDuration(details.Duration),
			formatBytes(details.Size),
			details.Kind,
			locationLabel(details.Location))

		// Search library for candidate canonicals.
		candidates, err := applemusic.FindLibraryTracksByTitleArtist(cleanedTitle, d.Artist)
		if err != nil {
			Styles.Error.Printf("    search error: %v\n", err)
			continue
		}

		// Filter: not the same persistent ID, duration within tolerance.
		var filtered []applemusic.LibraryTrack
		for _, c := range candidates {
			if c.PersistentID == d.PersistentID {
				continue
			}
			if absFloat(c.Duration-details.Duration) > resolveTolerance {
				continue
			}
			filtered = append(filtered, c)
		}

		if len(filtered) == 0 {
			Styles.Error.Println("    → no canonical candidate found")
			fmt.Println()
			continue
		}

		for _, c := range filtered {
			Styles.Success.Printf("    → %s — %s\n", c.Name, c.Artist)
			fmt.Printf("        %s, %s, %s, %s, cloud=%s\n",
				formatDuration(c.Duration),
				formatBytes(c.Size),
				c.Kind,
				locationLabel(c.Location),
				c.CloudStatus)
		}
		fmt.Println()
	}
}

func formatDuration(seconds float64) string {
	m := int(seconds) / 60
	s := int(seconds) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
	)
	switch {
	case b >= mb:
		return fmt.Sprintf("%.1fMB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.1fKB", float64(b)/kb)
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func locationLabel(loc string) string {
	if loc == "" {
		return "(cloud only)"
	}
	return loc
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
