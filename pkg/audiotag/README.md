# AudioTag Package

A unified Go library for reading and writing audio metadata tags across multiple formats.

## Supported Formats

- **MP3** - ID3v2 tags using `github.com/bogem/id3v2`
- **FLAC** - Vorbis comments using `github.com/wtolson/go-taglib`
- **WAV** - INFO chunks/ID3 tags using `github.com/wtolson/go-taglib`
- **AIFF** - ID3 tags using `github.com/wtolson/go-taglib`

## Features

- ✅ Read metadata from audio files
- ✅ Write metadata to audio files
- ✅ Automatic format detection
- ✅ Unified interface across all formats
- ✅ Support for common fields: Title, Artist, Album, Genre, Year, Track, etc.
- ✅ Thread-safe operations
- ✅ Comprehensive test coverage

## Installation

```bash
go get github.com/gbasile17/foe/cli/pkg/audiotag
```

## Dependencies

The package requires the following system dependencies for `go-taglib`:

### macOS
```bash
brew install taglib
```

### Ubuntu/Debian
```bash
sudo apt-get install libtag1-dev
```

### CentOS/RHEL
```bash
sudo yum install taglib-devel
```

## Quick Start

### Reading Tags

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/gbasile17/foe/cli/pkg/audiotag"
)

func main() {
    // Read tags from any supported audio file
    tags, err := audiotag.ReadTags("song.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Title: %s\n", tags.Title)
    fmt.Printf("Artist: %s\n", tags.Artist)
    fmt.Printf("Album: %s\n", tags.Album)
    fmt.Printf("Genre: %s\n", tags.Genre)
    fmt.Printf("Year: %d\n", tags.Year)
}
```

### Writing Tags

```go
package main

import (
    "log"
    
    "github.com/gbasile17/foe/cli/pkg/audiotag"
)

func main() {
    // Create new tags
    tags := &audiotag.AudioTags{
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
        Comment:     "Amazing song",
    }
    
    // Write tags to file
    err := audiotag.WriteTags("song.mp3", tags)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Checking for Genre Tags

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/gbasile17/foe/cli/pkg/audiotag"
)

func main() {
    hasGenre, err := audiotag.HasGenre("song.mp3")
    if err != nil {
        log.Fatal(err)
    }
    
    if hasGenre {
        fmt.Println("File has a genre tag")
    } else {
        fmt.Println("File needs a genre tag")
    }
}
```

### Format Detection

```go
package main

import (
    "fmt"
    
    "github.com/gbasile17/foe/cli/pkg/audiotag"
)

func main() {
    format := audiotag.DetectFormat("song.mp3")
    fmt.Printf("Format: %s\n", format) // Output: Format: MP3
    
    isAudio := audiotag.IsAudioFile("song.mp3")
    fmt.Printf("Is audio file: %t\n", isAudio) // Output: Is audio file: true
}
```

## API Reference

### Types

```go
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

type AudioFormat int

const (
    FormatUnknown AudioFormat = iota
    FormatMP3
    FormatFLAC
    FormatWAV
    FormatAIFF
)
```

### Functions

```go
// Read tags from any supported audio file
func ReadTags(filePath string) (*AudioTags, error)

// Write tags to any supported audio file
func WriteTags(filePath string, tags *AudioTags) error

// Check if file has a genre tag
func HasGenre(filePath string) (bool, error)

// Detect audio format from file extension
func DetectFormat(filePath string) AudioFormat

// Check if file is a supported audio format
func IsAudioFile(filePath string) bool

// Create format-specific reader/writer
func NewTagReaderWriter(format AudioFormat) (TagReaderWriter, error)
```

## Testing

Run the test suite:

```bash
go test ./pkg/audiotag/...
```

Run tests with verbose output:

```bash
go test -v ./pkg/audiotag/...
```

Run benchmarks:

```bash
go test -bench=. ./pkg/audiotag/...
```

## Implementation Notes

### MP3 Files
- Uses ID3v2 tags via `github.com/bogem/id3v2`
- Supports all ID3v2 frames
- Handles track/disc numbering in "n/total" format

### FLAC Files
- Uses Vorbis comments via `github.com/wtolson/go-taglib`
- Supports standard Vorbis comment fields
- Cross-platform compatibility

### WAV Files
- Uses INFO chunks or embedded ID3 tags via `github.com/wtolson/go-taglib`
- Limited metadata support compared to other formats

### AIFF Files
- Uses embedded ID3 tags via `github.com/wtolson/go-taglib`
- Similar capabilities to WAV files

## Error Handling

The library returns descriptive errors for:
- Unsupported file formats
- File access errors
- Corrupted metadata
- Missing dependencies

Always check for errors when calling library functions.

## Contributing

1. Ensure all tests pass
2. Add tests for new functionality
3. Follow Go coding conventions
4. Update documentation

## License

This package is part of the foe CLI tool and follows the same license terms.