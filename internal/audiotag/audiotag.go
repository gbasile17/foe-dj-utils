// Package audiotag provides a unified interface for reading and writing
// audio metadata tags across multiple formats (MP3, FLAC, WAV, AIFF)
package audiotag

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AudioTags represents common metadata fields across audio formats
type AudioTags struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Year        int
	Track       int
	TrackTotal  int
	Disc        int
	DiscTotal   int
	Composer    string
	Comment     string
}

// TagReader interface for reading tags from audio files
type TagReader interface {
	ReadTags(filePath string) (*AudioTags, error)
}

// TagWriter interface for writing tags to audio files
type TagWriter interface {
	WriteTags(filePath string, tags *AudioTags) error
}

// TagReaderWriter combines reading and writing capabilities
type TagReaderWriter interface {
	TagReader
	TagWriter
}

// AudioFormat represents the supported audio formats
type AudioFormat int

const (
	FormatUnknown AudioFormat = iota
	FormatMP3
	FormatFLAC
	FormatWAV
	FormatAIFF
)

// String returns the string representation of AudioFormat
func (f AudioFormat) String() string {
	switch f {
	case FormatMP3:
		return "MP3"
	case FormatFLAC:
		return "FLAC"
	case FormatWAV:
		return "WAV"
	case FormatAIFF:
		return "AIFF"
	default:
		return "Unknown"
	}
}

// DetectFormat determines the audio format from file extension
func DetectFormat(filePath string) AudioFormat {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp3":
		return FormatMP3
	case ".flac":
		return FormatFLAC
	case ".wav":
		return FormatWAV
	case ".aiff", ".aif":
		return FormatAIFF
	default:
		return FormatUnknown
	}
}

// NewTagReaderWriter creates a new TagReaderWriter for the given audio format
func NewTagReaderWriter(format AudioFormat) (TagReaderWriter, error) {
	switch format {
	case FormatMP3:
		return NewMP3TagReaderWriter(), nil
	case FormatFLAC:
		return NewFLACTagReaderWriter(), nil
	case FormatWAV:
		return NewWAVTagReaderWriter(), nil
	case FormatAIFF:
		return NewAIFFTagReaderWriter(), nil
	default:
		return nil, fmt.Errorf("unsupported audio format: %s", format)
	}
}

// ReadTags reads tags from an audio file, automatically detecting the format
func ReadTags(filePath string) (*AudioTags, error) {
	format := DetectFormat(filePath)
	if format == FormatUnknown {
		return nil, fmt.Errorf("unsupported file format: %s", filePath)
	}

	reader, err := NewTagReaderWriter(format)
	if err != nil {
		return nil, err
	}

	return reader.ReadTags(filePath)
}

// WriteTags writes tags to an audio file, automatically detecting the format
func WriteTags(filePath string, tags *AudioTags) error {
	format := DetectFormat(filePath)
	if format == FormatUnknown {
		return fmt.Errorf("unsupported file format: %s", filePath)
	}

	writer, err := NewTagReaderWriter(format)
	if err != nil {
		return err
	}

	return writer.WriteTags(filePath, tags)
}

// HasGenre checks if the file has a genre tag
func HasGenre(filePath string) (bool, error) {
	tags, err := ReadTags(filePath)
	if err != nil {
		return false, err
	}
	return tags.Genre != "", nil
}

// IsAudioFile checks if a file is a supported audio format
func IsAudioFile(filePath string) bool {
	return DetectFormat(filePath) != FormatUnknown
}