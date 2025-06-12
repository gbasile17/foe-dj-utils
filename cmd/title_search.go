package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(titleSearchCmd)
}

var titleSearchCmd = &cobra.Command{
	Use:     "title-search [directories...] [query]",
	Aliases: []string{"ts"},
	Short:   "Search for all audio file titles containing the given string",
	Long: `Searches the specified directories for audio files and returns all titles
that contain the given string (case-insensitive). If no directories are provided, it defaults to the $MUSICDIR environment variable.`,
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
		results, err := searchTitles(dirs, query)
		if err != nil {
			fmt.Println(Styles.Error.Sprintf("Error searching titles: %v", err))
			return nil
		}

		// Display results
		if len(results) == 0 {
			fmt.Println(Styles.Header.Sprintf("No titles found containing '%s'.", query))
		} else {
			fmt.Println(Styles.Header.Sprintf("\nTitles containing '%s':\n", query))
			for _, result := range results {
				fmt.Printf("%s %s\n", Styles.Title.Sprint("Title:"), Styles.Title.Sprint(result.Title))
				fmt.Printf(" %s\n", Styles.Path.Sprint(result.Path))
				fmt.Println()
			}
		}

		return nil
	},
}
