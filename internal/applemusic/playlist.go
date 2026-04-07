package applemusic

import (
	"fmt"
	"strings"
	"sync"
)

// GetPlaylistTracks returns all tracks from a playlist with their file locations.
// Tracks with missing locations (orphaned) will have IsOrphaned=true and empty Location.
func GetPlaylistTracks(playlistName string) ([]Track, error) {
	// Escape playlist name for AppleScript
	escapedName := strings.ReplaceAll(playlistName, "\"", "\\\"")

	// First get the track count
	countScript := fmt.Sprintf(`tell application "Music" to return count of tracks of playlist "%s"`, escapedName)
	countOutput, err := runAppleScript(countScript)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist track count: %w", err)
	}

	var total int
	fmt.Sscanf(countOutput, "%d", &total)

	if total == 0 {
		return []Track{}, nil
	}

	// Process in batches
	batchSize := 50
	var tracks []Track

	for offset := 0; offset < total; offset += batchSize {
		end := offset + batchSize
		if end > total {
			end = total
		}

		script := fmt.Sprintf(`
tell application "Music"
	set trackData to {}
	set p to playlist "%s"
	repeat with i from %d to %d
		set t to track i of p
		set trackName to name of t
		set trackArtist to artist of t
		set trackAlbum to album of t
		set trackPID to persistent ID of t
		try
			set trackLoc to POSIX path of (location of t as alias)
		on error
			set trackLoc to "MISSING"
		end try
		set end of trackData to trackPID & "||" & trackName & "||" & trackArtist & "||" & trackAlbum & "||" & trackLoc
	end repeat
	set AppleScript's text item delimiters to "^^^"
	return trackData as text
end tell
`, escapedName, offset+1, end)

		output, err := runAppleScriptFile(script)
		if err != nil {
			return nil, fmt.Errorf("failed to get tracks %d-%d: %w", offset+1, end, err)
		}

		if output != "" {
			entries := strings.Split(output, "^^^")
			for _, entry := range entries {
				parts := strings.Split(entry, "||")
				if len(parts) >= 5 {
					track := Track{
						PersistentID: parts[0],
						Name:         parts[1],
						Artist:       parts[2],
						Album:        parts[3],
					}
					if parts[4] == "MISSING" {
						track.IsOrphaned = true
					} else {
						track.Location = parts[4]
					}
					tracks = append(tracks, track)
				}
			}
		}
	}

	return tracks, nil
}

// GetPlaylistTrackCount returns the number of tracks in a playlist
func GetPlaylistTrackCount(playlistName string) (int, error) {
	escapedName := strings.ReplaceAll(playlistName, "\"", "\\\"")
	script := fmt.Sprintf(`tell application "Music" to return count of tracks of playlist "%s"`, escapedName)
	output, err := runAppleScript(script)
	if err != nil {
		return 0, err
	}
	var count int
	fmt.Sscanf(output, "%d", &count)
	return count, nil
}

// PlaylistExists checks if a playlist with the given name exists
func PlaylistExists(playlistName string) (bool, error) {
	escapedName := strings.ReplaceAll(playlistName, "\"", "\\\"")
	script := fmt.Sprintf(`
tell application "Music"
	try
		set p to playlist "%s"
		return "exists"
	on error
		return "not found"
	end try
end tell
`, escapedName)
	output, err := runAppleScript(script)
	if err != nil {
		return false, err
	}
	return output == "exists", nil
}

// GetUserPlaylists returns all user-created playlists (excluding smart playlists and folders)
func GetUserPlaylists() ([]string, error) {
	script := `
tell application "Music"
	set playlistNames to {}
	repeat with p in user playlists
		if not (smart of p) and not (special kind of p is folder) then
			set end of playlistNames to name of p
		end if
	end repeat
	set AppleScript's text item delimiters to "^^^"
	return playlistNames as text
end tell
`
	output, err := runAppleScriptFile(script)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return []string{}, nil
	}
	return strings.Split(output, "^^^"), nil
}

// GetAllLibraryLocations returns all file locations in the library.
// This is used to find files on disk that aren't in the library.
func GetAllLibraryLocations(progressFn func(checked, total int)) (map[string]bool, error) {
	total, err := GetTrackCount()
	if err != nil {
		return nil, err
	}

	if progressFn != nil {
		progressFn(0, total)
	}

	locations := make(map[string]bool)
	batchSize := 100
	numWorkers := 4 // Limited concurrency to avoid overwhelming Music app

	// Create batch jobs
	type batchJob struct {
		start int
		end   int
	}

	var jobs []batchJob
	for offset := 0; offset < total; offset += batchSize {
		end := offset + batchSize
		if end > total {
			end = total
		}
		jobs = append(jobs, batchJob{start: offset + 1, end: end})
	}

	// Results channel
	type batchResult struct {
		paths []string
		end   int
		err   error
	}

	jobChan := make(chan batchJob, len(jobs))
	resultChan := make(chan batchResult, len(jobs))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobChan {
				script := fmt.Sprintf(`
tell application "Music"
	set locs to {}
	set trackList to file tracks %d thru %d of playlist "Library"
	repeat with t in trackList
		try
			set loc to POSIX path of (location of t as alias)
			set end of locs to loc
		end try
	end repeat
	set AppleScript's text item delimiters to "^^^"
	return locs as text
end tell
`, job.start, job.end)

				output, err := runAppleScriptFile(script)
				if err != nil {
					resultChan <- batchResult{err: fmt.Errorf("failed to get locations %d-%d: %w", job.start, job.end, err), end: job.end}
					continue
				}

				var paths []string
				if output != "" {
					paths = strings.Split(output, "^^^")
				}
				resultChan <- batchResult{paths: paths, end: job.end}
			}
		}()
	}

	// Send jobs
	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	// Wait for workers and close results channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var lastErr error
	processed := 0
	for result := range resultChan {
		if result.err != nil {
			lastErr = result.err
			continue
		}

		for _, p := range result.paths {
			if p != "" {
				locations[p] = true
			}
		}

		processed += batchSize
		if processed > total {
			processed = total
		}
		if progressFn != nil {
			progressFn(processed, total)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return locations, nil
}

// CreatePlaylist creates a new playlist with the given name.
// Returns an error if playlist already exists.
func CreatePlaylist(name string) error {
	escapedName := strings.ReplaceAll(name, "\"", "\\\"")
	script := fmt.Sprintf(`
tell application "Music"
	try
		make new playlist with properties {name:"%s"}
		return "ok"
	on error errMsg
		return "error: " & errMsg
	end try
end tell
`, escapedName)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return err
	}
	if strings.HasPrefix(output, "error:") {
		return fmt.Errorf(output)
	}
	return nil
}

// AddFileToPlaylist adds a file to a playlist by its path.
// The file must already be in the library.
func AddFileToPlaylist(filePath, playlistName string) error {
	escapedPath := strings.ReplaceAll(filePath, "\\", "\\\\")
	escapedPath = strings.ReplaceAll(escapedPath, "\"", "\\\"")
	escapedPlaylist := strings.ReplaceAll(playlistName, "\"", "\\\"")

	script := fmt.Sprintf(`
tell application "Music"
	try
		set targetFile to POSIX file "%s"
		set matchingTracks to (every file track of playlist "Library" whose location is targetFile)
		if (count of matchingTracks) > 0 then
			duplicate item 1 of matchingTracks to playlist "%s"
			return "ok"
		else
			return "error: track not found in library"
		end if
	on error errMsg
		return "error: " & errMsg
	end try
end tell
`, escapedPath, escapedPlaylist)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return err
	}
	if strings.HasPrefix(output, "error:") {
		return fmt.Errorf(output)
	}
	return nil
}

// AddFilesToLibrary adds files to the library and returns the count of successfully added files.
func AddFilesToLibrary(filePaths []string, progressFn func(added, total int)) (int, error) {
	added := 0
	total := len(filePaths)

	for i, path := range filePaths {
		escapedPath := strings.ReplaceAll(path, "\\", "\\\\")
		escapedPath = strings.ReplaceAll(escapedPath, "\"", "\\\"")

		script := fmt.Sprintf(`
tell application "Music"
	try
		add POSIX file "%s"
		return "ok"
	on error errMsg
		return "error: " & errMsg
	end try
end tell
`, escapedPath)

		output, err := runAppleScriptFile(script)
		if err == nil && output == "ok" {
			added++
		}

		if progressFn != nil {
			progressFn(i+1, total)
		}
	}

	return added, nil
}

// AddFilesToPlaylist adds multiple files to a playlist.
// Files must already be in the library.
func AddFilesToPlaylist(filePaths []string, playlistName string, progressFn func(added, total int)) (int, error) {
	added := 0
	total := len(filePaths)

	for i, path := range filePaths {
		err := AddFileToPlaylist(path, playlistName)
		if err == nil {
			added++
		}

		if progressFn != nil {
			progressFn(i+1, total)
		}
	}

	return added, nil
}
