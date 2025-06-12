package audiotag

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDetectFormat tests audio format detection
func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filename string
		expected AudioFormat
	}{
		{"test.mp3", FormatMP3},
		{"test.MP3", FormatMP3},
		{"test.flac", FormatFLAC},
		{"test.FLAC", FormatFLAC},
		{"test.wav", FormatWAV},
		{"test.WAV", FormatWAV},
		{"test.aiff", FormatAIFF},
		{"test.aif", FormatAIFF},
		{"test.AIFF", FormatAIFF},
		{"test.txt", FormatUnknown},
		{"test.m4a", FormatUnknown},
		{"test", FormatUnknown},
	}

	for _, test := range tests {
		result := DetectFormat(test.filename)
		if result != test.expected {
			t.Errorf("DetectFormat(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}

// TestAudioFormatString tests the String method of AudioFormat
func TestAudioFormatString(t *testing.T) {
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

// TestIsAudioFile tests audio file detection
func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"test.mp3", true},
		{"test.flac", true},
		{"test.wav", true},
		{"test.aiff", true},
		{"test.aif", true},
		{"test.txt", false},
		{"test.m4a", false},
		{"test", false},
	}

	for _, test := range tests {
		result := IsAudioFile(test.filename)
		if result != test.expected {
			t.Errorf("IsAudioFile(%s) = %v, expected %v", test.filename, result, test.expected)
		}
	}
}

// TestNewTagReaderWriter tests creating tag reader/writers for different formats
func TestNewTagReaderWriter(t *testing.T) {
	tests := []struct {
		format      AudioFormat
		shouldError bool
	}{
		{FormatMP3, false},
		{FormatFLAC, false},
		{FormatWAV, false},
		{FormatAIFF, false},
		{FormatUnknown, true},
	}

	for _, test := range tests {
		reader, err := NewTagReaderWriter(test.format)
		if test.shouldError {
			if err == nil {
				t.Errorf("NewTagReaderWriter(%v) should have returned an error", test.format)
			}
		} else {
			if err != nil {
				t.Errorf("NewTagReaderWriter(%v) returned unexpected error: %v", test.format, err)
			}
			if reader == nil {
				t.Errorf("NewTagReaderWriter(%v) returned nil reader", test.format)
			}
		}
	}
}

// TestAudioTagsStruct tests the AudioTags struct
func TestAudioTagsStruct(t *testing.T) {
	tags := &AudioTags{
		Title:       "Test Title",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Genre:       "Test Genre",
		Year:        2023,
		Track:       5,
		TrackTotal:  12,
		Disc:        1,
		DiscTotal:   2,
		Composer:    "Test Composer",
		Comment:     "Test Comment",
	}

	// Test that all fields are properly set
	if tags.Title != "Test Title" {
		t.Errorf("Title not set correctly")
	}
	if tags.Artist != "Test Artist" {
		t.Errorf("Artist not set correctly")
	}
	if tags.Album != "Test Album" {
		t.Errorf("Album not set correctly")
	}
	if tags.AlbumArtist != "Test Album Artist" {
		t.Errorf("AlbumArtist not set correctly")
	}
	if tags.Genre != "Test Genre" {
		t.Errorf("Genre not set correctly")
	}
	if tags.Year != 2023 {
		t.Errorf("Year not set correctly")
	}
	if tags.Track != 5 {
		t.Errorf("Track not set correctly")
	}
	if tags.TrackTotal != 12 {
		t.Errorf("TrackTotal not set correctly")
	}
	if tags.Disc != 1 {
		t.Errorf("Disc not set correctly")
	}
	if tags.DiscTotal != 2 {
		t.Errorf("DiscTotal not set correctly")
	}
	if tags.Composer != "Test Composer" {
		t.Errorf("Composer not set correctly")
	}
	if tags.Comment != "Test Comment" {
		t.Errorf("Comment not set correctly")
	}
}

// TestReadTagsWithInvalidFile tests reading tags from non-existent files
func TestReadTagsWithInvalidFile(t *testing.T) {
	invalidFiles := []string{
		"/nonexistent/file.mp3",
		"/nonexistent/file.flac",
		"/nonexistent/file.wav",
		"/nonexistent/file.aiff",
		"test.txt", // unsupported format
	}

	for _, file := range invalidFiles {
		_, err := ReadTags(file)
		if err == nil {
			t.Errorf("ReadTags(%s) should have returned an error", file)
		}
	}
}

// TestWriteTagsWithInvalidFile tests writing tags to non-existent files
func TestWriteTagsWithInvalidFile(t *testing.T) {
	tags := &AudioTags{
		Title:  "Test",
		Artist: "Test",
		Genre:  "Test",
	}

	invalidFiles := []string{
		"/nonexistent/file.mp3",
		"/nonexistent/file.flac",
		"/nonexistent/file.wav",
		"/nonexistent/file.aiff",
		"test.txt", // unsupported format
	}

	for _, file := range invalidFiles {
		err := WriteTags(file, tags)
		if err == nil {
			t.Errorf("WriteTags(%s) should have returned an error", file)
		}
	}
}

// TestHasGenreWithInvalidFile tests HasGenre with invalid files
func TestHasGenreWithInvalidFile(t *testing.T) {
	_, err := HasGenre("/nonexistent/file.mp3")
	if err == nil {
		t.Error("HasGenre with non-existent file should return an error")
	}

	_, err = HasGenre("test.txt")
	if err == nil {
		t.Error("HasGenre with unsupported format should return an error")
	}
}

// createTestFile creates a temporary file for testing
func createTestFile(t *testing.T, extension string) string {
	t.Helper()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test"+extension)
	
	// Create an empty file
	file, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()
	
	return testFile
}

// BenchmarkDetectFormat benchmarks the format detection
func BenchmarkDetectFormat(b *testing.B) {
	filename := "test.mp3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DetectFormat(filename)
	}
}

// BenchmarkIsAudioFile benchmarks the audio file detection
func BenchmarkIsAudioFile(b *testing.B) {
	filename := "test.mp3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsAudioFile(filename)
	}
}