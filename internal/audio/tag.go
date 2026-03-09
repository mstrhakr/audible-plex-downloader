package audio

import (
	"fmt"
	"strings"

	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
)

var tagLog = logging.Component("tagger")

// Metadata represents audio file metadata to embed.
type Metadata struct {
	Title       string
	Author      string
	Narrator    string
	Writer      string // Narrator/writer credit (may be multiple narrators)
	Publisher   string // Publisher name
	Copyright   string // Copyright holder/year
	Language    string // ISO 639-1 language code (e.g., "en", "fr")
	Album       string // Usually same as title
	AlbumArtist string // Usually same as author
	Genre       string
	Year        string
	Comment     string // Description
	Track       string // Series position
	Disc        string
	CoverPath   string // Path to cover image to embed
}

// EmbedMetadata writes metadata tags to an M4B/M4A file using FFmpeg.
func (f *FFmpeg) EmbedMetadata(inputPath, outputPath string, meta Metadata) error {
	tagLog.Info().Str("input", inputPath).Str("title", meta.Title).Str("author", meta.Author).Msg("embedding metadata")
	args := []string{
		"-i", inputPath,
	}

	// Add cover art if provided
	if meta.CoverPath != "" {
		args = append(args,
			"-i", meta.CoverPath,
			"-map", "0:a",
			"-map", "1:v",
			"-disposition:v:0", "attached_pic",
		)
	}

	args = append(args, "-c", "copy")

	// Build metadata options
	metaArgs := buildMetadataArgs(meta)
	args = append(args, metaArgs...)

	args = append(args, "-y", outputPath)

	return f.run(args...)
}

func buildMetadataArgs(meta Metadata) []string {
	var args []string
	add := func(key, value string) {
		if value != "" {
			args = append(args, "-metadata", fmt.Sprintf("%s=%s", key, value))
		}
	}

	add("title", meta.Title)
	add("artist", meta.Author)
	add("album_artist", meta.AlbumArtist)
	add("album", meta.Album)
	add("genre", meta.Genre)
	add("date", meta.Year)
	add("comment", meta.Comment)
	add("composer", meta.Narrator)      // narrator/reader
	add("copyright", meta.Copyright)    // copyright holder/year
	add("publisher", meta.Publisher)    // publisher name
	add("language", meta.Language)      // ISO 639-1 language code
	add("description", meta.Writer)     // writer/narrator credit
	add("track", meta.Track)
	add("disc", meta.Disc)

	// Media type = Audiobook (for iTunes/iOS)
	add("media_type", "2")

	return args
}

// EmbedCover adds a cover image to an existing audio file.
func (f *FFmpeg) EmbedCover(inputPath, outputPath, coverPath string) error {
	tagLog.Info().Str("input", inputPath).Str("cover", coverPath).Msg("embedding cover art")
	return f.run(
		"-i", inputPath,
		"-i", coverPath,
		"-map", "0:a",
		"-map", "1:v",
		"-c", "copy",
		"-disposition:v:0", "attached_pic",
		"-y",
		outputPath,
	)
}

// FormatMetadataString creates a display string from metadata.
func FormatMetadataString(meta Metadata) string {
	var parts []string
	if meta.Title != "" {
		parts = append(parts, meta.Title)
	}
	if meta.Author != "" {
		parts = append(parts, "by "+meta.Author)
	}
	if meta.Narrator != "" {
		parts = append(parts, "narrated by "+meta.Narrator)
	}
	return strings.Join(parts, " ")
}
