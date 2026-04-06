package applemusic

import (
	"fmt"
	"strings"
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
