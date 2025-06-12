package audiotag

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestTestFileStateManagement validates that test files can be properly managed between tests
func TestTestFileStateManagement(t *testing.T) {
	tmpDir := t.TempDir()
	
	t.Run("InitialGeneration", func(t *testing.T) {
		// Generate test files for the first time
		err := GenerateTestFiles(tmpDir)
		if err != nil {
			t.Fatalf("Failed to generate test files: %v", err)
		}
		
		testFilesDir := filepath.Join(tmpDir, "testfiles")
		expectedFiles := []string{"test.mp3", "test.flac", "test.wav", "test.aiff"}
		
		for _, filename := range expectedFiles {
			filePath := filepath.Join(testFilesDir, filename)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File %s was not created", filename)
			}
		}
	})
	
	t.Run("FileRefresh", func(t *testing.T) {
		testFilesDir := filepath.Join(tmpDir, "testfiles")
		mp3File := filepath.Join(testFilesDir, "test.mp3")
		
		// Get initial file info
		initialInfo, err := os.Stat(mp3File)
		if err != nil {
			t.Fatalf("Failed to stat initial file: %v", err)
		}
		
		// Refresh the specific file
		err = generateMP3File(mp3File)
		if err != nil {
			t.Fatalf("Failed to refresh MP3 file: %v", err)
		}
		
		// Verify file still exists
		refreshedInfo, err := os.Stat(mp3File)
		if err != nil {
			t.Fatalf("File disappeared after refresh: %v", err)
		}
		
		// File should exist and be readable
		if refreshedInfo.Size() == 0 {
			t.Error("Refreshed file is empty")
		}
		
		t.Logf("Initial file size: %d, Refreshed file size: %d", initialInfo.Size(), refreshedInfo.Size())
	})
	
	t.Run("MultipleRefreshes", func(t *testing.T) {
		testFilesDir := filepath.Join(tmpDir, "testfiles")
		mp3File := filepath.Join(testFilesDir, "test.mp3")
		
		// Refresh multiple times to ensure stability
		for i := 0; i < 3; i++ {
			err := generateMP3File(mp3File)
			if err != nil {
				t.Fatalf("Failed to refresh MP3 file on iteration %d: %v", i, err)
			}
			
			// Verify file is still valid
			info, err := os.Stat(mp3File)
			if err != nil {
				t.Fatalf("File invalid after refresh %d: %v", i, err)
			}
			
			if info.Size() == 0 {
				t.Errorf("File is empty after refresh %d", i)
			}
		}
	})
	
	t.Run("AllFormatsRefresh", func(t *testing.T) {
		testFilesDir := filepath.Join(tmpDir, "testfiles")
		
		// Test refreshing all format files
		formatFiles := map[string]func(string) error{
			"test.mp3":  generateMP3File,
			"test.flac": generateFLACFile,
			"test.wav":  generateWAVFile,
			"test.aiff": generateAIFFFile,
		}
		
		for filename, generator := range formatFiles {
			filePath := filepath.Join(testFilesDir, filename)
			
			// Refresh the file
			err := generator(filePath)
			if err != nil {
				t.Errorf("Failed to refresh %s: %v", filename, err)
				continue
			}
			
			// Verify file exists and is valid
			info, err := os.Stat(filePath)
			if err != nil {
				t.Errorf("File %s invalid after refresh: %v", filename, err)
				continue
			}
			
			if info.Size() == 0 {
				t.Errorf("File %s is empty after refresh", filename)
			}
			
			// Test format detection still works
			format := DetectFormat(filePath)
			if format == FormatUnknown {
				t.Errorf("Format detection failed for refreshed %s", filename)
			}
		}
	})
}

// TestConcurrentFileGeneration tests that file generation works correctly with concurrent access
func TestConcurrentFileGeneration(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Run multiple goroutines generating files simultaneously
	numGoroutines := 5
	done := make(chan error, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			subDir := filepath.Join(tmpDir, fmt.Sprintf("test_%d", id))
			err := GenerateTestFiles(subDir)
			done <- err
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		if err := <-done; err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}
	
	// Verify all directories were created with proper files
	for i := 0; i < numGoroutines; i++ {
		testFilesDir := filepath.Join(tmpDir, fmt.Sprintf("test_%d", i), "testfiles")
		expectedFiles := []string{"test.mp3", "test.flac", "test.wav", "test.aiff"}
		
		for _, filename := range expectedFiles {
			filePath := filepath.Join(testFilesDir, filename)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("File %s missing in concurrent test %d", filename, i)
			}
		}
	}
}

// TestTestFileIntegration tests integration between test file generation and tag operations
func TestTestFileIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Generate test files
	err := GenerateTestFiles(tmpDir)
	if err != nil {
		t.Fatalf("Failed to generate test files: %v", err)
	}
	
	testFilesDir := filepath.Join(tmpDir, "testfiles")
	
	t.Run("MP3Integration", func(t *testing.T) {
		mp3File := filepath.Join(testFilesDir, "test.mp3")
		
		// Create MP3 reader/writer
		mp3RW := NewMP3TagReaderWriter()
		
		// Test basic tag operations
		testTags := &AudioTags{
			Title:  "Integration Test",
			Artist: "Test Artist",
			Genre:  "Test Genre",
		}
		
		// Write tags
		err := mp3RW.WriteTags(mp3File, testTags)
		if err != nil {
			t.Logf("MP3 write failed (expected with minimal file): %v", err)
			return
		}
		
		// Read tags back
		readTags, err := mp3RW.ReadTags(mp3File)
		if err != nil {
			t.Logf("MP3 read failed: %v", err)
			return
		}
		
		// Verify some data persisted
		if readTags == nil {
			t.Error("ReadTags returned nil")
		}
		
		t.Logf("Successfully completed MP3 integration test")
	})
	
	t.Run("UnifiedAPIIntegration", func(t *testing.T) {
		// Test unified API with all formats
		formats := map[string]AudioFormat{
			"test.mp3":  FormatMP3,
			"test.flac": FormatFLAC,
			"test.wav":  FormatWAV,
			"test.aiff": FormatAIFF,
		}
		
		for filename, expectedFormat := range formats {
			filePath := filepath.Join(testFilesDir, filename)
			
			// Test format detection
			detectedFormat := DetectFormat(filePath)
			if detectedFormat != expectedFormat {
				t.Errorf("Format detection failed for %s: got %v, want %v", filename, detectedFormat, expectedFormat)
			}
			
			// Test IsAudioFile
			if !IsAudioFile(filePath) {
				t.Errorf("IsAudioFile failed for %s", filename)
			}
			
			// Test unified ReadTags (may fail with minimal files)
			tags, err := ReadTags(filePath)
			if err != nil {
				t.Logf("ReadTags failed for %s (expected): %v", filename, err)
			} else if tags == nil {
				t.Errorf("ReadTags returned nil for %s", filename)
			}
		}
	})
}

// TestCleanupBetweenTests demonstrates proper test isolation
func TestCleanupBetweenTests(t *testing.T) {
	// Each test should use t.TempDir() to get isolated temporary directories
	// This ensures no test interferes with another
	
	t.Run("Test1", func(t *testing.T) {
		tmpDir1 := t.TempDir()
		err := GenerateTestFiles(tmpDir1)
		if err != nil {
			t.Fatalf("Test1 failed to generate files: %v", err)
		}
		
		// Modify files in this test
		mp3File := filepath.Join(tmpDir1, "testfiles", "test.mp3")
		mp3RW := NewMP3TagReaderWriter()
		tags := &AudioTags{Title: "Test1 Title"}
		
		err = mp3RW.WriteTags(mp3File, tags)
		if err != nil {
			t.Logf("Test1 write failed (expected): %v", err)
		}
	})
	
	t.Run("Test2", func(t *testing.T) {
		tmpDir2 := t.TempDir()
		err := GenerateTestFiles(tmpDir2)
		if err != nil {
			t.Fatalf("Test2 failed to generate files: %v", err)
		}
		
		// This test gets fresh files, unaffected by Test1
		mp3File := filepath.Join(tmpDir2, "testfiles", "test.mp3")
		mp3RW := NewMP3TagReaderWriter()
		
		// Read tags - should be clean/empty since it's a fresh file
		tags, err := mp3RW.ReadTags(mp3File)
		if err != nil {
			t.Logf("Test2 read failed (expected): %v", err)
		} else if tags != nil && tags.Title == "Test1 Title" {
			t.Error("Test2 is contaminated by Test1 - test isolation failed")
		}
	})
}