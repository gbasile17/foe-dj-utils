package music

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SearchTitles searches for audio file titles containing the given query.
func SearchTitles(dirs []string, query string) ([]TitleResult, error) {
	var results []TitleResult
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !IsAudioFile(path) {
				continue
			}

			title, err := ExtractTitle(path)
			if err != nil {
				continue
			}

			if strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
				mu.Lock()
				results = append(results, TitleResult{Path: path, Title: title})
				mu.Unlock()
			}
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

	return results, nil
}

// SearchArtists searches for audio files with artist tags containing the given query.
func SearchArtists(dirs []string, query string) ([]ArtistResult, error) {
	var results []ArtistResult
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !IsAudioFile(path) {
				continue
			}

			title, artist, err := ExtractTitleAndArtist(path)
			if err != nil {
				continue
			}

			if strings.Contains(strings.ToLower(artist), strings.ToLower(query)) {
				mu.Lock()
				results = append(results, ArtistResult{Path: path, Title: title, Artist: artist})
				mu.Unlock()
			}
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

	return results, nil
}
