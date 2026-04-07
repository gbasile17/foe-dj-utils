package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/applemusic"
	"github.com/gbasile17/foe/dj-utils/internal/music"
	"github.com/spf13/cobra"
)

var detectAppleCleanup bool

func init() {
	rootCmd.AddCommand(detectDuplicatesCmd)
	detectDuplicatesCmd.Flags().BoolVarP(&detectAppleCleanup, "apple-cleanup", "a", false, "Clean up orphaned tracks in Apple Music after deleting files")
}

var detectDuplicatesCmd = &cobra.Command{
	Use:     "detect-duplicates [directories...]",
	Aliases: []string{"dd"},
	Short:   "Detect duplicate music files in one or more directories",
	Long: `Detect duplicate music files by comparing their content hashes and metadata.
If no directories are provided, it defaults to the $MUSICDIR environment variable.
After detecting duplicates, the user is prompted to delete selected duplicates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dirs := args
		if len(dirs) == 0 {
			envDir := os.Getenv("MUSICDIR")
			if envDir == "" {
				Styles.Error.Println("No directories provided and $MUSICDIR is not set.")
				return nil
			}
			dirs = []string{envDir}
		}

		duplicates, err := music.FindDuplicates(dirs)
		if err != nil {
			Styles.Error.Printf("Error analyzing files: %v\n", err)
			return nil
		}

		if len(duplicates) == 0 {
			Styles.Header.Println("\nNo duplicates found.")
			return nil
		}

		Styles.Header.Println("\nDuplicate files found:")
		for group, files := range duplicates {
			fmt.Printf("\n%s\n", Styles.Group.Sprint(group))
			for i, file := range files {
				fmt.Printf("  %d. %s\n", i+1, Styles.Title.Sprint(file.Title))
				fmt.Printf("     %s\n", Styles.Path.Sprint(file.Path))
			}
		}

		if !promptYesNo(Styles.Prompt.Sprint("Would you like to proceed with deletion?")) {
			Styles.Success.Println("No files were deleted.")
			return nil
		}

		var allToDelete []string
		for group, files := range duplicates {
			fmt.Printf("\n%s\n", Styles.Group.Sprint(group))
			toDelete := promptForDeletionsMusicFile(files)
			for _, index := range toDelete {
				allToDelete = append(allToDelete, files[index].Path)
			}
		}

		if len(allToDelete) > 0 {
			manageDeletionList(allToDelete)
		} else {
			Styles.Success.Println("No files were marked for deletion.")
		}

		return nil
	},
}

// promptYesNo prompts the user for a yes or no answer
func promptYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (y/n): ", question)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		if input == "y" || input == "yes" {
			return true
		} else if input == "n" || input == "no" {
			return false
		}
		Styles.Error.Println("Invalid input. Please enter 'y' or 'n'.")
	}
}

// promptForDeletionsMusicFile prompts the user for the numbers of the files they want to delete
func promptForDeletionsMusicFile(files []music.File) []int {
	reader := bufio.NewReader(os.Stdin)
	var toDelete []int

	Styles.Header.Println("\nSelect files to delete from the following list:")

	for i, file := range files {
		fmt.Printf("  %s %d. %s\n", Styles.Title.Sprint("Title:"), i+1, Styles.Title.Sprint(file.Title))
		fmt.Printf("     %s\n\n", Styles.Path.Sprint(file.Path))
	}

	for {
		Styles.Prompt.Print("Enter the number(s) of the file(s) you want to delete, separated by commas (or press Enter to skip): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			break
		}

		selections := strings.Split(input, ",")
		validInput := true
		var tempToDelete []int

		for _, s := range selections {
			num, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || num < 1 || num > len(files) {
				Styles.Error.Printf("Invalid selection: %s. Please enter valid numbers.\n", s)
				validInput = false
				break
			}
			tempToDelete = append(tempToDelete, num-1)
		}

		if validInput {
			toDelete = tempToDelete
			break
		}
		Styles.Error.Println("Please try again with valid input.")
	}

	return toDelete
}

// manageDeletionList handles the final confirmation and allows the user to modify the deletion list
func manageDeletionList(files []string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		Styles.Header.Println("\nYou have marked the following files for deletion:")
		for i, path := range files {
			fmt.Printf("  %d. %s\n", i+1, Styles.Path.Sprint(path))
		}

		for {
			Styles.Prompt.Print("\nEnter 'd' to delete these files, 'r' to remove files from the list, or 'c' to cancel: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input == "d" {
				var deletedPaths []string
				for _, path := range files {
					fmt.Printf("Deleting: %s\n", Styles.Path.Sprint(path))
					err := os.Remove(path)
					if err != nil {
						Styles.Error.Printf("Failed to delete %s: %v\n", path, err)
					} else {
						Styles.Success.Printf("Successfully deleted: %s\n", path)
						deletedPaths = append(deletedPaths, path)
					}
				}

				if len(deletedPaths) > 0 && detectAppleCleanup && applemusic.IsAvailable() {
					cleanupAppleMusicOrphans()
				}
				return
			} else if input == "r" {
				removeFilesFromList(&files)
				break
			} else if input == "c" {
				Styles.Success.Println("No files were deleted.")
				return
			} else {
				Styles.Error.Println("Invalid input. Please enter 'd', 'r', or 'c'.")
			}
		}

		if len(files) == 0 {
			Styles.Success.Println("No files are marked for deletion.")
			return
		}
	}
}

// removeFilesFromList allows the user to remove files from the deletion list
func removeFilesFromList(files *[]string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		Styles.Prompt.Print("\nEnter the number(s) of the file(s) you want to remove from the list, separated by commas: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		selections := strings.Split(input, ",")
		toRemove := make(map[int]bool)
		isValid := true

		for _, s := range selections {
			num, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || num < 1 || num > len(*files) {
				Styles.Error.Printf("Invalid selection: %s. Please enter valid numbers.\n", s)
				isValid = false
				break
			}
			toRemove[num-1] = true
		}

		if !isValid {
			continue
		}

		var newList []string
		for i, path := range *files {
			if !toRemove[i] {
				newList = append(newList, path)
			}
		}

		*files = newList
		if len(*files) == 0 {
			Styles.Success.Println("No files are marked for deletion.")
		}
		return
	}
}

// cleanupAppleMusicOrphans finds and removes orphaned tracks from Apple Music
func cleanupAppleMusicOrphans() {
	Styles.Header.Println("\nScanning Apple Music for orphaned tracks...")

	orphans, err := applemusic.FindOrphanedTracks(func(checked, total int) {
		fmt.Printf("\rChecking tracks: %d/%d", checked, total)
	})
	fmt.Println()

	if err != nil {
		Styles.Error.Printf("Failed to scan Apple Music: %v\n", err)
		return
	}

	if len(orphans) == 0 {
		Styles.Success.Println("No orphaned tracks found in Apple Music.")
		return
	}

	Styles.Header.Printf("\nFound %d orphaned tracks:\n", len(orphans))
	for i, track := range orphans {
		fmt.Printf("  %d. %s - %s\n", i+1, track.Artist, track.Name)
	}

	if !promptYesNo(Styles.Prompt.Sprint("\nRemove these from Apple Music?")) {
		fmt.Println("Skipped Apple Music cleanup.")
		return
	}

	var persistentIDs []string
	for _, t := range orphans {
		persistentIDs = append(persistentIDs, t.PersistentID)
	}

	if err = applemusic.DeleteTracksByPersistentID(persistentIDs); err != nil {
		Styles.Error.Printf("Failed to delete tracks: %v\n", err)
		return
	}

	Styles.Success.Printf("Removed %d orphaned tracks from Apple Music.\n", len(orphans))
}
