package library

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/mstrhakr/audible-plex-downloader/internal/database"
	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
	audible "github.com/mstrhakr/go-audible"
)

var syncLog = logging.Component("sync")

// ErrSyncInProgress is returned when a sync is already running.
var ErrSyncInProgress = errors.New("sync already in progress")

// SyncProgress tracks the current state of a library sync.
type SyncProgress struct {
	Running      bool
	Status       string
	Message      string
	Error        string
	BooksFound   int
	BooksScanned int
	BooksAdded   int
	StartedAt    time.Time
	CompletedAt  time.Time
}

// Percent returns progress in the range [0,1].
func (p SyncProgress) Percent() float64 {
	if p.BooksFound <= 0 {
		if p.Running {
			return 0
		}
		if p.Status == "complete" {
			return 1
		}
		return 0
	}
	percent := float64(p.BooksScanned) / float64(p.BooksFound)
	if percent < 0 {
		return 0
	}
	if percent > 1 {
		return 1
	}
	return percent
}

// SyncService handles syncing the Audible library to the local database.
type SyncService struct {
	db     database.Database
	client *audible.Client

	libraryDir string

	mu       sync.RWMutex
	progress SyncProgress
}

// NewSyncService creates a new library sync service.
func NewSyncService(db database.Database, client *audible.Client, libraryDir string) *SyncService {
	return &SyncService{db: db, client: client, libraryDir: libraryDir}
}

// GetProgress returns the latest sync progress snapshot.
func (s *SyncService) GetProgress() SyncProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.progress
}

// Sync fetches the full Audible library and upserts new books into the database.
// Returns the number of new books added.
func (s *SyncService) Sync(ctx context.Context) (int, error) {
	s.mu.Lock()
	if s.progress.Running {
		s.mu.Unlock()
		return 0, ErrSyncInProgress
	}
	now := time.Now()
	s.progress = SyncProgress{
		Running:   true,
		Status:    "running",
		Message:   "Starting sync...",
		StartedAt: now,
	}
	s.mu.Unlock()

	syncRecord := &database.SyncHistory{
		StartedAt: now,
		Status:    "running",
	}
	if err := s.db.CreateSync(ctx, syncRecord); err != nil {
		s.finishProgressWithError(err)
		return 0, err
	}

	added, err := s.doSync(ctx, syncRecord)
	if err != nil {
		finished := time.Now()
		syncRecord.CompletedAt = &finished
		syncRecord.Status = "failed"
		syncRecord.Error = err.Error()
		_ = s.db.UpdateSync(ctx, syncRecord)
		s.finishProgressWithError(err)
		return 0, err
	}

	if updated, err := reconcileExistingAudiobookFiles(ctx, s.db, s.libraryDir); err != nil {
		syncLog.Warn().Err(err).Msg("failed to reconcile existing audiobook files after sync")
	} else if updated > 0 {
		syncLog.Info().Int("books_reconciled", updated).Msg("reconciled audiobook files against disk after sync")
	}

	finished := time.Now()
	syncRecord.CompletedAt = &finished
	syncRecord.BooksAdded = added
	syncRecord.Status = "complete"
	_ = s.db.UpdateSync(ctx, syncRecord)

	s.mu.Lock()
	s.progress.Running = false
	s.progress.Status = "complete"
	s.progress.Message = "Sync complete"
	s.progress.CompletedAt = finished
	if s.progress.BooksFound > 0 {
		s.progress.BooksScanned = s.progress.BooksFound
	}
	s.mu.Unlock()

	return added, nil
}

func (s *SyncService) doSync(ctx context.Context, syncRecord *database.SyncHistory) (int, error) {
	syncLog.Info().Msg("starting library sync")

	books, err := s.client.GetAllLibrary(ctx)
	if err != nil {
		syncLog.Error().Err(err).Msg("failed to fetch audible library")
		return 0, err
	}

	syncRecord.BooksFound = len(books)
	s.mu.Lock()
	s.progress.BooksFound = len(books)
	s.progress.Message = "Sync in progress"
	s.mu.Unlock()
	_ = s.db.UpdateSync(ctx, syncRecord)
	syncLog.Info().Int("total_books", len(books)).Msg("fetched audible library")

	added := 0
	scanned := 0
	for _, item := range books {
		book := convertBook(item)
		syncLog.Trace().Str("asin", book.ASIN).Str("title", book.Title).Msg("processing book")

		existing, err := s.db.GetBookByASIN(ctx, book.ASIN)
		if err != nil {
			syncLog.Error().Err(err).Str("asin", book.ASIN).Msg("failed to check existing book")
			scanned++
			s.mu.Lock()
			s.progress.BooksScanned = scanned
			s.progress.BooksAdded = added
			s.mu.Unlock()
			continue
		}

		// Preserve status/file info for existing books
		if existing != nil {
			book.Status = existing.Status
			book.FilePath = existing.FilePath
			book.FileSize = existing.FileSize
			syncLog.Debug().Str("asin", book.ASIN).Str("status", string(existing.Status)).Msg("book already exists, preserving state")
		} else {
			book.Status = database.BookStatusNew
			added++
			syncLog.Info().Str("asin", book.ASIN).Str("title", book.Title).Msg("new book discovered")
		}

		if err := s.db.UpsertBook(ctx, &book); err != nil {
			syncLog.Error().Err(err).Str("asin", book.ASIN).Msg("failed to upsert book")
			scanned++
			s.mu.Lock()
			s.progress.BooksScanned = scanned
			s.progress.BooksAdded = added
			s.mu.Unlock()
			if scanned%20 == 0 {
				syncRecord.BooksAdded = added
				_ = s.db.UpdateSync(ctx, syncRecord)
			}
			continue
		}

		scanned++
		s.mu.Lock()
		s.progress.BooksScanned = scanned
		s.progress.BooksAdded = added
		s.mu.Unlock()
		if scanned%20 == 0 {
			syncRecord.BooksAdded = added
			_ = s.db.UpdateSync(ctx, syncRecord)
		}
	}

	syncLog.Info().Int("added", added).Int("total", len(books)).Msg("library sync complete")
	return added, nil
}

func (s *SyncService) finishProgressWithError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress.Running = false
	s.progress.Status = "failed"
	s.progress.Message = "Sync failed"
	s.progress.Error = err.Error()
	s.progress.CompletedAt = time.Now()
}

func convertBook(b audible.Book) database.Book {
	authors := make([]string, len(b.Authors))
	for i, a := range b.Authors {
		authors[i] = a.Name
	}
	narrators := make([]string, len(b.Narrators))
	for i, n := range b.Narrators {
		narrators[i] = n.Name
	}

	var authorASIN string
	if len(b.Authors) > 0 {
		authorASIN = b.Authors[0].ASIN
	}

	var series, seriesPos string
	if len(b.Series) > 0 {
		series = b.Series[0].Title
		seriesPos = b.Series[0].Sequence
	}

	coverURL := b.ProductImages.Image2400
	if coverURL == "" {
		coverURL = b.ProductImages.Image1024
	}
	if coverURL == "" {
		coverURL = b.ProductImages.Image500
	}

	purchaseDate, _ := time.Parse("2006-01-02", b.PurchaseDate)
	releaseDate, _ := time.Parse("2006-01-02", b.ReleaseDate)

	drmType := b.ContentDeliveryType
	if drmType == "" {
		drmType = b.FormatType
	}

	return database.Book{
		ASIN:           b.ASIN,
		Title:          b.Title,
		Author:         strings.Join(authors, ", "),
		AuthorASIN:     authorASIN,
		Narrator:       strings.Join(narrators, ", "),
		Publisher:      b.Publisher,
		Description:    b.PublisherSummary,
		Duration:       int64(b.RuntimeMinutes) * 60,
		Series:         series,
		SeriesPosition: seriesPos,
		CoverURL:       coverURL,
		PurchaseDate:   purchaseDate,
		ReleaseDate:    releaseDate,
		DRMType:        drmType,
	}
}
