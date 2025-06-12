package audiotag

import (
	"strconv"
	"strings"

	"github.com/bogem/id3v2/v2"
)

// MP3TagReaderWriter handles MP3 file tag operations using ID3v2
type MP3TagReaderWriter struct{}

// NewMP3TagReaderWriter creates a new MP3 tag reader/writer
func NewMP3TagReaderWriter() *MP3TagReaderWriter {
	return &MP3TagReaderWriter{}
}

// ReadTags reads ID3v2 tags from an MP3 file
func (mp3 *MP3TagReaderWriter) ReadTags(filePath string) (*AudioTags, error) {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return nil, err
	}
	defer tag.Close()

	tags := &AudioTags{
		Title:       tag.Title(),
		Artist:      tag.Artist(),
		Album:       tag.Album(),
		AlbumArtist: tag.GetTextFrame(tag.CommonID("Band/Orchestra/Accompaniment")).Text,
		Genre:       tag.Genre(),
		Comment:     getFirstComment(tag),
	}

	// Parse year
	if yearStr := tag.Year(); yearStr != "" {
		if year, err := strconv.Atoi(yearStr); err == nil {
			tags.Year = year
		}
	}

	// Parse track number
	if trackStr := tag.GetTextFrame(tag.CommonID("Track number/Position in set")).Text; trackStr != "" {
		parseTrackString(trackStr, &tags.Track, &tags.TrackTotal)
	}

	// Parse disc number
	if discStr := tag.GetTextFrame(tag.CommonID("Part of a set")).Text; discStr != "" {
		parseTrackString(discStr, &tags.Disc, &tags.DiscTotal)
	}

	// Parse composer
	if composerFrames := tag.GetFrames(tag.CommonID("Composer")); len(composerFrames) > 0 {
		if textFrame, ok := composerFrames[0].(id3v2.TextFrame); ok {
			tags.Composer = textFrame.Text
		}
	}

	return tags, nil
}

// WriteTags writes ID3v2 tags to an MP3 file
func (mp3 *MP3TagReaderWriter) WriteTags(filePath string, tags *AudioTags) error {
	tag, err := id3v2.Open(filePath, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()

	// Set basic tags
	tag.SetTitle(tags.Title)
	tag.SetArtist(tags.Artist)
	tag.SetAlbum(tags.Album)
	tag.SetGenre(tags.Genre)

	// Set year
	if tags.Year > 0 {
		tag.SetYear(strconv.Itoa(tags.Year))
	}

	// Set album artist
	if tags.AlbumArtist != "" {
		tag.AddTextFrame(tag.CommonID("Band/Orchestra/Accompaniment"), id3v2.EncodingUTF8, tags.AlbumArtist)
	}

	// Set track number
	if tags.Track > 0 {
		trackStr := strconv.Itoa(tags.Track)
		if tags.TrackTotal > 0 {
			trackStr += "/" + strconv.Itoa(tags.TrackTotal)
		}
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), id3v2.EncodingUTF8, trackStr)
	}

	// Set disc number
	if tags.Disc > 0 {
		discStr := strconv.Itoa(tags.Disc)
		if tags.DiscTotal > 0 {
			discStr += "/" + strconv.Itoa(tags.DiscTotal)
		}
		tag.AddTextFrame(tag.CommonID("Part of a set"), id3v2.EncodingUTF8, discStr)
	}

	// Set composer
	if tags.Composer != "" {
		tag.AddTextFrame(tag.CommonID("Composer"), id3v2.EncodingUTF8, tags.Composer)
	}

	// Set comment
	if tags.Comment != "" {
		tag.AddCommentFrame(id3v2.CommentFrame{
			Encoding:    id3v2.EncodingUTF8,
			Language:    "eng",
			Description: "",
			Text:        tags.Comment,
		})
	}

	return tag.Save()
}

// getFirstComment extracts the first comment from ID3v2 tag
func getFirstComment(tag *id3v2.Tag) string {
	comments := tag.GetFrames(tag.CommonID("Comments"))
	if len(comments) > 0 {
		if commentFrame, ok := comments[0].(id3v2.CommentFrame); ok {
			return commentFrame.Text
		}
	}
	return ""
}

// parseTrackString parses strings like "3/12" into track number and total
func parseTrackString(trackStr string, track, total *int) {
	parts := strings.Split(trackStr, "/")
	if len(parts) >= 1 {
		if num, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil {
			*track = num
		}
	}
	if len(parts) >= 2 {
		if num, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			*total = num
		}
	}
}