package scheduler

import (
	"context"
	"errors"

	"github.com/mstrhakr/audible-plex-downloader/internal/library"
	"github.com/mstrhakr/audible-plex-downloader/internal/logging"
	"github.com/robfig/cron/v3"
)

var schedLog = logging.Component("scheduler")

// Scheduler manages periodic tasks using cron expressions.
type Scheduler struct {
	cron      *cron.Cron
	syncSvc   *library.SyncService
	dlMgr     *library.DownloadManager
	syncEntry cron.EntryID
}

// New creates a new scheduler.
func New(syncSvc *library.SyncService, dlMgr *library.DownloadManager) *Scheduler {
	return &Scheduler{
		cron:    cron.New(),
		syncSvc: syncSvc,
		dlMgr:   dlMgr,
	}
}

// SetSyncSchedule configures the library sync cron schedule.
// Pass an empty string to disable.
func (s *Scheduler) SetSyncSchedule(schedule string) error {
	// Remove previous entry if set
	if s.syncEntry != 0 {
		s.cron.Remove(s.syncEntry)
		s.syncEntry = 0
		schedLog.Info().Msg("removed previous sync schedule")
	}

	if schedule == "" {
		schedLog.Info().Msg("sync schedule disabled")
		return nil
	}

	id, err := s.cron.AddFunc(schedule, func() {
		s.runSync()
	})
	if err != nil {
		schedLog.Error().Err(err).Str("schedule", schedule).Msg("invalid cron expression")
		return err
	}

	s.syncEntry = id
	schedLog.Info().Str("schedule", schedule).Msg("sync schedule configured")
	return nil
}

func (s *Scheduler) runSync() {
	schedLog.Info().Msg("scheduled sync starting")
	ctx := context.Background()

	added, err := s.syncSvc.Sync(ctx)
	if err != nil {
		if errors.Is(err, library.ErrSyncInProgress) {
			schedLog.Info().Msg("sync already running, skipping scheduled run")
			return
		}
		schedLog.Error().Err(err).Msg("scheduled sync failed")
		return
	}
	schedLog.Info().Int("added", added).Msg("scheduled sync complete")

	if added > 0 {
		queued, err := s.dlMgr.QueueNewBooks(ctx)
		if err != nil {
			schedLog.Error().Err(err).Msg("failed to queue new books after sync")
			return
		}
		schedLog.Info().Int("queued", queued).Msg("queued new books after sync")
	}
}

// Start begins the cron scheduler.
func (s *Scheduler) Start() {
	schedLog.Info().Msg("starting scheduler")
	s.cron.Start()
}

// Stop gracefully stops the scheduler, waiting for running jobs.
func (s *Scheduler) Stop() {
	schedLog.Info().Msg("stopping scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	schedLog.Info().Msg("scheduler stopped")
}
