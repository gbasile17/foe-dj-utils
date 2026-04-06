package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/gbasile17/foe/dj-utils/internal/fileutil"
	"github.com/spf13/cobra"
)

var appleExportPlaylistCmd = &cobra.Command{
	Use:     "apple-export-playlist [playlist-name] [output-dir]",
	Aliases: []string{"aep"},
	Short:   "Export an Apple Music playlist with all audio files",
	Long: `Exports a playlist from Apple Music by copying all associated audio files
to a specified directory. This is useful for backing up playlists or preparing
files for use with other DJ software.

Files are copied with a track number prefix to maintain playlist order.
An M3U playlist file is also generated for compatibility with other players.

Example:
  foe apple-export-playlist "My Playlist" ./export
  foe aep "House" ~/Music/exports/house`,
	Args: cobra.ExactArgs(2),
	Run:  runAppleExportPlaylist,
}

var (
	exportPlaylistFlat    bool
	exportPlaylistM3U     bool
	exportPlaylistSkipDRM bool
)

func init() {
	rootCmd.AddCommand(appleExportPlaylistCmd)
	appleExportPlaylistCmd.Flags().BoolVarP(&exportPlaylistFlat, "flat", "f", true, "Copy all files to a single directory (default: true)")
	appleExportPlaylistCmd.Flags().BoolVarP(&exportPlaylistM3U, "m3u", "m", true, "Generate M3U playlist file (default: true)")
	appleExportPlaylistCmd.Flags().BoolVarP(&exportPlaylistSkipDRM, "skip-drm", "s", true, "Skip DRM-protected files with a warning (default: true)")
}

func runAppleExportPlaylist(cmd *cobra.Command, args []string) {
	playlistName := args[0]
	outputDir := args[1]

	if !applemusic.IsAvailable() {
		Styles.Error.Println("Error: Apple Music app is not available on this system")
		os.Exit(1)
	}

	exists, err := applemusic.PlaylistExists(playlistName)
	if err != nil {
		Styles.Error.Printf("Error checking playlist: %v\n", err)
		os.Exit(1)
	}
	if !exists {
		Styles.Error.Printf("Playlist '%s' not found.\n", playlistName)
		fmt.Println("\nAvailable playlists:")
		playlists, _ := applemusic.GetUserPlaylists()
		for _, p := range playlists {
			fmt.Printf("  - %s\n", p)
		}
		os.Exit(1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		Styles.Error.Printf("Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	Styles.Header.Printf("Exporting playlist: %s\n", playlistName)
	fmt.Printf("Output directory: %s\n\n", outputDir)

	tracks, err := applemusic.GetPlaylistTracks(playlistName)
	if err != nil {
		Styles.Error.Printf("Failed to get playlist tracks: %v\n", err)
		os.Exit(1)
	}

	if len(tracks) == 0 {
		Styles.Error.Println("Playlist is empty.")
		return
	}

	fmt.Printf("Found %d tracks\n\n", len(tracks))

	var m3uEntries []string
	var successCount, skipCount, errorCount int

	for i, track := range tracks {
		trackNum := i + 1

		if track.IsOrphaned {
			Styles.Error.Printf("%3d. [MISSING] %s - %s\n", trackNum, track.Artist, track.Name)
			skipCount++
			continue
		}

		if strings.HasSuffix(strings.ToLower(track.Location), ".m4p") {
			if exportPlaylistSkipDRM {
				Styles.Error.Printf("%3d. [DRM] %s - %s\n", trackNum, track.Artist, track.Name)
				skipCount++
				continue
			}
		}

		ext := filepath.Ext(track.Location)
		safeFilename := fileutil.SanitizeFilename(fmt.Sprintf("%03d - %s - %s%s",
			trackNum, track.Artist, track.Name, ext))
		destPath := filepath.Join(outputDir, safeFilename)

		if err := fileutil.CopyFile(track.Location, destPath); err != nil {
			Styles.Error.Printf("%3d. [ERROR] %s - %s: %v\n", trackNum, track.Artist, track.Name, err)
			errorCount++
			continue
		}

		Styles.Success.Printf("%3d. ", trackNum)
		Styles.Title.Printf("%s", track.Name)
		fmt.Printf(" - %s\n", track.Artist)
		successCount++

		m3uEntries = append(m3uEntries, safeFilename)
	}

	if exportPlaylistM3U && len(m3uEntries) > 0 {
		m3uPath := filepath.Join(outputDir, fileutil.SanitizeFilename(playlistName)+".m3u")
		if err := fileutil.WriteM3U(m3uPath, m3uEntries); err != nil {
			Styles.Error.Printf("\nFailed to write M3U file: %v\n", err)
		} else {
			fmt.Printf("\nGenerated playlist file: %s\n", m3uPath)
		}
	}

	fmt.Println()
	Styles.Header.Println("Export Summary:")
	Styles.Success.Printf("  Copied: %d tracks\n", successCount)
	if skipCount > 0 {
		Styles.Error.Printf("  Skipped: %d tracks (missing or DRM)\n", skipCount)
	}
	if errorCount > 0 {
		Styles.Error.Printf("  Errors: %d tracks\n", errorCount)
	}
}
