package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gbasile17/foe/dj-utils/pkg/audiotag"
)

// MusicFile represents a music file's details
type MusicFile struct {
	Path  string
	Title string
	Hash  string
}

// TitleResult represents an audio file's title search result
type TitleResult struct {
	Path  string
	Title string
}

// ArtistResult represents an audio file's artist search result
type ArtistResult struct {
	Path   string
	Title  string
	Artist string
}

// isAudioFile checks if a file is an audio file based on its extension
func isAudioFile(path string) bool {
	// Use our audiotag package's function
	return audiotag.IsAudioFile(path)
}

// analyzeFile computes the hash of a file and extracts its metadata
func analyzeFile(filePath string) (MusicFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return MusicFile{}, err
	}
	defer file.Close()

	// Compute the MD5 hash of the file content
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return MusicFile{}, err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Extract metadata using our audiotag library
	tags, err := audiotag.ReadTags(filePath)
	title := "Unknown Title"
	if err == nil && tags.Title != "" {
		title = tags.Title
	} else {
		// Fallback: extract title from filename
		if fallbackTitle := extractTitleFromFilename(filePath); fallbackTitle != "" {
			title = fallbackTitle
		}
	}

	return MusicFile{Path: filePath, Title: title, Hash: hash}, nil
}

// findDuplicatesAcrossDirectories analyzes multiple directories and identifies duplicates
func findDuplicatesAcrossDirectories(dirs []string) (map[string][]MusicFile, error) {
	hashToFiles := make(map[string][]MusicFile)  // Group by hash
	titleToFiles := make(map[string][]MusicFile) // Group by title (case-insensitive)
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !isAudioFile(path) {
				continue
			}

			// Analyze the file to get its hash and title
			musicFile, err := analyzeFile(path)
			if err != nil {
				fmt.Printf("Failed to analyze file %s: %v\n", path, err)
				continue
			}

			// Group by hash
			mu.Lock()
			hashToFiles[musicFile.Hash] = append(hashToFiles[musicFile.Hash], musicFile)

			// Group by title (case-insensitive)
			lowerTitle := strings.ToLower(musicFile.Title)
			titleToFiles[lowerTitle] = append(titleToFiles[lowerTitle], musicFile)
			mu.Unlock()
		}
	}

	// Start worker goroutines
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processFile()
	}

	// Walk through directories and send file paths to the channel
	for _, dir := range dirs {
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
			return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
		}
	}

	// Close the channel and wait for workers
	close(fileChan)
	wg.Wait()

	// Combine results from both hash-based and title-based grouping
	combinedResults := combineDuplicateGroups(hashToFiles, titleToFiles)
	return combinedResults, nil
}

// combineDuplicateGroups merges hash-based and title-based duplicates
func combineDuplicateGroups(hashToFiles map[string][]MusicFile, titleToFiles map[string][]MusicFile) map[string][]MusicFile {
	combined := make(map[string][]MusicFile)
	processedFiles := make(map[string]bool) // Track files already processed by hash

	// Add hash-based duplicates (these take priority)
	for hash, files := range hashToFiles {
		if len(files) > 1 {
			combined[fmt.Sprintf("Hash: %s", hash)] = files
			// Mark these files as processed
			for _, file := range files {
				processedFiles[file.Path] = true
			}
		}
	}

	// Add title-based duplicates only if files weren't already processed by hash
	for title, files := range titleToFiles {
		if len(files) > 1 {
			// Filter out files already processed by hash matching
			unprocessedFiles := []MusicFile{}
			for _, file := range files {
				if !processedFiles[file.Path] {
					unprocessedFiles = append(unprocessedFiles, file)
				}
			}
			// Only add if we still have duplicates after filtering
			if len(unprocessedFiles) > 1 {
				combined[fmt.Sprintf("Title: %s", title)] = unprocessedFiles
			}
		}
	}

	return combined
}

// searchTitles searches for audio file titles containing the given query
func searchTitles(dirs []string, query string) ([]TitleResult, error) {
	results := []TitleResult{}
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	// Worker function to process files
	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !isAudioFile(path) {
				continue
			}

			title, err := extractTitle(path)
			if err != nil {
				fmt.Printf("Failed to extract title for file %s: %v\n", path, err)
				continue
			}

			if strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
				mu.Lock()
				results = append(results, TitleResult{Path: path, Title: title})
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

	// Walk through all directories and send file paths to the channel
	for _, dir := range dirs {
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
			return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
		}
	}

	// Close the channel after all directories are processed
	close(fileChan)
	wg.Wait()

	return results, nil
}

// extractTitleFromFilename attempts to extract a meaningful title from the filename
func extractTitleFromFilename(filePath string) string {
	// Get the filename without path and extension
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Common patterns to clean up:
	// "1. Artist - Title" -> "Artist - Title"
	// "01 Artist - Title" -> "Artist - Title"
	// Remove leading track numbers
	if strings.Contains(filename, ". ") {
		parts := strings.SplitN(filename, ". ", 2)
		if len(parts) == 2 {
			filename = parts[1]
		}
	}
	
	// If the filename is still meaningful (not just numbers/symbols), return it
	if len(strings.TrimSpace(filename)) > 0 {
		return strings.TrimSpace(filename)
	}
	
	return ""
}

// extractTitle extracts the title metadata from an audio file
func extractTitle(filePath string) (string, error) {
	tags, err := audiotag.ReadTags(filePath)
	if err != nil || tags.Title == "" {
		// Fallback: extract title from filename
		if fallbackTitle := extractTitleFromFilename(filePath); fallbackTitle != "" {
			return fallbackTitle, nil
		}
		return "Unknown Title", nil
	}

	return tags.Title, nil
}

// extractTitleAndArtist extracts the title and artist metadata from an audio file
func extractTitleAndArtist(filePath string) (string, string, error) {
	tags, err := audiotag.ReadTags(filePath)
	if err != nil {
		// Fallback: extract title from filename
		title := "Unknown Title"
		if fallbackTitle := extractTitleFromFilename(filePath); fallbackTitle != "" {
			title = fallbackTitle
		}
		return title, "Unknown Artist", nil
	}

	title := tags.Title
	if title == "" {
		// Fallback: extract title from filename
		if fallbackTitle := extractTitleFromFilename(filePath); fallbackTitle != "" {
			title = fallbackTitle
		} else {
			title = "Unknown Title"
		}
	}

	artist := tags.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}

	return title, artist, nil
}

// searchArtists searches for audio files with artist tags containing the given query
func searchArtists(dirs []string, query string) ([]ArtistResult, error) {
	results := []ArtistResult{}
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	// Worker function to process files
	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !isAudioFile(path) {
				continue
			}

			title, artist, err := extractTitleAndArtist(path)
			if err != nil {
				fmt.Printf("Failed to extract metadata for file %s: %v\n", path, err)
				continue
			}

			// Check if the artist contains the query string
			if strings.Contains(strings.ToLower(artist), strings.ToLower(query)) {
				mu.Lock()
				results = append(results, ArtistResult{Path: path, Title: title, Artist: artist})
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

	// Walk through all directories and send file paths to the channel
	for _, dir := range dirs {
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
			return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
		}
	}

	// Close the channel after all directories are processed
	close(fileChan)
	wg.Wait()

	return results, nil
}
