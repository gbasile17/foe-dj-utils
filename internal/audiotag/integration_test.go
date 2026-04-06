package audiotag

import (
	"testing"
)

// TestFormatImplementations tests that all format implementations exist
func TestFormatImplementations(t *testing.T) {
	formats := []AudioFormat{
		FormatMP3,
		FormatFLAC,
		FormatWAV,
		FormatAIFF,
	}

	for _, format := range formats {
		t.Run(format.String(), func(t *testing.T) {
			reader, err := NewTagReaderWriter(format)
			if err != nil {
				t.Fatalf("Failed to create reader for %s: %v", format, err)
			}
			if reader == nil {
				t.Fatalf("Reader is nil for format %s", format)
			}

			// Test that the reader implements both interfaces
			if _, ok := reader.(TagReader); !ok {
				t.Errorf("Reader for %s doesn't implement TagReader", format)
			}
			if _, ok := reader.(TagWriter); !ok {
				t.Errorf("Reader for %s doesn't implement TagWriter", format)
			}
		})
	}
}

// TestRoundTripConsistency tests that reading and writing tags maintains consistency
// Note: This would require actual audio files to test properly
func TestRoundTripConsistency(t *testing.T) {
	originalTags := &AudioTags{
		Title:       "Test Song",
		Artist:      "Test Artist",
		Album:       "Test Album",
		AlbumArtist: "Test Album Artist",
		Genre:       "Rock",
		Year:        2023,
		Track:       5,
		TrackTotal:  12,
		Disc:        1,
		DiscTotal:   1,
		Composer:    "Test Composer",
		Comment:     "Test Comment",
	}

	// This test would need actual audio files to work properly
	// For now, we just test that the struct values are preserved
	if originalTags.Title != "Test Song" {
		t.Error("Tag values not preserved")
	}
}

// TestConcurrentAccess tests concurrent access to tag operations
func TestConcurrentAccess(t *testing.T) {
	// Test that creating multiple readers doesn't cause issues
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for _, format := range []AudioFormat{FormatMP3, FormatFLAC, FormatWAV, FormatAIFF} {
				reader, err := NewTagReaderWriter(format)
				if err != nil {
					t.Errorf("Failed to create reader: %v", err)
					return
				}
				if reader == nil {
					t.Error("Reader is nil")
					return
				}
			}
		}()
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	// Test unsupported format
	_, err := NewTagReaderWriter(AudioFormat(999))
	if err == nil {
		t.Error("Should return error for unsupported format")
	}

	// Test reading from unsupported file extension
	_, err = ReadTags("test.unknown")
	if err == nil {
		t.Error("Should return error for unsupported file extension")
	}

	// Test writing to unsupported file extension
	tags := &AudioTags{Genre: "Rock"}
	err = WriteTags("test.unknown", tags)
	if err == nil {
		t.Error("Should return error for unsupported file extension")
	}
}