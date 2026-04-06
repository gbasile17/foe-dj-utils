package cmd

import (
	"fmt"
	"os"

	"github.com/gbasile17/foe/dj-utils/internal/music"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(artistSearchCmd)
}

var artistSearchCmd = &cobra.Command{
	Use:     "artist-search [directories...] [query]",
	Aliases: []string{"as"},
	Short:   "Search for all tracks containing the given string in the artist tag",
	Long: `Searches the specified directories for audio files and returns all tracks
that contain the given string in the artist tag (case-insensitive). If no directories are provided, it defaults to the $MUSICDIR environment variable.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[len(args)-1]
		dirs := args[:len(args)-1]
		if len(dirs) == 0 {
			envDir := os.Getenv("MUSICDIR")
			if envDir == "" {
				Styles.Error.Println("No directories provided and $MUSICDIR is not set.")
				return nil
			}
			dirs = []string{envDir}
		}

		results, err := music.SearchArtists(dirs, query)
		if err != nil {
			Styles.Error.Printf("Error searching artists: %v\n", err)
			return nil
		}

		if len(results) == 0 {
			Styles.Header.Printf("No tracks found with artists containing '%s'.\n", query)
		} else {
			Styles.Header.Printf("\nTracks with artists containing '%s':\n\n", query)
			for _, result := range results {
				fmt.Printf("%s %s\n", Styles.Title.Sprint("Title:"), Styles.Title.Sprint(result.Title))
				fmt.Printf("%s %s\n", Styles.Group.Sprint("Artist:"), Styles.Group.Sprint(result.Artist))
				fmt.Printf(" %s\n\n", Styles.Path.Sprint(result.Path))
			}
		}

		return nil
	},
}
