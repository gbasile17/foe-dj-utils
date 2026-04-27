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

// CloneAndDeletePlaylist clones a user playlist's tracks into a new playlist
// named "<name> (resync)", verifies the track count matches, then deletes the
// original. The clone keeps the temporary name on return — call RenamePlaylist
// (with retries) afterwards to take over the original name.
//
// This is split from the rename step because Music's name index lags after a
// delete, and the lag grows when iCloud Music Library is processing batch
// deletes. A `delay` inside AppleScript blocks Music's event loop, which is
// the very thing we need to drain — so the retry must happen from Go.
//
// Returns the new playlist's persistent ID.
//
// Refuses to operate on smart playlists, folder playlists, or special-kind
// playlists. Errors out (without deleting anything) if the source can't be
// found unambiguously, if "<name> (resync)" already exists from a prior run,
// or if the cloned track count doesn't match the source.
func CloneAndDeletePlaylist(playlistName string) (string, error) {
	escapedName := strings.ReplaceAll(playlistName, "\"", "\\\"")
	script := fmt.Sprintf(`
on run
	tell application "Music"
		set srcName to "%s"
		set matches to (every user playlist whose name is srcName)
		if (count of matches) is 0 then
			error "no user playlist named " & srcName
		end if
		if (count of matches) > 1 then
			error "multiple user playlists named " & srcName
		end if
		set src to item 1 of matches

		if smart of src is true then
			error "refusing to clone smart playlist"
		end if
		if (special kind of src) is not none then
			error "refusing to clone special-kind playlist"
		end if

		set srcPID to persistent ID of src
		set srcCount to count of tracks of src
		set tmpName to srcName & " (resync)"

		if (count of (every user playlist whose name is tmpName)) > 0 then
			error "playlist named " & tmpName & " already exists"
		end if

		set dst to make new user playlist with properties {name:tmpName}
		with timeout of 600 seconds
			duplicate (every track of src) to dst
		end timeout
		set dstPID to persistent ID of dst
		set dstCount to count of tracks of dst

		if dstCount is not srcCount then
			error "track count mismatch: source=" & srcCount & " clone=" & dstCount
		end if

		delete (some user playlist whose persistent ID is srcPID)

		return dstPID
	end tell
end run
`, escapedName)

	return runAppleScriptFile(script)
}

// TryRenamePlaylist attempts to rename a playlist (by persistent ID) to a new
// name in a single AppleScript call, then verifies the rename took effect.
// Returns true if the new name is set on return, false otherwise.
//
// Music.app's `set name of ... to ...` can return without error and silently
// fail when the target name was very recently held by another playlist that
// was deleted; the only reliable confirmation is to read the name back.
func TryRenamePlaylist(persistentID, newName string) (bool, error) {
	escapedPID := strings.ReplaceAll(persistentID, "\"", "\\\"")
	escapedNewName := strings.ReplaceAll(newName, "\"", "\\\"")
	script := fmt.Sprintf(`
on run
	tell application "Music"
		set p to (some user playlist whose persistent ID is "%s")
		try
			set name of p to "%s"
		end try
		if name of p is "%s" then
			return "ok"
		else
			return "pending"
		end if
	end tell
end run
`, escapedPID, escapedNewName, escapedNewName)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return false, err
	}
	return output == "ok", nil
}

// BlockingTrack represents a track in a source playlist that has a non-syncing
// cloud status.
type BlockingTrack struct {
	SourcePlaylist string
	PersistentID   string
	Name           string
	Artist         string
	CloudStatus    string
}

// blockingStatusList is the set of `cloud status` values that prevent a track
// from syncing to iCloud Music Library AND are unlikely to resolve themselves.
// `not uploaded` is intentionally excluded — Apple's matcher may not have run
// yet, and that state often clears on its own with time.
//
// We compare via string coercion (`(cloud status of t) as text`) inside a
// per-track loop because AppleScript's `whose cloud status is ...` filter
// rejects these enum literals at parse-time (`error` is a reserved keyword
// in `whose` context, and the others fail with -10006).
const blockingStatusListAppleScript = `{"ineligible", "error", "removed", "duplicate", "no longer available"}`

// QuarantineBlockingTracks moves every track with a non-syncing cloud status
// out of `srcName` and into `dstName`, creating `dstName` if needed.
//
// Tracks are de-duplicated within `dstName` by persistent ID — if the same
// track is blocked in multiple source playlists, it appears in `dstName`
// only once. The track stays in the library; only its membership in `srcName`
// is removed.
//
// Returns the list of tracks moved (one entry per source-playlist removal,
// even if a duplicate copy was already in `dstName`).
func QuarantineBlockingTracks(srcName, dstName string) ([]BlockingTrack, error) {
	escapedSrc := strings.ReplaceAll(srcName, "\"", "\\\"")
	escapedDst := strings.ReplaceAll(dstName, "\"", "\\\"")

	script := fmt.Sprintf(`
on run
	tell application "Music"
		set srcName to "%s"
		set dstName to "%s"
		set blockingList to %s

		set srcMatches to (every user playlist whose name is srcName)
		if (count of srcMatches) is 0 then
			error "no user playlist named " & srcName
		end if
		set src to item 1 of srcMatches

		-- Ensure destination exists.
		set dstMatches to (every user playlist whose name is dstName)
		if (count of dstMatches) is 0 then
			set dst to make new user playlist with properties {name:dstName}
		else
			set dst to item 1 of dstMatches
		end if

		-- Snapshot existing destination PIDs so we can de-dupe.
		set existingPIDs to {}
		repeat with t in (every track of dst)
			set end of existingPIDs to (persistent ID of t)
		end repeat

		-- Find blocking tracks. AppleScript's whose-filter on cloud status
		-- enum is unreliable (parser conflicts on the error keyword,
		-- -10006 on others), so we iterate and string-coerce per track.
		set blockers to {}
		repeat with t in (every track of src)
			set cs to (cloud status of t) as text
			if blockingList contains cs then
				set end of blockers to t
			end if
		end repeat

		set output to ""
		repeat with t in blockers
			set tPID to persistent ID of t
			set tName to name of t
			set tArtist to artist of t
			set tStatus to (cloud status of t) as text

			-- Add to destination only if not already there.
			if existingPIDs does not contain tPID then
				duplicate t to dst
				set end of existingPIDs to tPID
			end if

			-- Remove from source playlist (NOT from library — that would be
			-- delete on playlist "Library").
			delete t

			set output to output & tPID & "||" & tName & "||" & tArtist & "||" & tStatus & "^^^"
		end repeat

		return output
	end tell
end run
`, escapedSrc, escapedDst, blockingStatusListAppleScript)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return nil, err
	}

	var moved []BlockingTrack
	if output == "" {
		return moved, nil
	}
	for _, entry := range strings.Split(strings.TrimSuffix(output, "^^^"), "^^^") {
		parts := strings.Split(entry, "||")
		if len(parts) >= 4 {
			moved = append(moved, BlockingTrack{
				SourcePlaylist: srcName,
				PersistentID:   parts[0],
				Name:           parts[1],
				Artist:         parts[2],
				CloudStatus:    parts[3],
			})
		}
	}
	return moved, nil
}

// ListBlockingTracks returns every track in `srcName` whose cloud status
// would block iCloud sync. Read-only — does not mutate the playlist.
func ListBlockingTracks(srcName string) ([]BlockingTrack, error) {
	escapedSrc := strings.ReplaceAll(srcName, "\"", "\\\"")
	script := fmt.Sprintf(`
on run
	tell application "Music"
		set src to (some user playlist whose name is "%s")
		set blockingList to %s
		set output to ""
		repeat with t in (every track of src)
			set cs to (cloud status of t) as text
			if blockingList contains cs then
				set output to output & (persistent ID of t) & "||" & (name of t) & "||" & (artist of t) & "||" & cs & "^^^"
			end if
		end repeat
		return output
	end tell
end run
`, escapedSrc, blockingStatusListAppleScript)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}
	var tracks []BlockingTrack
	for _, entry := range strings.Split(strings.TrimSuffix(output, "^^^"), "^^^") {
		parts := strings.Split(entry, "||")
		if len(parts) >= 4 {
			tracks = append(tracks, BlockingTrack{
				SourcePlaylist: srcName,
				PersistentID:   parts[0],
				Name:           parts[1],
				Artist:         parts[2],
				CloudStatus:    parts[3],
			})
		}
	}
	return tracks, nil
}

// CountBlockingTracks returns the number of tracks in `srcName` whose
// cloud status would block iCloud sync. Read-only — for dry-run reporting.
func CountBlockingTracks(srcName string) (int, error) {
	escapedSrc := strings.ReplaceAll(srcName, "\"", "\\\"")
	script := fmt.Sprintf(`
tell application "Music"
	set src to (some user playlist whose name is "%s")
	set blockingList to %s
	set n to 0
	repeat with t in (every track of src)
		set cs to (cloud status of t) as text
		if blockingList contains cs then set n to n + 1
	end repeat
	return n
end tell
`, escapedSrc, blockingStatusListAppleScript)
	output, err := runAppleScriptFile(script)
	if err != nil {
		return 0, err
	}
	var n int
	fmt.Sscanf(output, "%d", &n)
	return n, nil
}

// LibraryTrack represents a candidate canonical track in the library.
type LibraryTrack struct {
	PersistentID string
	Name         string
	Artist       string
	Album        string
	Duration     float64 // seconds
	Size         int64   // bytes
	Kind         string  // e.g. "Matched AAC audio file", "Apple Music AAC audio file"
	CloudStatus  string
	Location     string // POSIX path; empty if cloud-only / missing
}

// FindLibraryTracksByTitleArtist searches the entire library for tracks
// whose name (case-insensitive substring) and artist match the given values.
// Used to locate the canonical copy of a track flagged as `cloud status =
// duplicate`. Caller is responsible for filtering down to the right one
// (e.g. by duration tolerance) — this returns all candidates so the caller
// can decide.
func FindLibraryTracksByTitleArtist(titleSubstring, artist string) ([]LibraryTrack, error) {
	escapedTitle := strings.ReplaceAll(titleSubstring, "\"", "\\\"")
	escapedArtist := strings.ReplaceAll(artist, "\"", "\\\"")

	script := fmt.Sprintf(`
on run
	tell application "Music"
		set wantTitle to "%s"
		set wantArtist to "%s"
		set lib to playlist "Library"
		set output to ""

		-- AppleScript "contains" on text is case-insensitive by default. We
		-- match on artist exactly and title as a substring (the source title
		-- has been stripped of DJ prefixes by the caller, but Apple's catalog
		-- title may differ slightly — substring is more forgiving).
		set hits to (every track of lib whose artist is wantArtist and name contains wantTitle)
		repeat with t in hits
			set tLoc to ""
			try
				set tLoc to POSIX path of (location of t as alias)
			end try
			set tDur to 0
			try
				set tDur to duration of t
			end try
			set tSize to 0
			try
				set tSize to size of t
			end try
			set output to output & ¬
				(persistent ID of t) & "||" & ¬
				(name of t) & "||" & ¬
				(artist of t) & "||" & ¬
				(album of t) & "||" & ¬
				tDur & "||" & ¬
				tSize & "||" & ¬
				(kind of t) & "||" & ¬
				((cloud status of t) as text) & "||" & ¬
				tLoc & "^^^"
		end repeat
		return output
	end tell
end run
`, escapedTitle, escapedArtist)

	output, err := runAppleScriptFile(script)
	if err != nil {
		return nil, err
	}
	if output == "" {
		return nil, nil
	}

	var tracks []LibraryTrack
	for _, entry := range strings.Split(strings.TrimSuffix(output, "^^^"), "^^^") {
		parts := strings.Split(entry, "||")
		if len(parts) >= 9 {
			lt := LibraryTrack{
				PersistentID: parts[0],
				Name:         parts[1],
				Artist:       parts[2],
				Album:        parts[3],
				Kind:         parts[6],
				CloudStatus:  parts[7],
				Location:     parts[8],
			}
			fmt.Sscanf(parts[4], "%f", &lt.Duration)
			fmt.Sscanf(parts[5], "%d", &lt.Size)
			tracks = append(tracks, lt)
		}
	}
	return tracks, nil
}

// GetTrackDetails returns full details for a single track identified by
// persistent ID. Used to fetch the duration of a `duplicate` track so we
// can match candidates by duration tolerance.
func GetTrackDetails(persistentID string) (*LibraryTrack, error) {
	escapedPID := strings.ReplaceAll(persistentID, "\"", "\\\"")
	script := fmt.Sprintf(`
on run
	tell application "Music"
		set t to (some track of playlist "Library" whose persistent ID is "%s")
		set tLoc to ""
		try
			set tLoc to POSIX path of (location of t as alias)
		end try
		return (persistent ID of t) & "||" & ¬
			(name of t) & "||" & ¬
			(artist of t) & "||" & ¬
			(album of t) & "||" & ¬
			(duration of t) & "||" & ¬
			(size of t) & "||" & ¬
			(kind of t) & "||" & ¬
			((cloud status of t) as text) & "||" & ¬
			tLoc
	end tell
end run
`, escapedPID)
	output, err := runAppleScriptFile(script)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(output, "||")
	if len(parts) < 9 {
		return nil, fmt.Errorf("unexpected output: %q", output)
	}
	lt := &LibraryTrack{
		PersistentID: parts[0],
		Name:         parts[1],
		Artist:       parts[2],
		Album:        parts[3],
		Kind:         parts[6],
		CloudStatus:  parts[7],
		Location:     parts[8],
	}
	fmt.Sscanf(parts[4], "%f", &lt.Duration)
	fmt.Sscanf(parts[5], "%d", &lt.Size)
	return lt, nil
}

// DeleteTrackFromLibrary removes a track from the Music library entirely
// (not just from one playlist). Whether the underlying file is moved to
// Trash depends on Music.app's "Keep Music Media folder organized" /
// "delete from disk" preference; the caller should treat the file path as
// possibly still present and handle the on-disk file separately.
func DeleteTrackFromLibrary(persistentID string) error {
	escapedPID := strings.ReplaceAll(persistentID, "\"", "\\\"")
	script := fmt.Sprintf(`
tell application "Music"
	set matches to (every track of playlist "Library" whose persistent ID is "%s")
	if (count of matches) is 0 then
		error "no track with persistent ID " & "%s"
	end if
	delete (item 1 of matches)
	return "ok"
end tell
`, escapedPID, escapedPID)
	output, err := runAppleScriptFile(script)
	if err != nil {
		return err
	}
	if output != "ok" {
		return fmt.Errorf("unexpected output: %q", output)
	}
	return nil
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
