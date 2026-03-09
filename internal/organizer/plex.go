package organizer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mstrhakr/audible-plex-downloader/internal/audio"
	"github.com/mstrhakr/audible-plex-downloader/internal/audnexus"
	"github.com/mstrhakr/audible-plex-downloader/internal/database"
	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
)

var orgLog = logging.Component("organizer")

// PlexOrganizer handles organizing audiobook files into Plex-compatible structure.
type PlexOrganizer struct {
	db          database.Database
	ffmpeg      *audio.FFmpeg
	libraryRoot string
	embedCover  bool
	chapterFile bool
}

// NewPlexOrganizer creates a new Plex file organizer.
func NewPlexOrganizer(db database.Database, ffmpeg *audio.FFmpeg, libraryRoot string, embedCover, chapterFile bool) *PlexOrganizer {
	return &PlexOrganizer{
		db:          db,
		ffmpeg:      ffmpeg,
		libraryRoot: libraryRoot,
		embedCover:  embedCover,
		chapterFile: chapterFile,
	}
}

// Organize takes a decrypted audiobook file and moves it into the Plex library structure.
// Structure: {libraryRoot}/{Author}/{Title}/{Title}.m4b
// Optionally embeds metadata, cover art, and generates a chapters file.
func (o *PlexOrganizer) Organize(ctx context.Context, book *database.Book, enriched *audnexus.EnrichedBook, inputPath string) (string, error) {
	_ = ctx
	author := sanitizePath(enriched.Author())
	title := buildTitle(enriched)

	if author == "" {
		author = "Unknown Author"
	}
	if title == "" {
		title = "Unknown Title"
	}

	bookDir := filepath.Join(o.libraryRoot, author, sanitizePath(title))
	if err := os.MkdirAll(bookDir, 0750); err != nil {
		return "", fmt.Errorf("create book directory: %w", err)
	}

	ext := filepath.Ext(inputPath)
	finalPath := filepath.Join(bookDir, sanitizePath(title)+ext)

	orgLog.Info().
		Str("asin", book.ASIN).
		Str("title", enriched.Title()).
		Str("author", enriched.Author()).
		Str("dest", finalPath).
		Msg("organizing audiobook")

	// File is already decrypted and tagged earlier in the pipeline.
	if err := os.Rename(inputPath, finalPath); err != nil {
		// Cross-device rename; fall back to copy+delete.
		if err := copyFile(inputPath, finalPath); err != nil {
			return "", fmt.Errorf("move file: %w", err)
		}
		_ = os.Remove(inputPath)
	}

	// Generate chapters file
	if o.chapterFile {
		chapters := enriched.ChapterMarks()
		if len(chapters) > 0 {
			chapterPath := filepath.Join(bookDir, sanitizePath(title)+".chapters.txt")
			if err := writeChaptersFile(chapterPath, chapters); err != nil {
				orgLog.Warn().Err(err).Msg("failed to write chapters file")
			} else {
				orgLog.Debug().Str("path", chapterPath).Int("chapters", len(chapters)).Msg("chapters file written")
			}
		}
	}

	// Update book in database
	book.FilePath = finalPath
	fi, _ := os.Stat(finalPath)
	if fi != nil {
		book.FileSize = fi.Size()
	}
	book.Status = database.BookStatusComplete
	if err := o.db.UpsertBook(ctx, book); err != nil {
		orgLog.Error().Err(err).Str("asin", book.ASIN).Msg("failed to update book record")
	}

	orgLog.Info().
		Str("asin", book.ASIN).
		Str("path", finalPath).
		Int64("size", book.FileSize).
		Msg("audiobook organized successfully")

	return finalPath, nil
}

// buildTitle creates the display title, including series info if available.
func buildTitle(enriched *audnexus.EnrichedBook) string {
	title := enriched.Title()
	series := enriched.Series()
	pos := enriched.SeriesPosition()

	if series != "" && pos != "" {
		return fmt.Sprintf("%s - %s, Book %s", title, series, pos)
	}
	return title
}

// writeChaptersFile writes a Plex-compatible chapters.txt file.
// Format: HH:MM:SS.mmm Chapter Title
func writeChaptersFile(path string, chapters []audio.ChapterMark) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, ch := range chapters {
		ts := formatTimestamp(ch.StartMs)
		fmt.Fprintf(f, "%s %s\n", ts, ch.Title)
	}
	return nil
}

// formatTimestamp converts milliseconds to HH:MM:SS.mmm format.
func formatTimestamp(ms int) string {
	totalSec := ms / 1000
	millis := ms % 1000
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, millis)
}

// downloadCover downloads cover art and saves as cover.jpg in the book directory.
func downloadCover(ctx context.Context, coverURL, bookDir, titleBase string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coverURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cover download returned status %d", resp.StatusCode)
	}

	coverPath := filepath.Join(bookDir, "cover.jpg")
	out, err := os.Create(coverPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 10*1024*1024)); err != nil {
		return "", err
	}

	return coverPath, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

var unsafeChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

// sanitizePath removes characters that are unsafe in filenames.
func sanitizePath(name string) string {
	s := unsafeChars.ReplaceAllString(name, "")
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	return s
}
