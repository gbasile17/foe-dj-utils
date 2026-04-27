# DJ Utils

Apple Music is, somewhat unfortunately, the best option available for keeping a local file-based music library synced across multiple devices while also doubling as a playlist manager for DJ software like Serato and Rekordbox. The combination of cloud sync and local lossless files is unique among consumer music apps — but the cost is that managing the library across synced devices comes with a long list of quirks: playlists that silently desync from iCloud Music Library, tracks that get flagged as `duplicate` and refuse to sync, lossless originals that get matched against lower-quality catalog versions, orphaned references to files that no longer exist on disk, and so on.

This command-line utility aims to solve some of these issues. It provides a set of focused tools for working *around* Music.app rather than fighting it: detecting and recreating desynced playlists, quarantining tracks whose cloud status blocks sync, separating lossless local files from cloud-synced copies, and the usual file-and-tag housekeeping you need on a DJ-oriented collection.

The binary is named `foe`.

## Features

### File / metadata
- **Find Duplicates** — identify duplicate audio files across directories using both file hash and title matching
- **Search Titles** — search for audio files by title across multiple directories
- **Search Artists** — find audio files by artist name
- **FLAC to AIFF Conversion** — convert FLAC files to AIFF while preserving metadata
- **Artist Tag Cleanup** — fix artist tags with number prefixes (e.g., `101. CoolTasty` → `CoolTasty`)
- **Genre Tagging** — automatically tag audio files with genre information from MusicBrainz

### Apple Music / iCloud Music Library
- **Export Playlist** — copy a playlist's files (and optional M3U / zip) to a directory
- **Orphans** — find and remove tracks whose files no longer exist on disk
- **Recover** — find audio files on disk that aren't in the library and add them
- **Resync Playlists** — recreate playlists to force iCloud Music Library to re-upload them (fixes desynced playlists)
- **Quarantine Blockers** — move tracks whose `cloud status` blocks iCloud sync (`ineligible`, `error`, `removed`, `duplicate`, `no longer available`) into a single `_block` playlist for triage
- **Resolve Duplicates** — inspect tracks flagged as `duplicate` by iCloud and search the library for the canonical copy
- **Dedupe** — copy duplicate-flagged files to `~/Music/duplicates`, then remove them from the library and disk so iCloud's canonical copy can sync cleanly

## Installation

### Prerequisites

You'll need the following installed on your system:

1. **Go** — see the [official Go installation guide](https://golang.org/doc/install)
2. **TagLib** — required for tag manipulation
3. **ffmpeg** — required for FLAC → AIFF conversion
4. **macOS Music.app** — required for any `apple-*` command

#### Installing TagLib

**macOS (Homebrew):**
```bash
brew install taglib
```

**Ubuntu/Debian:**
```bash
sudo apt-get install libtag1-dev
```

**Fedora/RHEL:**
```bash
sudo dnf install taglib-devel
```

**Windows:** download dev libraries from [taglib.org](https://taglib.org/).

### Building from source

```bash
go build -o ~/.local/bin/foe .
```

Make sure `~/.local/bin` is on your `$PATH`. The build downloads dependencies on first run; you can also pre-fetch them with `go mod download`.

## Usage

Run with no arguments to see all available commands:

```bash
foe
```

### File / metadata commands

#### Find Duplicates

```bash
foe detect-duplicates /path/to/music/dir1 /path/to/music/dir2
```

Analyzes files using both MD5 hash matching (exact duplicates) and title matching (different versions of the same song).

#### Search by Title

```bash
foe title-search /path/to/music "song title"
```

#### Search by Artist

```bash
foe artist-search /path/to/music "artist name"
```

#### Convert FLAC to AIFF

```bash
foe flac-to-aiff /path/to/flac/files
```

Options:
- `-r, --recursive` — include subdirectories
- `-d, --remove` — delete original FLAC files after successful conversion

Requires `ffmpeg`.

#### Fix Artist Tags

```bash
foe fix-artist-tags /path/to/music
```

Scans for `101. Artist Name`-style prefixes, previews the changes, and asks for confirmation before writing.

#### Genre Tagging

```bash
foe genre-tag /path/to/music
```

Looks up missing genre tags via MusicBrainz (rate-limited to 1 req/sec) and asks for confirmation before applying.

### Apple Music commands

All `apple-*` commands talk to Music.app through `osascript` (AppleScript) and require macOS with the Music app available.

#### Export Playlist

```bash
foe apple-export-playlist "My Playlist" ./export
foe aep "House" ~/Music/exports/house
```

Copies the playlist's audio files to a directory with a track-number prefix to preserve order. Optional flags: `--m3u`, `--zip`, `--skip-drm`.

#### Orphans

```bash
foe apple-orphans
```

Finds and removes tracks whose underlying files are missing from disk.

#### Recover

```bash
foe apple-recover ~/Music/Music/Media
foe ar --playlist "Found Files"
```

Scans a directory for audio files not in the library and offers to add them, optionally to a named playlist.

#### Resync Playlists (`arp`)

```bash
foe apple-resync-playlists -a              # all user playlists
foe apple-resync-playlists -p "Tech House" -p "Afro"
```

For each target: clones the tracks into a new playlist, verifies the count, deletes the original, and renames the clone to the original name. The new persistent ID forces iCloud Music Library to re-upload the playlist — useful when a playlist has desynced from the cloud.

The rename step polls from Go (not AppleScript) because Music.app's name index lags after a delete and a `delay` inside AppleScript would block the event loop that needs to drain.

#### Quarantine Blockers (`aqb`)

```bash
foe apple-quarantine-blockers --dry-run
foe apple-quarantine-blockers
```

Scans every user playlist for tracks whose `cloud status` prevents iCloud sync and is unlikely to clear on its own (`ineligible`, `error`, `removed`, `duplicate`, `no longer available` — `not uploaded` is intentionally left alone since Apple's matcher may not have run yet). Moves them into a single `_block` playlist, de-duped by persistent ID. Tracks remain in your library — only their membership in source playlists changes.

By default skips `_lib` (the master playlist). Override with `--skip`. The dry-run output is a tree showing each blocker with its status and title, so you can triage before doing anything.

#### Resolve Duplicates (`ard`)

```bash
foe apple-resolve-duplicates --inspect
```

Read-only. For every track flagged as `cloud status = duplicate`, strips the DJ key/BPM prefix and `(... Mix)` suffix from the title and searches the library for the canonical copy iCloud is matching against. Prints both sides (file path, format, size, cloud status) so you can decide what to keep.

In practice this often reveals that iCloud is matching against the Apple Music *catalog* rather than another local file — meaning the "duplicate" is your local lossless copy and the canonical is a stream you don't own.

#### Dedupe (`adp`)

```bash
foe apple-dedupe --dry-run
foe apple-dedupe
foe adp --dest ~/Music/dj-duplicates
```

For each `cloud status = duplicate` track: copies the underlying file to `~/Music/duplicates` (flat layout), removes the track from the Apple Music library, and deletes the original on disk. Use this when you want to keep your lossless source files outside the library while letting iCloud sync the canonical copy. You can re-add the quarantined files later if you want them back.

## Supported audio formats

MP3, FLAC, AIFF, WAV, M4A, OGG.

## Dependencies

- **github.com/spf13/cobra** — CLI framework
- **github.com/fatih/color** — colored terminal output
- **github.com/bogem/id3v2** — ID3 tag manipulation
- **github.com/dhowden/tag** — audio metadata reading
- **github.com/wtolson/go-taglib** — TagLib bindings for Go
- **github.com/michiwend/gomusicbrainz** — MusicBrainz API client

## External dependencies

- **TagLib** — required for tag operations
- **ffmpeg** — required for FLAC → AIFF
- **macOS Music.app** — required for any `apple-*` command
- **MusicBrainz API** — used by `genre-tag` (rate-limited to 1 req/sec)

## Development

### Project structure

```
dj-utils/
├── main.go                # entry point; calls cmd.Execute()
├── cmd/                   # cobra subcommands, one file per command
│   ├── root.go
│   ├── styles.go          # shared color theme
│   ├── apple_*.go         # Apple Music commands
│   └── ...
└── internal/
    ├── applemusic/        # AppleScript / osascript wrappers for Music.app
    ├── audiotag/          # unified MP3/FLAC/WAV/AIFF tag read/write
    ├── fileutil/          # CopyFile, WriteM3U, WalkAudioFiles, ZipDirectory, …
    └── music/             # File / TitleResult / ArtistResult types + duplicate detection
```

### Building for other platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o foe-linux

# Windows
GOOS=windows GOARCH=amd64 go build -o foe.exe
```

The `apple-*` commands are macOS-only (they shell out to `osascript`).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source. Please check the license file for more details.

## Troubleshooting

- **TagLib build errors** — make sure the development libraries are installed (see Prerequisites).
- **`ffmpeg not found`** — install via your package manager.
- **`Music app is not available`** — Music.app must be installed and the user must have approved automation permission for the terminal in System Settings → Privacy & Security → Automation.
- **MusicBrainz lookup failures** — check your internet connection; the service is automatically rate-limited.
- **Playlists still named `… (resync)` after `arp`** — Music's name index is still draining iCloud deletes. Re-run the same command; phase 2 will pick up the leftovers cleanly.

### Getting help

```bash
foe <command> --help
```
