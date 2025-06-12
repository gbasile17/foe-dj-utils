package audiotag

import (
	"testing"
)

func TestMP3TagReaderWriter(t *testing.T) {
	// Test creating MP3 reader/writer
	mp3Reader := NewMP3TagReaderWriter()
	if mp3Reader == nil {
		t.Fatal("NewMP3TagReaderWriter() returned nil")
	}

	// Test reading from non-existent file
	_, err := mp3Reader.ReadTags("/nonexistent/file.mp3")
	if err == nil {
		t.Error("ReadTags should fail for non-existent file")
	}

	// Test writing to non-existent file
	tags := &AudioTags{
		Title:  "Test Title",
		Artist: "Test Artist",
		Genre:  "Rock",
	}
	err = mp3Reader.WriteTags("/nonexistent/file.mp3", tags)
	if err == nil {
		t.Error("WriteTags should fail for non-existent file")
	}
}

func TestParseTrackString(t *testing.T) {
	tests := []struct {
		input         string
		expectedTrack int
		expectedTotal int
	}{
		{"5", 5, 0},
		{"5/12", 5, 12},
		{"  3  /  10  ", 3, 10},
		{"invalid", 0, 0},
		{"5/invalid", 5, 0},
		{"", 0, 0},
	}

	for _, test := range tests {
		var track, total int
		parseTrackString(test.input, &track, &total)
		
		if track != test.expectedTrack {
			t.Errorf("parseTrackString(%s): track = %d, expected %d", test.input, track, test.expectedTrack)
		}
		if total != test.expectedTotal {
			t.Errorf("parseTrackString(%s): total = %d, expected %d", test.input, total, test.expectedTotal)
		}
	}
}

// BenchmarkMP3TagReaderCreation benchmarks creating MP3 tag readers
func BenchmarkMP3TagReaderCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMP3TagReaderWriter()
	}
}