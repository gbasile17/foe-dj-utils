package cmd

import (
	"fmt"
	"os"

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
				fmt.Println(Styles.Error.Sprint("No directories provided and $MUSICDIR is not set."))
				return nil
			}
			dirs = []string{envDir}
		}

		// Perform the search
		results, err := searchArtists(dirs, query)
		if err != nil {
			fmt.Println(Styles.Error.Sprintf("Error searching artists: %v", err))
			return nil
		}

		// Display results
		if len(results) == 0 {
			fmt.Println(Styles.Header.Sprintf("No tracks found with artists containing '%s'.", query))
		} else {
			fmt.Println(Styles.Header.Sprintf("\nTracks with artists containing '%s':\n", query))
			for _, result := range results {
				fmt.Printf("%s %s\n", Styles.Title.Sprint("Title:"), Styles.Title.Sprint(result.Title))
				fmt.Printf("%s %s\n", Styles.Group.Sprint("Artist:"), Styles.Group.Sprint(result.Artist))
				fmt.Printf(" %s\n", Styles.Path.Sprint(result.Path))
				fmt.Println()
			}
		}

		return nil
	},
}
