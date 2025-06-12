package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "foe",
	Short: "foe CLI for managing music files",
	Long: `foe is a CLI tool designed to help manage and organize your music collection.
Use "foe help" to see all commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available commands:")
		for _, c := range cmd.Commands() { // Use the current command to list subcommands
			if !c.Hidden {
				fmt.Printf("  %s - %s\n", c.Name(), c.Short)
			}
		}
		fmt.Println("\nUse 'foe [command] --help' for more information about a command.")
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
