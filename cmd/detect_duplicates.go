package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(detectDuplicatesCmd)
}

var detectDuplicatesCmd = &cobra.Command{
	Use:     "detect-duplicates [directories...]",
	Aliases: []string{"dd"},
	Short:   "Detect duplicate music files in one or more directories",
	Long: `Detect duplicate music files by comparing their content hashes and metadata.
If no directories are provided, it defaults to the $MUSICDIR environment variable.
After detecting duplicates, the user is prompted to delete selected duplicates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine directories to process
		dirs := args
		if len(dirs) == 0 {
			envDir := os.Getenv("MUSICDIR")
			if envDir == "" {
				fmt.Println(Styles.Error.Sprint("No directories provided and $MUSICDIR is not set."))
				return nil
			}
			dirs = []string{envDir}
		}

		// Find duplicates
		duplicates, err := findDuplicatesAcrossDirectories(dirs)
		if err != nil {
			fmt.Println(Styles.Error.Sprintf("Error analyzing files: %v", err))
			return nil
		}

		// Display duplicates and ask to proceed
		if len(duplicates) == 0 {
			fmt.Println(Styles.Header.Sprint("\nNo duplicates found."))
			return nil
		}

		fmt.Println(Styles.Header.Sprint("\nDuplicate files found:"))
		for group, files := range duplicates {
			fmt.Printf("\n%s\n", Styles.Group.Sprint(group))
			for i, file := range files {
				fmt.Printf("  %d. %s\n", i+1, Styles.Title.Sprint(file.Title))
				fmt.Printf("     %s\n", Styles.Path.Sprint(file.Path))
			}
		}

		// Prompt if user wants to proceed with deletion
		if !promptYesNo(Styles.Prompt.Sprint("Would you like to proceed with deletion?")) {
			fmt.Println(Styles.Success.Sprint("No files were deleted."))
			return nil
		}

		// Iterate through all duplicate groups and prompt for deletions
		allToDelete := []string{}
		for group, files := range duplicates {
			fmt.Printf("\n%s\n", Styles.Group.Sprint(group))
			toDelete := promptForDeletions(files)
			for _, index := range toDelete {
				allToDelete = append(allToDelete, files[index].Path)
			}
		}

		// Show the final list of files marked for deletion
		if len(allToDelete) > 0 {
			manageDeletionList(allToDelete)
		} else {
			fmt.Println(Styles.Success.Sprint("No files were marked for deletion."))
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
		fmt.Println(Styles.Error.Sprint("Invalid input. Please enter 'y' or 'n'."))
	}
}

// promptForDeletions prompts the user for the numbers of the files they want to delete
func promptForDeletions(files []MusicFile) []int {
	reader := bufio.NewReader(os.Stdin)
	toDelete := []int{}

	fmt.Println(Styles.Header.Sprint("\nSelect files to delete from the following list:\n"))

	// Print the files in a numbered list with spacing
	for i, file := range files {
		fmt.Printf("  %s %d. %s\n", Styles.Title.Sprint("Title:"), i+1, Styles.Title.Sprint(file.Title))
		fmt.Printf("     %s\n", Styles.Path.Sprint(file.Path))
		fmt.Println()
	}

	for {
		fmt.Print(Styles.Prompt.Sprint("Enter the number(s) of the file(s) you want to delete, separated by commas (or press Enter to skip): "))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// If the user presses Enter without input, skip deletion for this group
		if input == "" {
			break
		}

		// Parse the input
		selections := strings.Split(input, ",")
		validInput := true
		tempToDelete := []int{}

		for _, s := range selections {
			num, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || num < 1 || num > len(files) {
				fmt.Println(Styles.Error.Sprintf("Invalid selection: %s. Please enter valid numbers.", s))
				validInput = false
				break
			}
			tempToDelete = append(tempToDelete, num-1) // Adjust for zero-based index
		}

		// If all input is valid, store the selections and break
		if validInput {
			toDelete = tempToDelete
			break
		}
		// Otherwise, re-prompt
		fmt.Println(Styles.Error.Sprint("Please try again with valid input."))
	}

	return toDelete
}

// manageDeletionList handles the final confirmation and allows the user to modify the deletion list
func manageDeletionList(files []string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Display the current list of files to delete
		fmt.Println(Styles.Header.Sprint("\nYou have marked the following files for deletion:"))
		for i, path := range files {
			fmt.Printf("  %d. %s\n", i+1, Styles.Path.Sprint(path))
		}

		// Prompt user to confirm or modify the list
		for {
			fmt.Print(Styles.Prompt.Sprint("\nEnter 'd' to delete these files, 'r' to remove files from the list, or 'c' to cancel: "))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			if input == "d" {
				// Confirm deletion and delete files
				for _, path := range files {
					fmt.Printf("Deleting: %s\n", Styles.Path.Sprint(path))
					err := os.Remove(path)
					if err != nil {
						fmt.Printf("%s %s: %v\n", Styles.Error.Sprint("Failed to delete"), Styles.Path.Sprint(path), err)
					} else {
						fmt.Printf("%s %s\n", Styles.Success.Sprint("Successfully deleted:"), Styles.Path.Sprint(path))
					}
				}
				return
			} else if input == "r" {
				// Allow user to remove files from the list
				removeFilesFromList(&files)
				break
			} else if input == "c" {
				// Cancel the deletion process
				fmt.Println(Styles.Success.Sprint("No files were deleted."))
				return
			} else {
				// Invalid input, re-prompt
				fmt.Println(Styles.Error.Sprint("Invalid input. Please enter 'd', 'r', or 'c'."))
			}
		}

		// If the deletion list becomes empty, exit
		if len(files) == 0 {
			fmt.Println(Styles.Success.Sprint("No files are marked for deletion."))
			return
		}
	}
}

// removeFilesFromList allows the user to remove files from the deletion list
func removeFilesFromList(files *[]string) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(Styles.Prompt.Sprint("\nEnter the number(s) of the file(s) you want to remove from the list, separated by commas: "))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// Parse the input and remove the selected files
		selections := strings.Split(input, ",")
		toRemove := map[int]bool{}
		isValid := true

		for _, s := range selections {
			num, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || num < 1 || num > len(*files) {
				fmt.Println(Styles.Error.Sprintf("Invalid selection: %s. Please enter valid numbers.", s))
				isValid = false
				break
			}
			toRemove[num-1] = true
		}

		// If the input is invalid, re-prompt
		if !isValid {
			continue
		}

		// Create a new list excluding the removed files
		newList := []string{}
		for i, path := range *files {
			if !toRemove[i] {
				newList = append(newList, path)
			}
		}

		// Update the deletion list
		*files = newList
		if len(*files) == 0 {
			fmt.Println(Styles.Success.Sprint("No files are marked for deletion."))
		}
		return
	}
}
