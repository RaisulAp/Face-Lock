package attendance

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttendanceRetentionLockID is the PostgreSQL advisory lock key for attendance retention job.
const AttendanceRetentionLockID = 20260904

// ErrRetentionLockBusy indicates that another instance is currently running retention.
var ErrRetentionLockBusy = errors.New("attendance retention job: advisory lock is already held")

// RetentionResult tracks summary statistics of a retention job run.
type RetentionResult struct {
	PhotosPurged   int        `json:"photos_purged"`
	AttemptsPurged int64      `json:"attempts_purged"`
	OldestPhoto    *time.Time `json:"oldest_photo,omitempty"`
	NewestPhoto    *time.Time `json:"newest_photo,omitempty"`
	SkippedLock    bool       `json:"skipped_lock"`
}

// RetentionJob performs scheduled purging of expired attendance photos and attempt telemetry.
type RetentionJob struct {
	db       *pgxpool.Pool
	store    storage.Store
	settings *settings.Service
	audit    *audit.Recorder
	logger   *slog.Logger
}

// NewRetentionJob creates a new RetentionJob.
func NewRetentionJob(
	db *pgxpool.Pool,
	store storage.Store,
	setSvc *settings.Service,
	aud *audit.Recorder,
	logger *slog.Logger,
) *RetentionJob {
	if logger == nil {
		logger = slog.Default()
	}
	return &RetentionJob{
		db:       db,
		store:    store,
		settings: setSvc,
		audit:    aud,
		logger:   logger,
	}
}

// RunOnce executes one cycle of retention purging protected by PostgreSQL advisory lock.
func (j *RetentionJob) RunOnce(ctx context.Context) (*RetentionResult, error) {
	var locked bool
	if err := j.db.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", AttendanceRetentionLockID).Scan(&locked); err != nil {
		return nil, fmt.Errorf("acquiring retention advisory lock: %w", err)
	}
	if !locked {
		j.logger.Info("attendance retention job skipped: lock already held by another worker")
		return &RetentionResult{SkippedLock: true}, ErrRetentionLockBusy
	}

	defer func() {
		var unlocked bool
		_ = j.db.QueryRow(context.Background(), "SELECT pg_advisory_unlock($1)", AttendanceRetentionLockID).Scan(&unlocked)
	}()

	res := &RetentionResult{}

	// Resolve retention days
	photoDays := 90
	attemptDays := 90
	if j.settings != nil {
		photoDays = j.settings.GetInt(ctx, "attendance_photo_retention_days", 90)
		if photoDays <= 0 {
			photoDays = j.settings.GetInt(ctx, "photo_retention_days", 90)
		}
		attemptDays = j.settings.GetInt(ctx, "attendance_attempt_retention_days", 90)
		if attemptDays <= 0 {
			attemptDays = j.settings.GetInt(ctx, "attempt_retention_days", 90)
		}
	}

	// 1. Purge expired attendance photos (Batch size 500)
	const fetchExpiredPhotosQ = `
		SELECT id, photo_key, server_timestamp
		FROM attendances
		WHERE photo_purged_at IS NULL
		  AND photo_key IS NOT NULL
		  AND status <> 'pending_review'
		  AND coalesce(reviewed_at, server_timestamp) < now() - make_interval(days => $1)
		LIMIT 500
	`
	rows, err := j.db.Query(ctx, fetchExpiredPhotosQ, photoDays)
	if err != nil {
		return res, fmt.Errorf("querying expired attendance photos: %w", err)
	}

	type expiredPhoto struct {
		id        uuid.UUID
		key       string
		timestamp time.Time
	}
	var expiredList []expiredPhoto

	for rows.Next() {
		var ep expiredPhoto
		if err := rows.Scan(&ep.id, &ep.key, &ep.timestamp); err == nil {
			expiredList = append(expiredList, ep)
		}
	}
	rows.Close()

	for _, ep := range expiredList {
		// Update DB first (Rule: avoid dangling references to deleted objects)
		const updateQ = `UPDATE attendances SET photo_key = NULL, photo_purged_at = now(), updated_at = now() WHERE id = $1`
		if _, err := j.db.Exec(ctx, updateQ, ep.id); err != nil {
			j.logger.Error("failed to mark attendance photo purged in DB", "attendance_id", ep.id, "error", err)
			continue
		}

		// Delete from storage (best-effort)
		if j.store != nil && ep.key != "" {
			if err := j.store.Delete(ctx, ep.key); err != nil {
				j.logger.Warn("failed to delete attendance photo from storage", "key", ep.key, "error", err)
			}
		}

		res.PhotosPurged++
		if res.OldestPhoto == nil || ep.timestamp.Before(*res.OldestPhoto) {
			t := ep.timestamp
			res.OldestPhoto = &t
		}
		if res.NewestPhoto == nil || ep.timestamp.After(*res.NewestPhoto) {
			t := ep.timestamp
			res.NewestPhoto = &t
		}
	}

	// 2. Purge expired telemetry attempts
	const purgeAttemptsQ = `
		DELETE FROM attendance_attempts
		WHERE server_timestamp < now() - make_interval(days => $1)
	`
	tag, err := j.db.Exec(ctx, purgeAttemptsQ, attemptDays)
	if err != nil {
		j.logger.Error("failed to purge attendance attempts", "error", err)
	} else {
		res.AttemptsPurged = tag.RowsAffected()
	}

	// 3. Record audit log if anything was purged
	if (res.PhotosPurged > 0 || res.AttemptsPurged > 0) && j.audit != nil {
		meta := map[string]any{
			"photos_purged":   res.PhotosPurged,
			"attempts_purged": res.AttemptsPurged,
			"photo_days":      photoDays,
			"attempt_days":    attemptDays,
		}
		if res.OldestPhoto != nil {
			meta["oldest_photo"] = res.OldestPhoto.Format(time.RFC3339)
		}
		if res.NewestPhoto != nil {
			meta["newest_photo"] = res.NewestPhoto.Format(time.RFC3339)
		}

		_ = j.audit.Record(ctx, audit.LogEntry{
			Action:       "attendance.photos_purged",
			ResourceType: "attendance",
			Metadata:     meta,
		})
	}

	j.logger.Info("attendance retention job completed",
		"photos_purged", res.PhotosPurged,
		"attempts_purged", res.AttemptsPurged,
	)

	return res, nil
}

// Start runs the retention worker in a background loop with a given ticker interval.
func (j *RetentionJob) Start(ctx context.Context, interval time.Duration) {
	if interval < time.Minute {
		interval = 24 * time.Hour
	}
	j.logger.Info("starting attendance retention worker", "interval", interval)

	go func() {
		// Run initial cycle shortly after startup (e.g. 10 seconds)
		select {
		case <-time.After(10 * time.Second):
			_, _ = j.RunOnce(ctx)
		case <-ctx.Done():
			return
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				j.logger.Info("stopping attendance retention worker: context canceled")
				return
			case <-ticker.C:
				_, _ = j.RunOnce(ctx)
			}
		}
	}()
}
