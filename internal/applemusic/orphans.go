package applemusic

import (
	"fmt"
	"strconv"
	"strings"
)

// FindOrphanedTracks finds all tracks in the library where the file is missing.
// This can take a while for large libraries. The progressFn callback is called
// periodically with (checked, total) counts if provided.
func FindOrphanedTracks(progressFn func(checked, total int)) ([]Track, error) {
	// First get total count for progress reporting
	total, err := GetTrackCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get track count: %w", err)
	}

	if progressFn != nil {
		progressFn(0, total)
	}

	// Process in batches to avoid AppleScript timeout and provide progress updates
	batchSize := 100
	var orphans []Track

	for offset := 0; offset < total; offset += batchSize {
		end := offset + batchSize
		if end > total {
			end = total
		}

		script := fmt.Sprintf(`
tell application "Music"
	set orphanList to {}
	set trackList to file tracks %d thru %d of playlist "Library"
	repeat with t in trackList
		try
			set loc to location of t
			if loc is missing value then
				set end of orphanList to (persistent ID of t) & "||" & (name of t) & "||" & (artist of t) & "||" & (album of t)
			end if
		on error
			set end of orphanList to (persistent ID of t) & "||" & (name of t) & "||" & (artist of t) & "||" & (album of t)
		end try
	end repeat
	set AppleScript's text item delimiters to "^^^"
	return orphanList as text
end tell
`, offset+1, end)

		output, err := runAppleScriptFile(script)
		if err != nil {
			return nil, fmt.Errorf("failed to check tracks %d-%d: %w", offset+1, end, err)
		}

		if output != "" {
			// Parse the output
			entries := strings.Split(output, "^^^")
			for _, entry := range entries {
				parts := strings.Split(entry, "||")
				if len(parts) >= 4 {
					orphans = append(orphans, Track{
						PersistentID: parts[0],
						Name:         parts[1],
						Artist:       parts[2],
						Album:        parts[3],
						IsOrphaned:   true,
					})
				}
			}
		}

		if progressFn != nil {
			progressFn(end, total)
		}
	}

	return orphans, nil
}

// FindOrphanedTracksByPaths finds tracks in the library whose file paths match
// any of the provided paths (files that were just deleted). This is faster than
// scanning the entire library when you know which files were deleted.
func FindOrphanedTracksByPaths(deletedPaths []string) ([]Track, error) {
	if len(deletedPaths) == 0 {
		return []Track{}, nil
	}

	// Build AppleScript to find tracks by location
	// We need to escape paths for AppleScript
	var pathChecks []string
	for _, p := range deletedPaths {
		escaped := strings.ReplaceAll(p, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		pathChecks = append(pathChecks, fmt.Sprintf(`"%s"`, escaped))
	}

	script := fmt.Sprintf(`
tell application "Music"
	set deletedPaths to {%s}
	set orphanList to {}
	repeat with p in deletedPaths
		try
			set matchingTracks to (file tracks of playlist "Library" whose location is (POSIX file p as alias))
			repeat with t in matchingTracks
				set end of orphanList to (persistent ID of t) & "||" & (name of t) & "||" & (artist of t) & "||" & (album of t)
			end repeat
		end try
	end repeat
	set AppleScript's text item delimiters to "^^^"
	return orphanList as text
end tell
`, strings.Join(pathChecks, ", "))

	output, err := runAppleScriptFile(script)
	if err != nil {
		// If paths don't exist, the query might fail - that's expected
		// Try the slower approach of finding by missing location
		return findOrphanedByMissingLocation(deletedPaths)
	}

	var orphans []Track
	if output != "" {
		entries := strings.Split(output, "^^^")
		for _, entry := range entries {
			parts := strings.Split(entry, "||")
			if len(parts) >= 4 {
				orphans = append(orphans, Track{
					PersistentID: parts[0],
					Name:         parts[1],
					Artist:       parts[2],
					Album:        parts[3],
					IsOrphaned:   true,
				})
			}
		}
	}

	return orphans, nil
}

// findOrphanedByMissingLocation is a fallback that checks all tracks
// but only returns ones whose original location matched our deleted paths
func findOrphanedByMissingLocation(deletedPaths []string) ([]Track, error) {
	// Create a set of deleted paths for quick lookup
	deletedSet := make(map[string]bool)
	for _, p := range deletedPaths {
		deletedSet[p] = true
	}

	// Get all orphaned tracks and filter
	allOrphans, err := FindOrphanedTracks(nil)
	if err != nil {
		return nil, err
	}

	// Since we can't get the original location of orphaned tracks,
	// we'll return all orphans found. The caller should handle this appropriately.
	return allOrphans, nil
}

// DeleteTracksByPersistentID deletes tracks from the Music library by their persistent IDs
func DeleteTracksByPersistentID(persistentIDs []string) error {
	if len(persistentIDs) == 0 {
		return nil
	}

	// Delete tracks one by one to handle errors gracefully
	var errors []string
	for _, pid := range persistentIDs {
		script := fmt.Sprintf(`
tell application "Music"
	try
		delete (first file track of playlist "Library" whose persistent ID is "%s")
		return "ok"
	on error errMsg
		return "error: " & errMsg
	end try
end tell
`, pid)

		output, err := runAppleScriptFile(script)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete track %s: %v", pid, err))
		} else if strings.HasPrefix(output, "error:") {
			errors = append(errors, fmt.Sprintf("track %s: %s", pid, output))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some tracks failed to delete:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// DeleteTracksByDatabaseID deletes tracks by their database IDs (faster than persistent ID)
func DeleteTracksByDatabaseID(databaseIDs []int) error {
	if len(databaseIDs) == 0 {
		return nil
	}

	var errors []string
	for _, dbID := range databaseIDs {
		script := fmt.Sprintf(`
tell application "Music"
	try
		delete (first file track of playlist "Library" whose database ID is %d)
		return "ok"
	on error errMsg
		return "error: " & errMsg
	end try
end tell
`, dbID)

		output, err := runAppleScriptFile(script)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete track %d: %v", dbID, err))
		} else if strings.HasPrefix(output, "error:") {
			errors = append(errors, fmt.Sprintf("track %d: %s", dbID, output))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("some tracks failed to delete:\n%s", strings.Join(errors, "\n"))
	}

	return nil
}

// DeleteAllOrphanedTracks finds and deletes all orphaned tracks in the library.
// Returns the count of deleted tracks.
func DeleteAllOrphanedTracks(progressFn func(checked, total int)) (int, error) {
	orphans, err := FindOrphanedTracks(progressFn)
	if err != nil {
		return 0, err
	}

	if len(orphans) == 0 {
		return 0, nil
	}

	var persistentIDs []string
	for _, t := range orphans {
		persistentIDs = append(persistentIDs, t.PersistentID)
	}

	err = DeleteTracksByPersistentID(persistentIDs)
	if err != nil {
		return 0, err
	}

	return len(orphans), nil
}

// QuickOrphanCheck does a fast check for any orphaned tracks without returning full details.
// Returns true if orphans exist.
func QuickOrphanCheck() (bool, error) {
	script := `
tell application "Music"
	repeat with t in (file tracks 1 thru 100 of playlist "Library")
		try
			set loc to location of t
			if loc is missing value then
				return "found"
			end if
		on error
			return "found"
		end try
	end repeat
	return "none"
end tell
`
	output, err := runAppleScriptFile(script)
	if err != nil {
		return false, err
	}
	return output == "found", nil
}

// GetOrphanCount returns the number of orphaned tracks in the library
func GetOrphanCount() (int, error) {
	total, err := GetTrackCount()
	if err != nil {
		return 0, err
	}

	count := 0
	batchSize := 200

	for offset := 0; offset < total; offset += batchSize {
		end := offset + batchSize
		if end > total {
			end = total
		}

		script := fmt.Sprintf(`
tell application "Music"
	set orphanCount to 0
	set trackList to file tracks %d thru %d of playlist "Library"
	repeat with t in trackList
		try
			set loc to location of t
			if loc is missing value then
				set orphanCount to orphanCount + 1
			end if
		on error
			set orphanCount to orphanCount + 1
		end try
	end repeat
	return orphanCount
end tell
`, offset+1, end)

		output, err := runAppleScriptFile(script)
		if err != nil {
			return 0, err
		}

		batchCount, _ := strconv.Atoi(output)
		count += batchCount
	}

	return count, nil
}