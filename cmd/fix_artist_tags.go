package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/gbasile17/foe/dj-utils/internal/audiotag"
	"github.com/spf13/cobra"
)

// ArtistFixResult represents a file that needs artist tag fixing
type ArtistFixResult struct {
	Path         string
	OriginalArtist string
	CleanedArtist  string
	Format       audiotag.AudioFormat
}

var fixArtistTagsCmd = &cobra.Command{
	Use:   "fix-artist-tags [directory]",
	Short: "Fix artist tags that contain number prefixes",
	Long: `Recursively scan a directory for audio files with artist tags containing
number prefixes (e.g., "101. CoolTasty" -> "CoolTasty") and optionally fix them.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		directory := args[0]
		
		// Validate directory exists
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			color.Red("Directory does not exist: %s", directory)
			return
		}

		fmt.Printf("Scanning directory: %s\n", directory)
		fmt.Println("Looking for artist tags with number prefixes...")

		// Find files with artist tags that need fixing
		filesToFix, err := findArtistTagsToFix(directory)
		if err != nil {
			color.Red("Error scanning files: %v", err)
			return
		}

		if len(filesToFix) == 0 {
			color.Green("No artist tags found that need fixing!")
			return
		}

		// Print report
		printArtistFixReport(filesToFix)

		// Prompt user to apply fixes
		if promptUser("Would you like to fix these artist tags?") {
			applyArtistFixes(filesToFix)
		}
	},
}

func init() {
	rootCmd.AddCommand(fixArtistTagsCmd)
}

// findArtistTagsToFix recursively finds audio files with artist tags that have number prefixes
func findArtistTagsToFix(dir string) ([]ArtistFixResult, error) {
	var results []ArtistFixResult
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	// Worker function to process files
	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !audiotag.IsAudioFile(path) {
				continue
			}

			result, shouldFix := analyzeArtistTag(path)
			if shouldFix {
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}
	}

	// Start worker goroutines
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processFile()
	}

	// Walk directory and send file paths
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileChan <- path
		}
		return nil
	})

	if err != nil {
		close(fileChan)
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	close(fileChan)
	wg.Wait()

	return results, nil
}

// analyzeArtistTag checks if an artist tag needs fixing and returns the cleaned version
func analyzeArtistTag(filePath string) (ArtistFixResult, bool) {
	// Detect format
	format := audiotag.DetectFormat(filePath)
	result := ArtistFixResult{
		Path:   filePath,
		Format: format,
	}

	// Read tags using our audiotag library
	tags, err := audiotag.ReadTags(filePath)
	if err != nil {
		// Skip files we can't read
		return result, false
	}

	result.OriginalArtist = tags.Artist
	
	// Check if artist needs fixing and get cleaned version
	cleanedArtist, needsFix := cleanArtistTag(tags.Artist)
	if needsFix {
		result.CleanedArtist = cleanedArtist
		return result, true
	}

	return result, false
}

// cleanArtistTag removes number prefixes from artist names
func cleanArtistTag(artist string) (string, bool) {
	if artist == "" {
		return artist, false
	}

	original := artist
	artist = strings.TrimSpace(artist)

	// Regex to match number prefixes like "101. ", "1. ", "42. "
	// Matches: digit(s) followed by a period and space
	numberPrefixRegex := regexp.MustCompile(`^\d+\.\s+`)
	
	if numberPrefixRegex.MatchString(artist) {
		cleaned := numberPrefixRegex.ReplaceAllString(artist, "")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned != "" {
			return cleaned, true
		}
	}

	return original, false
}

// printArtistFixReport displays the files that need artist tag fixes
func printArtistFixReport(fixes []ArtistFixResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	color.Yellow("ARTIST TAG FIX REPORT")
	fmt.Println(strings.Repeat("=", 50))

	color.Yellow("\nFOUND %d files with artist tags that need fixing:\n", len(fixes))
	fmt.Println(strings.Repeat("-", 50))

	for i, fix := range fixes {
		fmt.Printf("%d. File: %s (%s)\n", i+1, filepath.Base(fix.Path), fix.Format)
		color.Red("   Original: %s", fix.OriginalArtist)
		color.Green("   Fixed:    %s", fix.CleanedArtist)
		fmt.Println()
	}

	fmt.Printf("Summary: %d artist tags to fix\n", len(fixes))
}

// applyArtistFixes writes the corrected artist tags to files
func applyArtistFixes(fixes []ArtistFixResult) {
	fmt.Println("\nApplying artist tag fixes...")
	
	successCount := 0
	for i, fix := range fixes {
		fmt.Printf("Fixing %d/%d: %s\n", i+1, len(fixes), filepath.Base(fix.Path))
		
		err := updateArtistTag(fix.Path, fix.CleanedArtist)
		if err != nil {
			color.Red("  Error: %v", err)
		} else {
			color.Green("  Fixed: %s -> %s", fix.OriginalArtist, fix.CleanedArtist)
			successCount++
		}
	}
	
	if successCount == len(fixes) {
		color.Green("\n✅ All artist tags fixed successfully!")
	} else {
		color.Yellow("\n⚠️  Fixed %d/%d artist tags", successCount, len(fixes))
	}
}

// updateArtistTag updates just the artist field of an audio file
func updateArtistTag(filePath, newArtist string) error {
	// Read existing tags
	existingTags, err := audiotag.ReadTags(filePath)
	if err != nil {
		// If we can't read existing tags, create new ones
		existingTags = &audiotag.AudioTags{}
	}
	
	// Update only the artist
	existingTags.Artist = newArtist
	
	// Write tags back using our unified API
	err = audiotag.WriteTags(filePath, existingTags)
	if err != nil {
		return fmt.Errorf("failed to write artist tag: %v", err)
	}
	
	return nil
}