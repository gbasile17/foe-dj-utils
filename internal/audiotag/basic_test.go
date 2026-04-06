package audiotag

import (
	"testing"
)

// TestBasicFunctionality tests core functionality without external dependencies
func TestBasicFunctionality(t *testing.T) {
	// Test format detection
	tests := []struct {
		filename string
		expected AudioFormat
	}{
		{"test.mp3", FormatMP3},
		{"test.flac", FormatFLAC},
		{"test.wav", FormatWAV},
		{"test.aiff", FormatAIFF},
		{"test.unknown", FormatUnknown},
	}

	for _, test := range tests {
		result := DetectFormat(test.filename)
		if result != test.expected {
			t.Errorf("DetectFormat(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}

// TestIsAudioFileBasic tests basic audio file detection
func TestIsAudioFileBasic(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.mp3", true},
		{"test.flac", true},
		{"test.wav", true},
		{"test.aiff", true},
		{"test.txt", false},
		{"test.unknown", false},
	}

	for _, test := range tests {
		result := IsAudioFile(test.filename)
		if result != test.expected {
			t.Errorf("IsAudioFile(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}

// TestAudioTagsStructure tests the AudioTags struct
func TestAudioTagsStructure(t *testing.T) {
	tags := &AudioTags{
		Title:       "Test Title",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Genre:       "Rock",
		Year:        2023,
		Track:       5,
		TrackTotal:  12,
		Disc:        1,
		DiscTotal:   2,
		Composer:    "Test Composer",
		Comment:     "Test Comment",
	}

	if tags.Title != "Test Title" {
		t.Error("Title field not working correctly")
	}
	if tags.Genre != "Rock" {
		t.Error("Genre field not working correctly")
	}
	if tags.Year != 2023 {
		t.Error("Year field not working correctly")
	}
	if tags.Track != 5 {
		t.Error("Track field not working correctly")
	}
}

// TestFormatString tests the String method of AudioFormat
func TestFormatString(t *testing.T) {
	tests := []struct {
		format   AudioFormat
		expected string
	}{
		{FormatMP3, "MP3"},
		{FormatFLAC, "FLAC"},
		{FormatWAV, "WAV"},
		{FormatAIFF, "AIFF"},
		{FormatUnknown, "Unknown"},
	}

	for _, test := range tests {
		result := test.format.String()
		if result != test.expected {
			t.Errorf("AudioFormat(%d).String() = %s, expected %s", test.format, result, test.expected)
		}
	}
}

// TestReaderWriterCreation tests creating readers and writers
func TestReaderWriterCreation(t *testing.T) {
	formats := []AudioFormat{FormatMP3, FormatFLAC, FormatWAV, FormatAIFF}
	
	for _, format := range formats {
		reader, err := NewTagReaderWriter(format)
		if err != nil {
			t.Errorf("Failed to create reader for %s: %v", format, err)
			continue
		}
		if reader == nil {
			t.Errorf("Reader is nil for format %s", format)
		}
	}

	// Test unsupported format
	_, err := NewTagReaderWriter(FormatUnknown)
	if err == nil {
		t.Error("Should return error for unsupported format")
	}
}