package reindex

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	hasRunning    bool
	candidates    int
	affected      int
	jobs          map[uuid.UUID]*JobModel
	items         map[uuid.UUID][]ItemCandidate
	locked        bool
	nextJob       *JobModel
	succeededRefs []ReferenceInsert
	failedItems   map[uuid.UUID]string
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		jobs:        make(map[uuid.UUID]*JobModel),
		items:       make(map[uuid.UUID][]ItemCandidate),
		failedItems: make(map[uuid.UUID]string),
	}
}

func (m *mockRepo) HasRunningJob(ctx context.Context) (bool, error) {
	return m.hasRunning, nil
}

func (m *mockRepo) CountCandidates(ctx context.Context, fromModel string) (int, int, error) {
	return m.candidates, m.affected, nil
}

func (m *mockRepo) CreateJob(ctx context.Context, fromModel, toModel string, createdBy *uuid.UUID) (*JobModel, int, error) {
	if m.hasRunning {
		return nil, 0, ErrReindexAlreadyActive
	}
	id := uuid.New()
	now := time.Now()
	j := &JobModel{
		ID:               id,
		FromModelVersion: fromModel,
		ToModelVersion:   toModel,
		Status:           "pending",
		TotalCount:       m.candidates,
		CreatedBy:        createdBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	m.jobs[id] = j
	return j, m.affected, nil
}

func (m *mockRepo) GetJob(ctx context.Context, id uuid.UUID, minRequired int) (*JobModel, []IncompleteEmployee, error) {
	j, ok := m.jobs[id]
	if !ok {
		return nil, nil, ErrJobNotFound
	}
	return j, []IncompleteEmployee{}, nil
}

func (m *mockRepo) ListJobs(ctx context.Context) ([]JobModel, error) {
	var list []JobModel
	for _, j := range m.jobs {
		list = append(list, *j)
	}
	return list, nil
}

func (m *mockRepo) CancelJob(ctx context.Context, id uuid.UUID) (*JobModel, error) {
	j, ok := m.jobs[id]
	if !ok || (j.Status != "pending" && j.Status != "running") {
		return nil, ErrJobCannotCancel
	}
	j.Status = "cancelled"
	return j, nil
}

func (m *mockRepo) TryAcquireLock(ctx context.Context) (bool, error) {
	if m.locked {
		return false, nil
	}
	m.locked = true
	return true, nil
}

func (m *mockRepo) ReleaseLock(ctx context.Context) error {
	m.locked = false
	return nil
}

func (m *mockRepo) GetNextPendingOrRunningJob(ctx context.Context) (*JobModel, error) {
	return m.nextJob, nil
}

func (m *mockRepo) MarkJobRunning(ctx context.Context, id uuid.UUID) error {
	if j, ok := m.jobs[id]; ok {
		j.Status = "running"
	}
	return nil
}

func (m *mockRepo) FetchPendingItems(ctx context.Context, jobID uuid.UUID, limit int) ([]ItemCandidate, error) {
	list := m.items[jobID]
	if len(list) == 0 {
		return nil, nil
	}
	if len(list) > limit {
		res := list[:limit]
		m.items[jobID] = list[limit:]
		return res, nil
	}
	m.items[jobID] = nil
	return list, nil
}

func (m *mockRepo) SaveItemSuccess(ctx context.Context, jobID, refID uuid.UUID, newRef ReferenceInsert) (uuid.UUID, error) {
	m.succeededRefs = append(m.succeededRefs, newRef)
	return uuid.New(), nil
}

func (m *mockRepo) SaveItemFailure(ctx context.Context, jobID, refID uuid.UUID, reason string, hints []string) error {
	m.failedItems[refID] = reason
	return nil
}

func (m *mockRepo) IncrementJobCounters(ctx context.Context, jobID uuid.UUID, succeededDelta, failedDelta int) error {
	if j, ok := m.jobs[jobID]; ok {
		j.ProcessedCount += succeededDelta + failedDelta
		j.SucceededCount += succeededDelta
		j.FailedCount += failedDelta
	}
	return nil
}

func (m *mockRepo) FinalizeEmployeeSwaps(ctx context.Context, jobID uuid.UUID, minRequired int) (int, int, error) {
	if j, ok := m.jobs[jobID]; ok {
		j.Status = "completed"
		j.EmployeesReadyCount = 1
		j.EmployeesIncompleteCount = 0
	}
	return 1, 0, nil
}

func (m *mockRepo) MarkJobFailed(ctx context.Context, jobID uuid.UUID, errMsg string) error {
	if j, ok := m.jobs[jobID]; ok {
		j.Status = "failed"
		j.Error = &errMsg
	}
	return nil
}

type mockEngine struct {
	status       string
	modelVersion string
}

func (m *mockEngine) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*inference.EmbedResult, error) {
	return nil, nil
}

func (m *mockEngine) EmbedBatch(ctx context.Context, images [][]byte) (*inference.BatchEmbedResult, error) {
	var items []inference.BatchEmbedItem
	for idx := range images {
		items = append(items, inference.BatchEmbedItem{
			Index: idx,
			Result: &inference.EmbedResult{
				Embedding:    make([]float32, 512),
				QualityScore: 0.95,
				DetScore:     0.98,
				Usable:       true,
			},
		})
	}
	return &inference.BatchEmbedResult{
		Items:        items,
		ModelVersion: m.modelVersion,
	}, nil
}

func (m *mockEngine) Health(ctx context.Context) (*inference.HealthStatus, error) {
	return &inference.HealthStatus{
		Status:       m.status,
		ModelVersion: m.modelVersion,
	}, nil
}

func (m *mockEngine) Ready(ctx context.Context) (*inference.ReadyData, error) {
	return &inference.ReadyData{
		Status:       m.status,
		ModelVersion: m.modelVersion,
	}, nil
}

type mockStore struct {
	files map[string][]byte
}

func (m *mockStore) Put(ctx context.Context, key string, r io.Reader, contentType string) (string, error) {
	b, _ := io.ReadAll(r)
	m.files[key] = b
	return key, nil
}

func (m *mockStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	b, ok := m.files[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(strings.NewReader(string(b))), nil
}

func (m *mockStore) Delete(ctx context.Context, key string) error {
	delete(m.files, key)
	return nil
}

func (m *mockStore) Copy(ctx context.Context, srcKey, dstKey string) error {
	b, ok := m.files[srcKey]
	if !ok {
		return storage.ErrNotFound
	}
	m.files[dstKey] = b
	return nil
}

func (m *mockStore) List(ctx context.Context, prefix string) ([]string, error) {
	var res []string
	for k := range m.files {
		if strings.HasPrefix(k, prefix) {
			res = append(res, k)
		}
	}
	return res, nil
}

func (m *mockStore) SignedURL(ctx context.Context, key string, ttlSeconds int) (string, error) {
	return "http://mock/" + key, nil
}

func TestReindexService_CreateJob_Validation(t *testing.T) {
	repo := newMockRepo()
	eng := &mockEngine{status: "ok", modelVersion: "buffalo_m"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, eng, nil, nil, logger)

	callerID := uuid.New()

	// Empty target model
	_, err := svc.CreateJob(context.Background(), CreateJobRequest{ToModelVersion: ""}, callerID)
	assert.Error(t, err)
	var appErr *httpx.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, httpx.CodeValidationError, appErr.Code)

	// Target model matches current active model ("buffalo_l" by default)
	_, err = svc.CreateJob(context.Background(), CreateJobRequest{ToModelVersion: "buffalo_l"}, callerID)
	assert.Error(t, err)

	// Engine model mismatch
	eng.modelVersion = "other_model"
	_, err = svc.CreateJob(context.Background(), CreateJobRequest{ToModelVersion: "buffalo_m"}, callerID)
	assert.Error(t, err)
}

func TestReindexService_CreateJob_DryRun(t *testing.T) {
	repo := newMockRepo()
	repo.candidates = 15
	repo.affected = 5

	eng := &mockEngine{status: "ok", modelVersion: "buffalo_m"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, eng, nil, nil, logger)

	callerID := uuid.New()
	resp, err := svc.CreateJob(context.Background(), CreateJobRequest{
		ToModelVersion: "buffalo_m",
		DryRun:         true,
	}, callerID)

	require.NoError(t, err)
	assert.True(t, resp.DryRun)
	assert.Equal(t, "dry_run", resp.Status)
	assert.Equal(t, 15, resp.TotalCount)
	assert.Equal(t, 5, resp.AffectedEmployeeCount)
	assert.Nil(t, resp.ID)
}

func TestReindexService_CreateAndGetJob(t *testing.T) {
	repo := newMockRepo()
	repo.candidates = 6
	repo.affected = 2

	eng := &mockEngine{status: "ok", modelVersion: "buffalo_m"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewService(repo, eng, nil, nil, logger)

	callerID := uuid.New()
	resp, err := svc.CreateJob(context.Background(), CreateJobRequest{
		ToModelVersion: "buffalo_m",
		DryRun:         false,
	}, callerID)

	require.NoError(t, err)
	assert.False(t, resp.DryRun)
	assert.Equal(t, "pending", resp.Status)
	require.NotNil(t, resp.ID)

	// Get Job
	detail, err := svc.GetJob(context.Background(), *resp.ID)
	require.NoError(t, err)
	assert.Equal(t, *resp.ID, detail.ID)
	assert.Equal(t, "pending", detail.Status)
	assert.Equal(t, 6, detail.TotalCount)

	// Cancel Job
	cancelled, err := svc.CancelJob(context.Background(), *resp.ID, callerID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", cancelled.Status)
}

func TestReindexWorker_RunOnce(t *testing.T) {
	repo := newMockRepo()
	eng := &mockEngine{status: "ok", modelVersion: "buffalo_m"}
	store := &mockStore{
		files: map[string][]byte{
			"face/emp1/photo1.jpg": []byte("dummy-photo-1"),
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	worker := NewWorker(repo, store, eng, nil, nil, logger, time.Second)

	jobID := uuid.New()
	empID := uuid.New()
	job := &JobModel{
		ID:               jobID,
		FromModelVersion: "buffalo_l",
		ToModelVersion:   "buffalo_m",
		Status:           "pending",
		TotalCount:       1,
	}
	repo.jobs[jobID] = job
	repo.nextJob = job
	repo.items[jobID] = []ItemCandidate{
		{
			JobID:           jobID,
			FaceReferenceID: uuid.New(),
			EmployeeID:      empID,
			PhotoKey:        "face/emp1/photo1.jpg",
			Position:        1,
			CaptureSource:   "webcam",
		},
	}

	err := worker.RunOnce(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "completed", job.Status)
	assert.Equal(t, 1, len(repo.succeededRefs))
	assert.Equal(t, 1, job.SucceededCount)
}
