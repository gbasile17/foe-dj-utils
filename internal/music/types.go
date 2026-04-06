// Package music provides types and operations for music file management.
package music

// File represents a music file with its metadata and analysis results.
type File struct {
	Path   string
	Title  string
	Artist string
	Album  string
	Genre  string
	Hash   string // MD5 hash for duplicate detection
}

// TitleResult represents a title search result.
type TitleResult struct {
	Path  string
	Title string
}

// ArtistResult represents an artist search result.
type ArtistResult struct {
	Path   string
	Title  string
	Artist string
}
