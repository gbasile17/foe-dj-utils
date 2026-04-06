package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/gbasile17/foe/dj-utils/internal/audiotag"
	"github.com/gbasile17/foe/dj-utils/internal/music"
	"github.com/michiwend/gomusicbrainz"
	"github.com/spf13/cobra"
)

// GenreTagResult represents a file that needs genre tagging
type GenreTagResult struct {
	Path      string
	Title     string
	Artist    string
	Album     string
	HasGenre  bool
	Format    audiotag.AudioFormat
	MBRecord  *gomusicbrainz.Recording
	Genres    []string
}

var genreTagCmd = &cobra.Command{
	Use:   "genre-tag [directory]",
	Short: "Tag audio files with genre information from MusicBrainz",
	Long: `Recursively scan a directory for audio files without genre tags,
lookup genre information from MusicBrainz, and optionally apply the tags.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		directory := args[0]
		
		// Validate directory exists
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			color.Red("Directory does not exist: %s", directory)
			return
		}

		fmt.Printf("Scanning directory: %s\n", directory)
		fmt.Println("Looking for audio files without genre tags...")

		// Find files without genre tags
		filesToTag, err := findFilesWithoutGenre(directory)
		if err != nil {
			color.Red("Error scanning files: %v", err)
			return
		}

		if len(filesToTag) == 0 {
			color.Green("No files found without genre tags!")
			return
		}

		fmt.Printf("\nFound %d files without genre tags\n", len(filesToTag))
		fmt.Println("Looking up genre information from MusicBrainz...")

		// Lookup genre information
		client, err := gomusicbrainz.NewWS2Client("https://musicbrainz.org/ws/2", "foe-cli", "1.0", "")
		if err != nil {
			color.Red("Error creating MusicBrainz client: %v", err)
			return
		}
		
		var foundTags, notFound []GenreTagResult
		foundTags, notFound = lookupGenres(client, filesToTag)

		// Print report
		printGenreReport(foundTags, notFound)

		// Prompt user to apply tags
		if len(foundTags) > 0 {
			if promptUser("Would you like to apply these genre tags to the files?") {
				applyGenreTags(foundTags)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(genreTagCmd)
}

// findFilesWithoutGenre recursively finds audio files without genre tags
func findFilesWithoutGenre(dir string) ([]GenreTagResult, error) {
	var results []GenreTagResult
	mu := sync.Mutex{}
	fileChan := make(chan string, 100)
	wg := sync.WaitGroup{}

	// Worker function to process files
	processFile := func() {
		defer wg.Done()
		for path := range fileChan {
			if !audiotag.IsAudioFile(path) {
				continue
			}

			result, err := analyzeFileForGenre(path)
			if err != nil {
				fmt.Printf("Failed to analyze file %s: %v\n", path, err)
				continue
			}

			// Only add files that don't have genre tags
			if !result.HasGenre {
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}
	}

	// Start worker goroutines
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go processFile()
	}

	// Walk directory and send file paths
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fileChan <- path
		}
		return nil
	})

	if err != nil {
		close(fileChan)
		return nil, fmt.Errorf("error walking directory: %v", err)
	}

	close(fileChan)
	wg.Wait()

	return results, nil
}

// analyzeFileForGenre extracts metadata and checks for genre
func analyzeFileForGenre(filePath string) (GenreTagResult, error) {
	// Detect format
	format := audiotag.DetectFormat(filePath)
	result := GenreTagResult{
		Path:   filePath,
		Format: format,
	}

	// Read tags using our audiotag library
	tags, err := audiotag.ReadTags(filePath)
	if err != nil {
		// Fallback to filename parsing
		result.Title = music.ExtractTitleFromFilename(filePath)
		result.Artist = "Unknown Artist"
		result.Album = "Unknown Album"
		result.HasGenre = false
		return result, nil
	}

	// Extract metadata
	result.Title = tags.Title
	result.Artist = tags.Artist
	result.Album = tags.Album
	result.HasGenre = tags.Genre != ""

	// Use filename as fallback for empty title
	if result.Title == "" {
		result.Title = music.ExtractTitleFromFilename(filePath)
	}
	if result.Artist == "" {
		result.Artist = "Unknown Artist"
	}
	if result.Album == "" {
		result.Album = "Unknown Album"
	}

	return result, nil
}

// lookupGenres queries MusicBrainz for genre information
func lookupGenres(client *gomusicbrainz.WS2Client, files []GenreTagResult) ([]GenreTagResult, []GenreTagResult) {
	var foundTags []GenreTagResult
	var notFound []GenreTagResult
	
	for i, file := range files {
		fmt.Printf("Looking up %d/%d: %s - %s\n", i+1, len(files), file.Artist, file.Title)
		
		// Clean title and artist for better MusicBrainz matching
		cleanTitle := cleanTitleForSearch(file.Title)
		cleanArtist := cleanArtistForSearch(file.Artist)
		
		if cleanTitle != file.Title || cleanArtist != file.Artist {
			fmt.Printf("  Cleaned: %s - %s\n", cleanArtist, cleanTitle)
		}
		
		// Search for recording
		query := fmt.Sprintf(`artist:"%s" AND recording:"%s"`, cleanArtist, cleanTitle)
		searchResp, err := client.SearchRecording(query, -1, -1)
		
		if err != nil {
			fmt.Printf("  Error searching: %v\n", err)
			notFound = append(notFound, file)
			time.Sleep(1 * time.Second) // Rate limiting
			continue
		}

		if len(searchResp.Recordings) == 0 {
			fmt.Printf("  No matches found\n")
			notFound = append(notFound, file)
			time.Sleep(1 * time.Second) // Rate limiting
			continue
		}

		// Use the first recording match
		recording := searchResp.Recordings[0]
		
		// Extract artist ID from the recording's artist credit
		var artistID gomusicbrainz.MBID
		if len(recording.ArtistCredit.NameCredits) > 0 {
			artistID = recording.ArtistCredit.NameCredits[0].Artist.ID
		}
		
		if artistID == "" {
			fmt.Printf("  No artist ID found\n")
			notFound = append(notFound, file)
			time.Sleep(1 * time.Second)
			continue
		}
		
		// Look up artist to get genre information
		artist, err := client.LookupArtist(artistID)
		if err != nil {
			fmt.Printf("  Error looking up artist: %v\n", err)
			notFound = append(notFound, file)
			time.Sleep(1 * time.Second)
			continue
		}
		
		// Extract genre tags from artist
		var genres []string
		for _, tag := range artist.Tags {
			if isGenreTag(tag.Name) && tag.Count >= 5 { // Only use tags with decent vote count
				genres = append(genres, tag.Name)
			}
		}
		
		if len(genres) > 0 {
			file.MBRecord = recording
			file.Genres = genres
			foundTags = append(foundTags, file)
			fmt.Printf("  Found genres: %s\n", strings.Join(genres, ", "))
		} else {
			fmt.Printf("  No suitable genres found in tags\n")
			notFound = append(notFound, file)
		}

		// Rate limiting - MusicBrainz allows 1 request per second
		time.Sleep(1 * time.Second)
	}

	return foundTags, notFound
}

// printGenreReport displays the results
func printGenreReport(foundTags, notFound []GenreTagResult) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	color.Green("GENRE TAGGING REPORT")
	fmt.Println(strings.Repeat("=", 50))

	if len(foundTags) > 0 {
		color.Green("\nFOUND GENRES (%d files):", len(foundTags))
		fmt.Println(strings.Repeat("-", 30))
		for _, file := range foundTags {
			fmt.Printf("File: %s (%s)\n", filepath.Base(file.Path), file.Format)
			fmt.Printf("  Artist: %s\n", file.Artist)
			fmt.Printf("  Title: %s\n", file.Title)
			color.Green("  Genres: %s", strings.Join(file.Genres, ", "))
			fmt.Println()
		}
	}

	if len(notFound) > 0 {
		color.Yellow("\nNOT FOUND (%d files):", len(notFound))
		fmt.Println(strings.Repeat("-", 20))
		for _, file := range notFound {
			fmt.Printf("File: %s (%s)\n", filepath.Base(file.Path), file.Format)
			fmt.Printf("  Artist: %s\n", file.Artist)
			fmt.Printf("  Title: %s\n", file.Title)
			fmt.Println()
		}
	}

	fmt.Printf("\nSummary: %d tagged, %d not found\n", len(foundTags), len(notFound))
}

// promptUser asks for user confirmation
func promptUser(message string) bool {
	fmt.Printf("\n%s (y/N): ", message)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// applyGenreTags writes genre tags to files
func applyGenreTags(files []GenreTagResult) {
	fmt.Println("\nApplying genre tags...")
	
	for i, file := range files {
		fmt.Printf("Tagging %d/%d: %s (%s)\n", i+1, len(files), filepath.Base(file.Path), file.Format)
		
		// For this implementation, we'll use the first genre
		// You might want to join multiple genres or let user choose
		primaryGenre := file.Genres[0]
		
		err := writeGenreTag(file.Path, primaryGenre, file.Format)
		if err != nil {
			color.Red("  Error: %v", err)
		} else {
			color.Green("  Tagged with: %s", primaryGenre)
		}
	}
	
	color.Green("\nGenre tagging complete!")
}

// writeGenreTag writes genre information to an audio file using our audiotag library
func writeGenreTag(filePath, genre string, format audiotag.AudioFormat) error {
	// Read existing tags
	existingTags, err := audiotag.ReadTags(filePath)
	if err != nil {
		// If we can't read existing tags, create new ones
		existingTags = &audiotag.AudioTags{}
	}
	
	// Update the genre
	existingTags.Genre = genre
	
	// Write tags back using our unified API
	err = audiotag.WriteTags(filePath, existingTags)
	if err != nil {
		return fmt.Errorf("failed to write genre tag: %v", err)
	}
	
	return nil
}

// isGenreTag determines if a MusicBrainz tag represents a musical genre
func isGenreTag(tagName string) bool {
	// Convert to lowercase for comparison
	tag := strings.ToLower(tagName)
	
	// Common genre indicators
	genreKeywords := []string{
		"rock", "pop", "jazz", "blues", "classical", "electronic", "hip hop", "rap",
		"country", "folk", "reggae", "metal", "punk", "alternative", "indie",
		"techno", "house", "trance", "ambient", "experimental", "world music",
		"latin", "soul", "funk", "disco", "new wave", "grunge", "progressive",
		"acoustic", "instrumental", "vocal", "dance", "r&b", "gospel", "spiritual",
	}
	
	// Check if tag contains any genre keywords
	for _, keyword := range genreKeywords {
		if strings.Contains(tag, keyword) {
			return true
		}
	}
	
	// Additional filters for common MusicBrainz genre patterns
	// Many genres end with specific suffixes
	genreSuffixes := []string{
		"music", "genre", "style",
	}
	
	for _, suffix := range genreSuffixes {
		if strings.HasSuffix(tag, suffix) {
			return true
		}
	}
	
	return false
}

// generateGenreFromContext creates basic genre suggestions based on artist/title context
// This is a simplified fallback approach while we work on proper MusicBrainz tag integration
func generateGenreFromContext(artist, title string) []string {
	var genres []string
	
	// Convert to lowercase for pattern matching
	artistLower := strings.ToLower(artist)
	titleLower := strings.ToLower(title)
	
	// Basic genre mapping based on common patterns
	genrePatterns := map[string][]string{
		"rock":       {"rock", "alternative rock", "indie rock", "classic rock"},
		"pop":        {"pop", "pop rock", "indie pop"},
		"jazz":       {"jazz", "smooth jazz", "jazz fusion"},
		"classical":  {"classical", "orchestral", "symphony"},
		"electronic": {"electronic", "techno", "house", "ambient"},
		"hip hop":    {"hip hop", "rap", "urban"},
		"country":    {"country", "folk", "americana"},
		"metal":      {"metal", "heavy metal", "hard rock"},
		"blues":      {"blues", "rhythm and blues"},
		"reggae":     {"reggae", "ska", "dub"},
	}
	
	// Check artist name for genre indicators
	for pattern, genreList := range genrePatterns {
		if strings.Contains(artistLower, pattern) {
			genres = append(genres, genreList[0]) // Use primary genre
			break
		}
	}
	
	// Check title for genre indicators if artist didn't match
	if len(genres) == 0 {
		for pattern, genreList := range genrePatterns {
			if strings.Contains(titleLower, pattern) {
				genres = append(genres, genreList[0])
				break
			}
		}
	}
	
	// Default fallback genre if no patterns match
	if len(genres) == 0 {
		genres = []string{"Popular Music"}
	}
	
	return genres
}

// cleanTitleForSearch removes key/BPM info and common DJ suffixes from titles
func cleanTitleForSearch(title string) string {
	// Title format: "KEY - BPM - TITLE"
	// Example: "2A - 130 - The One (Extended Mix)"
	
	title = strings.TrimSpace(title)
	dashParts := strings.Split(title, " - ")
	
	// Check if we have at least 3 parts (key - bpm - title)
	if len(dashParts) >= 3 {
		possibleKey := strings.TrimSpace(dashParts[0])
		possibleBPM := strings.TrimSpace(dashParts[1])
		
		// Check if first part looks like a key and second part looks like BPM
		if isMusicalKey(possibleKey) && isBPM(possibleBPM) {
			// Everything after the BPM is the actual title
			title = strings.Join(dashParts[2:], " - ")
		}
	}
	
	// Remove common DJ mix suffixes
	mixSuffixes := []string{
		"(Extended Mix)", "(Original Mix)", "(Radio Edit)", "(Club Mix)",
		"(Dub Mix)", "(Instrumental)", "(Acapella)", "(Remix)", "(Edit)",
		"(Extended)", "(Original)", "(Radio)", "(Club)", "(Dub)",
	}
	
	for _, suffix := range mixSuffixes {
		if strings.HasSuffix(title, suffix) {
			title = strings.TrimSpace(strings.TrimSuffix(title, suffix))
		}
	}
	
	return strings.TrimSpace(title)
}

// cleanArtistForSearch cleans artist names for better matching
func cleanArtistForSearch(artist string) string {
	artist = strings.TrimSpace(artist)
	
	// Take only the first artist if multiple are listed
	if strings.Contains(artist, ", ") {
		artists := strings.Split(artist, ", ")
		artist = artists[0]
	}
	
	// Remove featuring artists for cleaner search
	if strings.Contains(artist, " feat") {
		parts := strings.Split(artist, " feat")
		artist = parts[0]
	}
	if strings.Contains(artist, " ft") {
		parts := strings.Split(artist, " ft")
		artist = parts[0]
	}
	
	return strings.TrimSpace(artist)
}

// isMusicalKey checks if a string looks like a musical key (e.g., "2A", "7B", "Fm")
func isMusicalKey(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || len(s) > 3 {
		return false
	}
	
	// Camelot key system (1A-12A, 1B-12B)
	if len(s) == 2 && (s[1] == 'A' || s[1] == 'B') {
		if s[0] >= '1' && s[0] <= '9' {
			return true
		}
	}
	if len(s) == 3 && (s[2] == 'A' || s[2] == 'B') {
		if s[0] == '1' && (s[1] >= '0' && s[1] <= '2') {
			return true
		}
	}
	
	// Traditional keys (C, Dm, F#, etc.)
	if (s[0] >= 'A' && s[0] <= 'G') {
		return true
	}
	
	return false
}

// isBPM checks if a string looks like a BPM value (typically 80-200)
func isBPM(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || len(s) > 3 {
		return false
	}
	
	// Check if all characters are digits
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	// Convert to int and check range
	if bpm := parseInt(s); bpm >= 80 && bpm <= 200 {
		return true
	}
	
	return false
}

// parseInt is a simple integer parser
func parseInt(s string) int {
	result := 0
	for _, char := range s {
		if char >= '0' && char <= '9' {
			result = result*10 + int(char-'0')
		}
	}
	return result
}