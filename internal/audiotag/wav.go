package audiotag

import (
	"github.com/wtolson/go-taglib"
)

// WAVTagReaderWriter handles WAV file tag operations using go-taglib
type WAVTagReaderWriter struct{}

// NewWAVTagReaderWriter creates a new WAV tag reader/writer
func NewWAVTagReaderWriter() *WAVTagReaderWriter {
	return &WAVTagReaderWriter{}
}

// ReadTags reads tags from a WAV file
func (w *WAVTagReaderWriter) ReadTags(filePath string) (*AudioTags, error) {
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

// WriteTags writes tags to a WAV file
func (w *WAVTagReaderWriter) WriteTags(filePath string, tags *AudioTags) error {
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