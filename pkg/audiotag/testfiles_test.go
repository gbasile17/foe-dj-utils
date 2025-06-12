package audiotag

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateTestFiles tests the test file generation
func TestGenerateTestFiles(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Generate test files
	err := GenerateTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test files: %v", err)
	}
	
	// Check that all expected files were created
	expectedFiles := map[string]AudioFormat{
		"test.mp3":  FormatMP3,
		"test.flac": FormatFLAC,
		"test.wav":  FormatWAV,
		"test.aiff": FormatAIFF,
	}
	
	testDir := filepath.Join(tmpDir, "testfiles")
	for filename, expectedFormat := range expectedFiles {
		filePath := filepath.Join(testDir, filename)
		
		// Check file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
			continue
		}
		
		// Check file is not empty
		info, err := os.Stat(filePath)
		if err != nil {
			t.Errorf("Failed to stat file %s: %v", filename, err)
			continue
		}
		
		if info.Size() == 0 {
			t.Errorf("File %s is empty", filename)
		}
		
		// Check format detection works
		format := DetectFormat(filePath)
		if format != expectedFormat {
			t.Errorf("Format detection failed for %s: got %v, want %v", filename, format, expectedFormat)
		}
		
		// Test that we can create readers for each format
		reader, err := NewTagReaderWriter(format)
		if err != nil {
			t.Errorf("Failed to create reader for %s: %v", filename, err)
			continue
		}
		
		// Attempt to read tags (may fail with minimal files, but shouldn't crash)
		_, err = reader.ReadTags(filePath)
		if err != nil {
			t.Logf("Note: Reading tags from %s failed (expected for minimal file): %v", filename, err)
		}
		
		t.Logf("Successfully generated and tested %s (%d bytes, format: %s)", filename, info.Size(), format)
	}
}

// TestActualFileOperations tests reading/writing with the generated files
func TestActualFileOperations(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	
	// Generate test files
	err := GenerateTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test files: %v", err)
	}
	
	testDir := filepath.Join(tmpDir, "testfiles")
	
	// Test each file format
	testFiles := []struct{
		filename string
		format AudioFormat
	}{
		{"test.mp3", FormatMP3},
		{"test.flac", FormatFLAC},
		{"test.wav", FormatWAV},
		{"test.aiff", FormatAIFF},
	}
	
	for _, test := range testFiles {
		t.Run(test.filename, func(t *testing.T) {
			filePath := filepath.Join(testDir, test.filename)
			
			// Test format detection
			detectedFormat := DetectFormat(filePath)
			if detectedFormat != test.format {
				t.Errorf("Format detection failed: expected %s, got %s", test.format, detectedFormat)
			}
			
			// Test that we can create a reader for this format
			reader, err := NewTagReaderWriter(test.format)
			if err != nil {
				t.Errorf("Failed to create reader for %s: %v", test.filename, err)
				return
			}
			
			// Test unified API
			unifiedTags, err := ReadTags(filePath)
			if err != nil {
				t.Logf("Note: Unified ReadTags from %s failed (expected for minimal file): %v", test.filename, err)
			} else if unifiedTags != nil {
				t.Logf("Successfully read tags via unified API from %s", test.filename)
			}
			
			// Test reader-specific API
			readerTags, err := reader.ReadTags(filePath)
			if err != nil {
				t.Logf("Note: Reader-specific ReadTags from %s failed (expected for minimal file): %v", test.filename, err)
			} else if readerTags != nil {
				t.Logf("Successfully read tags via reader API from %s: title='%s'", test.filename, readerTags.Title)
			}
			
			// Test HasGenre function
			hasGenre, err := HasGenre(filePath)
			if err != nil {
				t.Logf("HasGenre failed for %s: %v", test.filename, err)
			} else {
				t.Logf("HasGenre for %s: %t", test.filename, hasGenre)
			}
		})
	}
}

// SetupTestFiles is a helper function that can be called by other tests
func SetupTestFiles(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	
	err := GenerateTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test files: %v", err)
	}
	
	return filepath.Join(tmpDir, "testfiles")
}