# DJ Utils

A command-line utility for managing DJ music collections. This tool provides various functions for audio file management, metadata manipulation, and collection organization.

## Features

- **Find Duplicates** - Identify duplicate audio files across directories using both file hash and title matching
- **Search Titles** - Search for audio files by title across multiple directories
- **Search Artists** - Find audio files by artist name
- **FLAC to AIFF Conversion** - Convert FLAC files to AIFF format while preserving metadata
- **Artist Tag Cleanup** - Fix artist tags that contain number prefixes (e.g., "101. CoolTasty" → "CoolTasty")
- **Genre Tagging** - Automatically tag audio files with genre information from MusicBrainz

## Installation

### Prerequisites

You'll need the following installed on your system:

1. **Go** - If you don't have Go installed, follow the [official Go installation guide](https://golang.org/doc/install)
2. **TagLib** - Audio metadata library required for tag manipulation. Install from [taglib.org](https://taglib.org/)

#### Installing TagLib

**macOS (using Homebrew):**
```bash
brew install taglib
```

**Ubuntu/Debian:**
```bash
sudo apt-get install libtag1-dev
```

**CentOS/RHEL/Fedora:**
```bash
sudo yum install taglib-devel
# or for newer versions:
sudo dnf install taglib-devel
```

**Windows:**
Download and install the development libraries from [taglib.org](https://taglib.org/)

### Building from Source

1. Clone or download this repository
2. Navigate to the project directory
3. Build the application:

```bash
go build -o dj-utils
```

This will create an executable named `dj-utils` in the current directory.

### Installing Dependencies

The application will automatically download and install its dependencies when you run `go build`. However, you can also run:

```bash
go mod download
```

## Usage

Run the application without arguments to see all available commands:

```bash
./dj-utils
```

### Find Duplicates

Identify duplicate audio files across multiple directories:

```bash
./dj-utils find-duplicates /path/to/music/dir1 /path/to/music/dir2
```

This command analyzes files using both MD5 hash matching (for exact duplicates) and title matching (for different versions of the same song).

### Search by Title

Search for songs containing specific text in their titles:

```bash
./dj-utils search-titles /path/to/music "song title"
```

### Search by Artist

Find all songs by artists containing specific text:

```bash
./dj-utils search-artists /path/to/music "artist name"
```

### Convert FLAC to AIFF

Convert FLAC files to AIFF format while preserving metadata:

```bash
./dj-utils flac-to-aiff /path/to/flac/files
```

Options:
- `-r, --recursive`: Include subdirectories
- `-d, --remove`: Delete original FLAC files after successful conversion

**Requirements**: This command requires `ffmpeg` to be installed on your system.

### Fix Artist Tags

Clean up artist tags that have number prefixes:

```bash
./dj-utils fix-artist-tags /path/to/music
```

This command will:
1. Scan for audio files with artist tags like "101. Artist Name"
2. Show you a preview of what will be changed
3. Ask for confirmation before making changes
4. Clean the tags to just "Artist Name"

### Genre Tagging

Automatically add genre tags to audio files using MusicBrainz:

```bash
./dj-utils genre-tag /path/to/music
```

This command will:
1. Find audio files without genre tags
2. Look up genre information from MusicBrainz database
3. Show you what genres will be added
4. Ask for confirmation before applying changes

**Note**: This command includes rate limiting to respect MusicBrainz's API limits (1 request per second).

## Supported Audio Formats

The application supports the following audio formats:
- MP3
- FLAC
- AIFF
- WAV
- M4A
- OGG

## Dependencies

- **github.com/spf13/cobra** - CLI framework
- **github.com/fatih/color** - Colored terminal output
- **github.com/bogem/id3v2** - ID3 tag manipulation
- **github.com/dhowden/tag** - Audio metadata reading
- **github.com/wtolson/go-taglib** - TagLib bindings for Go
- **github.com/michiwend/gomusicbrainz** - MusicBrainz API client

## External Dependencies

- **TagLib** - Audio metadata library (required for all audio tag operations)
- **ffmpeg** - Required for FLAC to AIFF conversion

## Development

### Project Structure

```
dj-utils/
├── main.go              # Application entry point
├── cmd/                 # Command implementations
│   ├── root.go         # Root command and CLI setup
│   ├── helpers.go      # Shared helper functions
│   ├── flac_to_aiff.go # FLAC conversion command
│   ├── fix_artist_tags.go # Artist tag cleanup
│   └── genre_tag.go    # Genre tagging functionality
└── pkg/
    └── audiotag/       # Audio tagging library
```

### Building for Different Platforms

To build for different operating systems:

```bash
# For Windows
GOOS=windows GOARCH=amd64 go build -o dj-utils.exe

# For macOS
GOOS=darwin GOARCH=amd64 go build -o dj-utils-mac

# For Linux
GOOS=linux GOARCH=amd64 go build -o dj-utils-linux
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the license file for more details.

## Troubleshooting

### Common Issues

1. **Build errors related to TagLib** - Make sure TagLib development libraries are installed (see Prerequisites section)
2. **"ffmpeg not found"** - Install ffmpeg using your system's package manager
3. **"Permission denied"** - Make sure the executable has proper permissions (`chmod +x dj-utils`)
4. **"No audio files found"** - Ensure the directory contains supported audio formats
5. **MusicBrainz lookup failures** - Check your internet connection; the service includes automatic rate limiting

### Getting Help

Run any command with the `--help` flag to see detailed usage information:

```bash
./dj-utils find-duplicates --help
```