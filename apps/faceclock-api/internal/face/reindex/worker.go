package reindex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
)

const BatchSize = 16

type Worker struct {
	repo     Repository
	store    storage.Store
	engine   inference.FaceEngine
	settings *settings.Service
	audit    *audit.Recorder
	logger   *slog.Logger
	interval time.Duration
}

func NewWorker(
	repo Repository,
	store storage.Store,
	engine inference.FaceEngine,
	settings *settings.Service,
	audit *audit.Recorder,
	logger *slog.Logger,
	interval time.Duration,
) *Worker {
	if interval <= 0 {
		interval = 10 * time.Second
	}
	return &Worker{
		repo:     repo,
		store:    store,
		engine:   engine,
		settings: settings,
		audit:    audit,
		logger:   logger,
		interval: interval,
	}
}

// Start runs the periodic worker loop in background until ctx is cancelled.
func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.logger.Info("face reindex worker started", "interval", w.interval.String())

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("face reindex worker stopping")
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil {
				w.logger.Error("face reindex worker run failed", "error", err)
			}
		}
	}
}

// RunOnce performs a single pass over pending/running reindex jobs.
func (w *Worker) RunOnce(ctx context.Context) error {
	locked, err := w.repo.TryAcquireLock(ctx)
	if err != nil {
		return fmt.Errorf("try acquire advisory lock: %w", err)
	}
	if !locked {
		// Another worker/replica holds the lock
		return nil
	}
	defer func() {
		_ = w.repo.ReleaseLock(context.Background())
	}()

	job, err := w.repo.GetNextPendingOrRunningJob(ctx)
	if err != nil {
		return fmt.Errorf("get pending job: %w", err)
	}
	if job == nil {
		return nil
	}

	w.logger.Info("processing face reindex job",
		"job_id", job.ID,
		"from_model", job.FromModelVersion,
		"to_model", job.ToModelVersion,
		"status", job.Status,
	)

	if job.Status == "pending" {
		if err := w.repo.MarkJobRunning(ctx, job.ID); err != nil {
			return fmt.Errorf("mark job running: %w", err)
		}
		job.Status = "running"
	}

	minRequired := w.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if minRequired <= 0 {
		minRequired = 3
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check cancellation
		latestJob, _, err := w.repo.GetJob(ctx, job.ID, minRequired)
		if err != nil {
			return fmt.Errorf("check job status: %w", err)
		}
		if latestJob.Status == "cancelled" {
			w.logger.Info("face reindex job was cancelled", "job_id", job.ID)
			return nil
		}

		candidates, err := w.repo.FetchPendingItems(ctx, job.ID, BatchSize)
		if err != nil {
			return fmt.Errorf("fetch pending items: %w", err)
		}
		if len(candidates) == 0 {
			// All items processed!
			break
		}

		if err := w.processBatch(ctx, job, candidates); err != nil {
			w.logger.Error("error processing batch for job", "job_id", job.ID, "error", err)
			// continue next iterations or stop? Let's log and retry
		}
	}

	// Finalize employee swaps
	readyCount, incompleteCount, err := w.repo.FinalizeEmployeeSwaps(ctx, job.ID, minRequired)
	if err != nil {
		_ = w.repo.MarkJobFailed(ctx, job.ID, err.Error())
		return fmt.Errorf("finalize swaps: %w", err)
	}

	// Update system settings face.model_version to new model version
	if w.settings != nil {
		_, _, _ = w.settings.UpdateSingle(ctx, job.CreatedBy, "face.model_version", job.ToModelVersion)
	}

	w.logger.Info("face reindex job completed",
		"job_id", job.ID,
		"employees_ready", readyCount,
		"employees_incomplete", incompleteCount,
	)

	if w.audit != nil {
		jobIDStr := job.ID.String()
		_ = w.audit.Record(ctx, audit.LogEntry{
			Action:       "face.reindex.completed",
			ActorUserID:  job.CreatedBy,
			ResourceType: "face_reindex_job",
			ResourceID:   &jobIDStr,
			Metadata: map[string]any{
				"job_id":                     jobIDStr,
				"from_model_version":         job.FromModelVersion,
				"to_model_version":           job.ToModelVersion,
				"employees_ready_count":      readyCount,
				"employees_incomplete_count": incompleteCount,
			},
		})
	}

	return nil
}

func (w *Worker) processBatch(ctx context.Context, job *JobModel, candidates []ItemCandidate) error {
	var validImages [][]byte
	var validCandidates []ItemCandidate

	succeededDelta := 0
	failedDelta := 0

	// 1. Fetch images from storage
	for _, c := range candidates {
		reader, err := w.store.Get(ctx, c.PhotoKey)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				_ = w.repo.SaveItemFailure(ctx, job.ID, c.FaceReferenceID, "photo_missing", []string{"photo_missing"})
				failedDelta++
				continue
			}
			// Transient read error
			return fmt.Errorf("read photo %s: %w", c.PhotoKey, err)
		}

		imgBytes, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			_ = w.repo.SaveItemFailure(ctx, job.ID, c.FaceReferenceID, "photo_read_failed", []string{"read_error"})
			failedDelta++
			continue
		}

		validImages = append(validImages, imgBytes)
		validCandidates = append(validCandidates, c)
	}

	if len(validImages) > 0 {
		batchRes, err := w.engine.EmbedBatch(ctx, validImages)
		if err != nil {
			return fmt.Errorf("embed batch from engine: %w", err)
		}

		for idx, item := range batchRes.Items {
			cand := validCandidates[idx]
			if item.Error != nil || item.Result == nil || !item.Result.Usable {
				hints := []string{}
				reason := "quality_insufficient"
				if item.Result != nil && len(item.Result.Hints) > 0 {
					hints = item.Result.Hints
				}
				if item.Error != nil {
					reason = item.Error.Code
				}
				_ = w.repo.SaveItemFailure(ctx, job.ID, cand.FaceReferenceID, reason, hints)
				failedDelta++
			} else {
				// Insert new reference
				_, err := w.repo.SaveItemSuccess(ctx, job.ID, cand.FaceReferenceID, ReferenceInsert{
					EmployeeID:    cand.EmployeeID,
					Position:      cand.Position,
					Embedding:     item.Result.Embedding,
					QualityScore:  item.Result.QualityScore,
					DetScore:      item.Result.DetScore,
					ModelVersion:  job.ToModelVersion,
					PhotoKey:      cand.PhotoKey,
					PhotoSHA256:   cand.PhotoSHA256,
					PhotoBytes:    cand.PhotoBytes,
					PhotoMIME:     cand.PhotoMIME,
					CaptureSource: cand.CaptureSource,
					EnrolledBy:    cand.EnrolledBy,
				})
				if err != nil {
					w.logger.Error("failed to save item success", "ref_id", cand.FaceReferenceID, "error", err)
					_ = w.repo.SaveItemFailure(ctx, job.ID, cand.FaceReferenceID, "insert_failed", []string{"db_error"})
					failedDelta++
				} else {
					succeededDelta++
				}
			}
		}
	}

	return w.repo.IncrementJobCounters(ctx, job.ID, succeededDelta, failedDelta)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
