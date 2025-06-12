package audiotag

import (
	"encoding/binary"
	"os"
	"path/filepath"
)

// GenerateTestFiles creates test audio files in the specified directory
func GenerateTestFiles(baseDir string) error {
	testDir := filepath.Join(baseDir, "testfiles")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		return err
	}

	files := []struct {
		name      string
		generator func(string) error
	}{
		{"test.mp3", generateMP3File},
		{"test.flac", generateFLACFile},
		{"test.wav", generateWAVFile},
		{"test.aiff", generateAIFFFile},
	}

	for _, file := range files {
		filePath := filepath.Join(testDir, file.name)
		if err := file.generator(filePath); err != nil {
			return err
		}
	}

	return nil
}

// generateMP3File creates a minimal valid MP3 file with ID3v2 header
func generateMP3File(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write minimal ID3v2.3 header
	id3Header := []byte{
		'I', 'D', '3',    // ID3 identifier
		0x03, 0x00,       // Version (3.0)
		0x00,             // Flags
		0x00, 0x00, 0x00, 0x00, // Size (syncsafe int, 0 for now)
	}
	
	if _, err := file.Write(id3Header); err != nil {
		return err
	}

	// Write minimal MP3 frame header
	mpegHeader := []byte{
		0xFF, 0xFB, 0x90, 0x00, // MPEG-1 Layer 3, 128kbps, 44.1kHz, mono
	}
	
	if _, err := file.Write(mpegHeader); err != nil {
		return err
	}

	// Add some dummy audio data (silence)
	dummyData := make([]byte, 1024)
	if _, err := file.Write(dummyData); err != nil {
		return err
	}

	return nil
}

// generateWAVFile creates a minimal valid WAV file
func generateWAVFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// WAV file header
	sampleRate := uint32(44100)
	bitsPerSample := uint16(16)
	channels := uint16(1)
	byteRate := sampleRate * uint32(channels) * uint32(bitsPerSample) / 8
	blockAlign := channels * bitsPerSample / 8
	
	// Number of samples (1 second of audio)
	numSamples := sampleRate
	dataSize := numSamples * uint32(channels) * uint32(bitsPerSample) / 8
	
	// RIFF header
	if err := writeString(file, "RIFF"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(36+dataSize)); err != nil {
		return err
	}
	if err := writeString(file, "WAVE"); err != nil {
		return err
	}

	// fmt chunk
	if err := writeString(file, "fmt "); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(16)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint16(1)); err != nil { // PCM format
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, channels); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, sampleRate); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}

	// data chunk
	if err := writeString(file, "data"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, dataSize); err != nil {
		return err
	}

	// Write silence (zeros)
	silence := make([]byte, dataSize)
	if _, err := file.Write(silence); err != nil {
		return err
	}

	return nil
}

// generateAIFFFile creates a minimal valid AIFF file
func generateAIFFFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// AIFF file header
	sampleRate := uint32(44100)
	bitsPerSample := uint16(16)
	channels := uint16(1)
	
	// Number of samples (1 second of audio)
	numSamples := sampleRate
	dataSize := numSamples * uint32(channels) * uint32(bitsPerSample) / 8
	
	// FORM header
	if err := writeString(file, "FORM"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(4+26+16+dataSize)); err != nil {
		return err
	}
	if err := writeString(file, "AIFF"); err != nil {
		return err
	}

	// COMM chunk
	if err := writeString(file, "COMM"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(18)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, channels); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, numSamples); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, bitsPerSample); err != nil {
		return err
	}
	
	// Sample rate in IEEE 754 extended precision (80-bit)
	extendedFloat := []byte{0x40, 0x0E, 0xAC, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if _, err := file.Write(extendedFloat); err != nil {
		return err
	}

	// SSND chunk
	if err := writeString(file, "SSND"); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(8+dataSize)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(0)); err != nil { // offset
		return err
	}
	if err := binary.Write(file, binary.BigEndian, uint32(0)); err != nil { // block size
		return err
	}

	// Write silence (zeros)
	silence := make([]byte, dataSize)
	if _, err := file.Write(silence); err != nil {
		return err
	}

	return nil
}

// generateFLACFile creates a minimal valid FLAC file
func generateFLACFile(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// FLAC signature
	if err := writeString(file, "fLaC"); err != nil {
		return err
	}

	// STREAMINFO metadata block (mandatory first block)
	blockHeader := []byte{0x80, 0x00, 0x00, 0x22} // last block, STREAMINFO, 34 bytes
	if _, err := file.Write(blockHeader); err != nil {
		return err
	}

	// STREAMINFO data (34 bytes)
	streamInfo := make([]byte, 34)
	// Min/Max block size
	binary.BigEndian.PutUint16(streamInfo[0:2], 4096)
	binary.BigEndian.PutUint16(streamInfo[2:4], 4096)
	// Sample rate, channels, bits per sample
	sampleRateAndChannels := uint32(44100<<12) | uint32(0<<9) | uint32(15<<4)
	binary.BigEndian.PutUint32(streamInfo[10:14], sampleRateAndChannels)
	
	if _, err := file.Write(streamInfo); err != nil {
		return err
	}

	// Add a minimal frame
	frame := []byte{0xFF, 0xF8, 0xC8, 0x00}
	if _, err := file.Write(frame); err != nil {
		return err
	}

	return nil
}

// writeString writes a string as bytes to the file
func writeString(file *os.File, s string) error {
	_, err := file.Write([]byte(s))
	return err
}