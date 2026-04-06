package music

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gbasile17/foe/dj-utils/internal/audiotag"
)

// AnalyzeFile computes the hash of a file and extracts its metadata.
func AnalyzeFile(filePath string) (File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return File{}, err
	}
	defer file.Close()

	// Compute the MD5 hash of the file content
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return File{}, err
	}
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Extract metadata using audiotag library
	tags, err := audiotag.ReadTags(filePath)
	title := "Unknown Title"
	artist := "Unknown Artist"
	album := "Unknown Album"
	genre := ""

	if err == nil {
		if tags.Title != "" {
			title = tags.Title
		} else if fallback := ExtractTitleFromFilename(filePath); fallback != "" {
			title = fallback
		}
		if tags.Artist != "" {
			artist = tags.Artist
		}
		if tags.Album != "" {
			album = tags.Album
		}
		genre = tags.Genre
	} else {
		if fallback := ExtractTitleFromFilename(filePath); fallback != "" {
			title = fallback
		}
	}

	return File{
		Path:   filePath,
		Title:  title,
		Artist: artist,
		Album:  album,
		Genre:  genre,
		Hash:   hash,
	}, nil
}

// ExtractTitle extracts the title metadata from an audio file.
func ExtractTitle(filePath string) (string, error) {
	tags, err := audiotag.ReadTags(filePath)
	if err != nil || tags.Title == "" {
		if fallback := ExtractTitleFromFilename(filePath); fallback != "" {
			return fallback, nil
		}
		return "Unknown Title", nil
	}
	return tags.Title, nil
}

// ExtractTitleAndArtist extracts title and artist from an audio file.
func ExtractTitleAndArtist(filePath string) (title, artist string, err error) {
	tags, err := audiotag.ReadTags(filePath)
	if err != nil {
		title = "Unknown Title"
		if fallback := ExtractTitleFromFilename(filePath); fallback != "" {
			title = fallback
		}
		return title, "Unknown Artist", nil
	}

	title = tags.Title
	if title == "" {
		if fallback := ExtractTitleFromFilename(filePath); fallback != "" {
			title = fallback
		} else {
			title = "Unknown Title"
		}
	}

	artist = tags.Artist
	if artist == "" {
		artist = "Unknown Artist"
	}

	return title, artist, nil
}

// ExtractTitleFromFilename attempts to extract a meaningful title from the filename.
func ExtractTitleFromFilename(filePath string) string {
	filename := filepath.Base(filePath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	// Remove leading track numbers like "1. Artist - Title" or "01 Artist - Title"
	if strings.Contains(filename, ". ") {
		parts := strings.SplitN(filename, ". ", 2)
		if len(parts) == 2 {
			filename = parts[1]
		}
	}

	if len(strings.TrimSpace(filename)) > 0 {
		return strings.TrimSpace(filename)
	}
	return ""
}

// IsAudioFile checks if a file is an audio file based on its extension.
func IsAudioFile(path string) bool {
	return audiotag.IsAudioFile(path)
}
