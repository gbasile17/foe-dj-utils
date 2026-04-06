package audiotag

import (
	"github.com/wtolson/go-taglib"
)

// AIFFTagReaderWriter handles AIFF file tag operations using go-taglib
type AIFFTagReaderWriter struct{}

// NewAIFFTagReaderWriter creates a new AIFF tag reader/writer
func NewAIFFTagReaderWriter() *AIFFTagReaderWriter {
	return &AIFFTagReaderWriter{}
}

// ReadTags reads tags from an AIFF file
func (a *AIFFTagReaderWriter) ReadTags(filePath string) (*AudioTags, error) {
	file, err := taglib.Read(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	audioTags := &AudioTags{
		Title:   file.Title(),
		Artist:  file.Artist(),
		Album:   file.Album(),
		Genre:   file.Genre(),
		Comment: file.Comment(),
		Year:    file.Year(),
		Track:   file.Track(),
	}

	return audioTags, nil
}

// WriteTags writes tags to an AIFF file
func (a *AIFFTagReaderWriter) WriteTags(filePath string, tags *AudioTags) error {
	file, err := taglib.Read(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if tags.Title != "" {
		file.SetTitle(tags.Title)
	}
	if tags.Artist != "" {
		file.SetArtist(tags.Artist)
	}
	if tags.Album != "" {
		file.SetAlbum(tags.Album)
	}
	if tags.Genre != "" {
		file.SetGenre(tags.Genre)
	}
	if tags.Comment != "" {
		file.SetComment(tags.Comment)
	}
	if tags.Year > 0 {
		file.SetYear(tags.Year)
	}
	if tags.Track > 0 {
		file.SetTrack(tags.Track)
	}

	return file.Save()
}