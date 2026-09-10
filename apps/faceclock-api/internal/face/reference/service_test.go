package reference

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
)

type mockRepo struct {
	refs          map[uuid.UUID]*ReferenceModel
	employeeExist map[uuid.UUID]bool
	summaries     map[uuid.UUID]*EmployeeFaceSummary
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		refs:          make(map[uuid.UUID]*ReferenceModel),
		employeeExist: make(map[uuid.UUID]bool),
		summaries:     make(map[uuid.UUID]*EmployeeFaceSummary),
	}
}

func (m *mockRepo) ListByEmployee(ctx context.Context, employeeID uuid.UUID, includeInactive bool) ([]ReferenceModel, error) {
	var res []ReferenceModel
	for _, r := range m.refs {
		if r.EmployeeID == employeeID {
			if includeInactive || r.IsActive {
				res = append(res, *r)
			}
		}
	}
	return res, nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReferenceModel, error) {
	r, ok := m.refs[id]
	if !ok {
		return nil, ErrReferenceNotFound
	}
	return r, nil
}

func (m *mockRepo) CountActive(ctx context.Context, employeeID uuid.UUID) (int, error) {
	cnt := 0
	for _, r := range m.refs {
		if r.EmployeeID == employeeID && r.IsActive {
			cnt++
		}
	}
	return cnt, nil
}

func (m *mockRepo) Deactivate(ctx context.Context, id uuid.UUID, reason string) (*ReferenceModel, error) {
	r, ok := m.refs[id]
	if !ok {
		return nil, ErrReferenceNotFound
	}
	r.IsActive = false
	now := time.Now()
	r.DeactivatedAt = &now
	r.DeactivatedReason = &reason
	return r, nil
}

func (m *mockRepo) DeleteAllForEmployee(ctx context.Context, employeeID uuid.UUID) (int, []string, error) {
	cnt := 0
	var keys []string
	for id, r := range m.refs {
		if r.EmployeeID == employeeID {
			cnt++
			keys = append(keys, r.PhotoKey)
			delete(m.refs, id)
		}
	}
	return cnt, keys, nil
}

func (m *mockRepo) GetEmployeeFaceSummary(ctx context.Context, employeeID uuid.UUID, currentDocVersion string) (*EmployeeFaceSummary, error) {
	s, ok := m.summaries[employeeID]
	if !ok {
		return nil, ErrEmployeeNotFound
	}
	return s, nil
}

func (m *mockRepo) EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	return m.employeeExist[employeeID], nil
}

type mockDocVerProvider struct {
	ver string
}

func (m *mockDocVerProvider) GetCurrentDocumentVersion(ctx context.Context) (string, error) {
	return m.ver, nil
}

func TestReferenceService_ListReferences(t *testing.T) {
	repo := newMockRepo()
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	logger := slog.Default()
	svc := NewService(repo, store, nil, &mockDocVerProvider{ver: "2026-09-v1"}, nil, logger)

	empID := uuid.New()
	callerID := uuid.New()
	repo.employeeExist[empID] = true

	refID := uuid.New()
	repo.refs[refID] = &ReferenceModel{
		ID:            refID,
		EmployeeID:    empID,
		Position:      1,
		QualityScore:  0.88,
		DetScore:      0.95,
		ModelVersion:  "buffalo_l",
		PhotoKey:      "face/123/1.jpg",
		PhotoMIME:     "image/jpeg",
		CaptureSource: "web_camera",
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	// 1. Permission check: another employee without PermFaceReadAny -> 404
	_, err = svc.ListReferences(context.Background(), empID, false, callerID, uuid.New(), false)
	if err == nil {
		t.Fatal("expected error for unauthorized access to other employee references")
	}

	// 2. Own employee -> success
	resp, err := svc.ListReferences(context.Background(), empID, false, callerID, empID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(resp.Data))
	}
	expectedURL := "/api/v1/face/references/" + refID.String() + "/photo"
	if resp.Data[0].PhotoURL != expectedURL {
		t.Errorf("expected photo_url %s, got %s", expectedURL, resp.Data[0].PhotoURL)
	}
}

func TestReferenceService_GetPhoto(t *testing.T) {
	repo := newMockRepo()
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	logger := slog.Default()
	svc := NewService(repo, store, nil, &mockDocVerProvider{ver: "2026-09-v1"}, nil, logger)

	empID := uuid.New()
	refID := uuid.New()
	photoKey := "face/" + empID.String() + "/" + refID.String() + ".jpg"

	_, _ = store.Put(context.Background(), photoKey, bytes.NewReader([]byte("fake_image_bytes")), "image/jpeg")

	repo.refs[refID] = &ReferenceModel{
		ID:         refID,
		EmployeeID: empID,
		PhotoKey:   photoKey,
		PhotoMIME:  "image/jpeg",
		IsActive:   true,
	}

	// 1. Successful read by owner
	r, mime, err := svc.GetPhoto(context.Background(), refID, uuid.New(), empID, false)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	content, _ := io.ReadAll(r)
	r.Close()
	if string(content) != "fake_image_bytes" || mime != "image/jpeg" {
		t.Errorf("unexpected content or mime: %s, %s", string(content), mime)
	}

	// 2. Photo purged retention -> ErrPhotoPurged
	purgedAt := time.Now()
	repo.refs[refID].PhotoPurgedAt = &purgedAt
	_, _, err = svc.GetPhoto(context.Background(), refID, uuid.New(), empID, false)
	if err == nil {
		t.Fatal("expected error for purged photo")
	}
	if !errors.Is(err, ErrPhotoPurged) {
		t.Errorf("expected ErrPhotoPurged, got %v", err)
	}
}

func TestReferenceService_DeactivateReference(t *testing.T) {
	repo := newMockRepo()
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	logger := slog.Default()
	svc := NewService(repo, store, nil, &mockDocVerProvider{ver: "2026-09-v1"}, nil, logger)

	empID := uuid.New()
	ref1 := uuid.New()
	ref2 := uuid.New()
	ref3 := uuid.New()

	for _, id := range []uuid.UUID{ref1, ref2, ref3} {
		repo.refs[id] = &ReferenceModel{
			ID:         id,
			EmployeeID: empID,
			IsActive:   true,
		}
	}

	// Attempting to reactivate should fail
	_, err = svc.DeactivateReference(context.Background(), ref1, DeactivateReferenceRequest{IsActive: true}, uuid.New())
	if err == nil {
		t.Fatal("expected validation error when setting is_active=true")
	}

	// Deactivating when total is 3 (remaining would be 2 < 3) without allow_below_minimum
	reason := "poor angle"
	_, err = svc.DeactivateReference(context.Background(), ref1, DeactivateReferenceRequest{IsActive: false, AllowBelowMinimum: false, Reason: &reason}, uuid.New())
	if err == nil {
		t.Fatal("expected 409 conflict when dropping below minimum photos without flag")
	}

	// With allow_below_minimum = true
	res, err := svc.DeactivateReference(context.Background(), ref1, DeactivateReferenceRequest{IsActive: false, AllowBelowMinimum: true, Reason: &reason}, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsActive {
		t.Error("expected reference to be inactive")
	}
	if res.RemainingActiveCount != 2 {
		t.Errorf("expected 2 remaining active, got %d", res.RemainingActiveCount)
	}
}

func TestReferenceService_DeleteFaceData(t *testing.T) {
	repo := newMockRepo()
	store, err := storage.NewLocalStore(t.TempDir())
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	logger := slog.Default()
	svc := NewService(repo, store, nil, &mockDocVerProvider{ver: "2026-09-v1"}, nil, logger)

	empID := uuid.New()
	repo.employeeExist[empID] = true
	refID := uuid.New()
	photoKey := "face/" + empID.String() + "/ref.jpg"
	_, _ = store.Put(context.Background(), photoKey, bytes.NewReader([]byte("test")), "image/jpeg")

	repo.refs[refID] = &ReferenceModel{
		ID:         refID,
		EmployeeID: empID,
		PhotoKey:   photoKey,
		IsActive:   true,
	}

	// Missing reason
	_, err = svc.DeleteFaceData(context.Background(), empID, "", uuid.New())
	if err == nil {
		t.Fatal("expected validation error when reason is empty")
	}

	// Valid erasure
	res, err := svc.DeleteFaceData(context.Background(), empID, "employee requested biometric deletion under UU PDP", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.DeletedReferenceCount != 1 || res.PhotosDeleted != 1 {
		t.Errorf("expected 1 ref and 1 photo deleted, got %d, %d", res.DeletedReferenceCount, res.PhotosDeleted)
	}

	// Storage key should be deleted
	_, err = store.Get(context.Background(), photoKey)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Error("expected photo to be deleted from store")
	}
}
