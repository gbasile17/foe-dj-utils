package audiotag

import (
	"fmt"
	"log"
)

// ExampleReadTags demonstrates how to read tags from an audio file
func ExampleReadTags() {
	// This example would work with a real audio file
	tags, err := ReadTags("example.mp3")
	if err != nil {
		log.Printf("Error reading tags: %v", err)
		return
	}

	fmt.Printf("Title: %s\n", tags.Title)
	fmt.Printf("Artist: %s\n", tags.Artist)
	fmt.Printf("Album: %s\n", tags.Album)
	fmt.Printf("Genre: %s\n", tags.Genre)
	fmt.Printf("Year: %d\n", tags.Year)
}

// ExampleWriteTags demonstrates how to write tags to an audio file
func ExampleWriteTags() {
	tags := &AudioTags{
		Title:       "My Song",
		Artist:      "My Artist",
		Album:       "My Album",
		AlbumArtist: "My Album Artist",
		Genre:       "Rock",
		Year:        2023,
		Track:       1,
		TrackTotal:  10,
		Disc:        1,
		DiscTotal:   1,
		Composer:    "My Composer",
		Comment:     "My favorite song",
	}

	// This example would work with a real audio file
	err := WriteTags("example.mp3", tags)
	if err != nil {
		log.Printf("Error writing tags: %v", err)
		return
	}

	fmt.Println("Tags written successfully")
}

// ExampleHasGenre demonstrates how to check if a file has a genre tag
func ExampleHasGenre() {
	// This example would work with a real audio file
	hasGenre, err := HasGenre("example.mp3")
	if err != nil {
		log.Printf("Error checking genre: %v", err)
		return
	}

	if hasGenre {
		fmt.Println("File has a genre tag")
	} else {
		fmt.Println("File does not have a genre tag")
	}
}

// ExampleDetectFormat demonstrates format detection
func ExampleDetectFormat() {
	files := []string{
		"song.mp3",
		"track.flac",
		"audio.wav",
		"music.aiff",
		"unknown.txt",
	}

	for _, file := range files {
		format := DetectFormat(file)
		fmt.Printf("%s: %s\n", file, format)
	}
	
	// Output:
	// song.mp3: MP3
	// track.flac: FLAC
	// audio.wav: WAV
	// music.aiff: AIFF
	// unknown.txt: Unknown
}

// ExampleNewTagReaderWriter demonstrates creating format-specific readers
func ExampleNewTagReaderWriter() {
	// Create an MP3 tag reader/writer
	mp3Reader, err := NewTagReaderWriter(FormatMP3)
	if err != nil {
		log.Printf("Error creating MP3 reader: %v", err)
		return
	}

	// Use it to read tags (would work with a real file)
	_, err = mp3Reader.ReadTags("example.mp3")
	if err != nil {
		log.Printf("Error reading MP3 tags: %v", err)
		return
	}

	fmt.Println("Successfully created MP3 tag reader")
}