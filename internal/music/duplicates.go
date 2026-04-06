package music

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DuplicateGroup represents a group of duplicate files.
type DuplicateGroup struct {
	Key   string // e.g., "Hash: abc123" or "Title: song name"
	Files []File
}

// FindDuplicates analyzes multiple directories and identifies duplicates.
// Returns a map where keys describe the duplicate group and values are the duplicate files.
func FindDuplicates(dirs []string) (map[string][]File, error) {
	hashToFiles := make(map[string][]File)
	titleToFiles := make(map[string][]File)
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !IsAudioFile(path) {
				continue
			}

			musicFile, err := AnalyzeFile(path)
			if err != nil {
				fmt.Printf("Failed to analyze file %s: %v\n", path, err)
				continue
			}

			mu.Lock()
			hashToFiles[musicFile.Hash] = append(hashToFiles[musicFile.Hash], musicFile)
			lowerTitle := strings.ToLower(musicFile.Title)
			titleToFiles[lowerTitle] = append(titleToFiles[lowerTitle], musicFile)
			mu.Unlock()
		}
	}

	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processFile()
	}

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
			close(fileChan)
			return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
		}
	}

	close(fileChan)
	wg.Wait()

	return combineDuplicateGroups(hashToFiles, titleToFiles), nil
}

// combineDuplicateGroups merges hash-based and title-based duplicates.
func combineDuplicateGroups(hashToFiles, titleToFiles map[string][]File) map[string][]File {
	combined := make(map[string][]File)
	processedFiles := make(map[string]bool)

	for hash, files := range hashToFiles {
		if len(files) > 1 {
			combined[fmt.Sprintf("Hash: %s", hash)] = files
			for _, file := range files {
				processedFiles[file.Path] = true
			}
		}
	}

	for title, files := range titleToFiles {
		if len(files) > 1 {
			var unprocessedFiles []File
			for _, file := range files {
				if !processedFiles[file.Path] {
					unprocessedFiles = append(unprocessedFiles, file)
				}
			}
			if len(unprocessedFiles) > 1 {
				combined[fmt.Sprintf("Title: %s", title)] = unprocessedFiles
			}
		}
	}

	return combined
}
