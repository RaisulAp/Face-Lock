package reindex

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/google/uuid"
)

type Service struct {
	repo     Repository
	engine   inference.FaceEngine
	settings *settings.Service
	audit    *audit.Recorder
	logger   *slog.Logger
}

func NewService(
	repo Repository,
	engine inference.FaceEngine,
	settings *settings.Service,
	audit *audit.Recorder,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:     repo,
		engine:   engine,
		settings: settings,
		audit:    audit,
		logger:   logger,
	}
}

func (s *Service) CreateJob(ctx context.Context, req CreateJobRequest, callerID uuid.UUID) (*CreateJobResponse, error) {
	req.ToModelVersion = strings.TrimSpace(req.ToModelVersion)
	if req.ToModelVersion == "" {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "Versi model tujuan (to_model_version) wajib diisi")
	}

	currentModel := s.settings.GetString(ctx, "face.model_version", "buffalo_l")
	if req.ToModelVersion == currentModel {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "Versi model tujuan tidak boleh sama dengan versi model aktif saat ini")
	}

	// Verify against inference health
	health, err := s.engine.Health(ctx)
	if err != nil || health == nil || health.Status != "ok" {
		return nil, httpx.NewAppError(httpx.CodeServiceUnavailable, "Layanan inferensi wajah tidak tersedia")
	}

	if health.ModelVersion != "" && health.ModelVersion != req.ToModelVersion {
		return nil, httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("Versi model tujuan tidak cocok dengan model yang sedang aktif di layanan inferensi (inferensi: %s, diminta: %s)", health.ModelVersion, req.ToModelVersion))
	}

	// Check if already running
	hasRunning, err := s.repo.HasRunningJob(ctx)
	if err != nil {
		return nil, fmt.Errorf("check running job: %w", err)
	}
	if hasRunning {
		return nil, httpx.NewAppError(httpx.CodeReindexInProgress, "Pekerjaan reindex wajah sedang berlangsung")
	}

	if req.DryRun {
		total, affected, err := s.repo.CountCandidates(ctx, currentModel)
		if err != nil {
			return nil, fmt.Errorf("count candidates: %w", err)
		}
		return &CreateJobResponse{
			FromModelVersion:      currentModel,
			ToModelVersion:        req.ToModelVersion,
			Status:                "dry_run",
			TotalCount:            total,
			AffectedEmployeeCount: affected,
			DryRun:                true,
		}, nil
	}

	job, affected, err := s.repo.CreateJob(ctx, currentModel, req.ToModelVersion, &callerID)
	if err != nil {
		if errors.Is(err, ErrReindexAlreadyActive) {
			return nil, httpx.NewAppError(httpx.CodeReindexInProgress, "Pekerjaan reindex wajah sedang berlangsung")
		}
		return nil, fmt.Errorf("create job: %w", err)
	}

	if s.audit != nil {
		jobIDStr := job.ID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.reindex.started",
			ActorUserID:  &callerID,
			ResourceType: "face_reindex_job",
			ResourceID:   &jobIDStr,
			Metadata: map[string]any{
				"job_id":             jobIDStr,
				"from_model_version": currentModel,
				"to_model_version":   req.ToModelVersion,
				"total_count":        job.TotalCount,
			},
		})
	}

	return &CreateJobResponse{
		ID:                    &job.ID,
		FromModelVersion:      job.FromModelVersion,
		ToModelVersion:        job.ToModelVersion,
		Status:                job.Status,
		TotalCount:            job.TotalCount,
		AffectedEmployeeCount: affected,
	}, nil
}

func (s *Service) GetJob(ctx context.Context, id uuid.UUID) (*JobDetailResponse, error) {
	minRequired := s.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if minRequired <= 0 {
		minRequired = 3
	}

	job, incomplete, err := s.repo.GetJob(ctx, id, minRequired)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Pekerjaan reindex tidak ditemukan")
		}
		return nil, fmt.Errorf("get job: %w", err)
	}

	if incomplete == nil {
		incomplete = []IncompleteEmployee{}
	}

	return &JobDetailResponse{
		ID:                       job.ID,
		FromModelVersion:         job.FromModelVersion,
		ToModelVersion:           job.ToModelVersion,
		Status:                   job.Status,
		TotalCount:               job.TotalCount,
		ProcessedCount:           job.ProcessedCount,
		SucceededCount:           job.SucceededCount,
		FailedCount:              job.FailedCount,
		EmployeesReadyCount:      job.EmployeesReadyCount,
		EmployeesIncompleteCount: job.EmployeesIncompleteCount,
		Error:                    job.Error,
		StartedAt:                job.StartedAt,
		FinishedAt:               job.FinishedAt,
		CreatedAt:                job.CreatedAt,
		IncompleteEmployees:      incomplete,
	}, nil
}

func (s *Service) ListJobs(ctx context.Context) ([]JobDetailResponse, error) {
	jobs, err := s.repo.ListJobs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}

	res := make([]JobDetailResponse, 0, len(jobs))
	for _, j := range jobs {
		res = append(res, JobDetailResponse{
			ID:                       j.ID,
			FromModelVersion:         j.FromModelVersion,
			ToModelVersion:           j.ToModelVersion,
			Status:                   j.Status,
			TotalCount:               j.TotalCount,
			ProcessedCount:           j.ProcessedCount,
			SucceededCount:           j.SucceededCount,
			FailedCount:              j.FailedCount,
			EmployeesReadyCount:      j.EmployeesReadyCount,
			EmployeesIncompleteCount: j.EmployeesIncompleteCount,
			Error:                    j.Error,
			StartedAt:                j.StartedAt,
			FinishedAt:               j.FinishedAt,
			CreatedAt:                j.CreatedAt,
			IncompleteEmployees:      []IncompleteEmployee{},
		})
	}
	return res, nil
}

func (s *Service) CancelJob(ctx context.Context, id uuid.UUID, callerID uuid.UUID) (*JobDetailResponse, error) {
	job, err := s.repo.CancelJob(ctx, id)
	if err != nil {
		if errors.Is(err, ErrJobCannotCancel) {
			return nil, httpx.NewAppError(httpx.CodeConflict, "Pekerjaan reindex tidak dapat dibatalkan (sudah selesai atau tidak ditemukan)")
		}
		return nil, fmt.Errorf("cancel job: %w", err)
	}

	if s.audit != nil {
		jobIDStr := job.ID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.reindex.cancelled",
			ActorUserID:  &callerID,
			ResourceType: "face_reindex_job",
			ResourceID:   &jobIDStr,
			Metadata: map[string]any{
				"job_id": jobIDStr,
			},
		})
	}

	return &JobDetailResponse{
		ID:                       job.ID,
		FromModelVersion:         job.FromModelVersion,
		ToModelVersion:           job.ToModelVersion,
		Status:                   job.Status,
		TotalCount:               job.TotalCount,
		ProcessedCount:           job.ProcessedCount,
		SucceededCount:           job.SucceededCount,
		FailedCount:              job.FailedCount,
		EmployeesReadyCount:      job.EmployeesReadyCount,
		EmployeesIncompleteCount: job.EmployeesIncompleteCount,
		Error:                    job.Error,
		StartedAt:                job.StartedAt,
		FinishedAt:               job.FinishedAt,
		CreatedAt:                job.CreatedAt,
		IncompleteEmployees:      []IncompleteEmployee{},
	}, nil
}
