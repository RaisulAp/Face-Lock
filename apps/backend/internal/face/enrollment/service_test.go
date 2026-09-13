package enrollment_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face/enrollment"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
)

type mockRepo struct {
	sessions    map[uuid.UUID]*enrollment.Session
	photos      map[uuid.UUID][]enrollment.Photo
	reindexing  bool
	attMode     string
	activeDraft *enrollment.Session
	duplicates  []enrollment.DuplicateMatch
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		sessions: make(map[uuid.UUID]*enrollment.Session),
		photos:   make(map[uuid.UUID][]enrollment.Photo),
		attMode:  "face",
	}
}

func (m *mockRepo) CreateSession(ctx context.Context, s *enrollment.Session) error {
	m.sessions[s.ID] = s
	return nil
}

func (m *mockRepo) GetSessionByID(ctx context.Context, id uuid.UUID) (*enrollment.Session, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, enrollment.ErrSessionNotFound
	}
	return s, nil
}

func (m *mockRepo) GetActiveDraftSession(ctx context.Context, employeeID uuid.UUID) (*enrollment.Session, error) {
	return m.activeDraft, nil
}

func (m *mockRepo) ExpireSession(ctx context.Context, id uuid.UUID) error {
	if s, ok := m.sessions[id]; ok {
		s.Status = "expired"
	}
	return nil
}

func (m *mockRepo) CancelSession(ctx context.Context, id uuid.UUID) error {
	if s, ok := m.sessions[id]; ok {
		s.Status = "cancelled"
	}
	return nil
}

func (m *mockRepo) CountActiveReferences(ctx context.Context, employeeID uuid.UUID) (int, error) {
	return 3, nil
}

func (m *mockRepo) IsReindexRunning(ctx context.Context) (bool, error) {
	return m.reindexing, nil
}

func (m *mockRepo) GetEmployeeAttendanceMode(ctx context.Context, employeeID uuid.UUID) (string, error) {
	return m.attMode, nil
}

func (m *mockRepo) EmployeeExists(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	return true, nil
}

func (m *mockRepo) InsertPhoto(ctx context.Context, p *enrollment.Photo) error {
	m.photos[p.SessionID] = append(m.photos[p.SessionID], *p)
	return nil
}

func (m *mockRepo) ListPhotosBySession(ctx context.Context, sessionID uuid.UUID) ([]enrollment.Photo, error) {
	return m.photos[sessionID], nil
}

func (m *mockRepo) GetPhotoByID(ctx context.Context, sessionID, photoID uuid.UUID) (*enrollment.Photo, error) {
	for _, p := range m.photos[sessionID] {
		if p.ID == photoID {
			return &p, nil
		}
	}
	return nil, enrollment.ErrPhotoNotFound
}

func (m *mockRepo) DeletePhoto(ctx context.Context, sessionID, photoID uuid.UUID) error {
	photos := m.photos[sessionID]
	var updated []enrollment.Photo
	for _, p := range photos {
		if p.ID != photoID {
			updated = append(updated, p)
		}
	}
	m.photos[sessionID] = updated
	return nil
}

func (m *mockRepo) PhotoHashExists(ctx context.Context, sessionID uuid.UUID, hash string) (bool, error) {
	for _, p := range m.photos[sessionID] {
		if p.PhotoSHA256 == hash {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepo) NextPosition(ctx context.Context, sessionID uuid.UUID) (int, error) {
	return len(m.photos[sessionID]) + 1, nil
}

func (m *mockRepo) CheckDuplicateFaces(ctx context.Context, embedding []float32, modelVersion string, excludeEmployeeID uuid.UUID, threshold float32) ([]enrollment.DuplicateMatch, error) {
	return m.duplicates, nil
}

func (m *mockRepo) CommitSession(ctx context.Context, session *enrollment.Session, photos []enrollment.Photo, callerID uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	session.Status = "committed"
	var ids []uuid.UUID
	for range photos {
		ids = append(ids, uuid.New())
	}
	return ids, nil, nil
}

type mockConsent struct {
	hasConsent bool
}

func (m *mockConsent) HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	return m.hasConsent, nil
}

func TestCreateSession_Validation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	store, _ := storage.NewLocalStore(t.TempDir())
	engine := inference.NewFakeClient()
	consent := &mockConsent{hasConsent: true}

	svc := enrollment.NewService(repo, store, engine, nil, consent, nil, nil)

	callerID := uuid.New()
	employeeID := uuid.New()

	t.Run("success create session", func(t *testing.T) {
		resp, err := svc.CreateSession(ctx, enrollment.CreateSessionRequest{
			EmployeeID: &employeeID,
			Mode:       "replace",
		}, callerID, callerID, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil || resp.ID == uuid.Nil {
			t.Fatalf("expected valid session response")
		}
		if resp.Status != "draft" {
			t.Errorf("expected status 'draft', got '%s'", resp.Status)
		}
		if resp.RequiredPhotos != 3 {
			t.Errorf("expected 3 required photos, got %d", resp.RequiredPhotos)
		}
	})

	t.Run("fails when employee attendance_mode is manual", func(t *testing.T) {
		repo.attMode = "manual"
		defer func() { repo.attMode = "face" }()

		_, err := svc.CreateSession(ctx, enrollment.CreateSessionRequest{
			EmployeeID: &employeeID,
		}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeAttendanceModeManual {
				t.Errorf("expected code %s, got %s", httpx.CodeAttendanceModeManual, appErr.Code)
			}
		}
	})

	t.Run("fails when biometric consent is missing", func(t *testing.T) {
		consent.hasConsent = false
		defer func() { consent.hasConsent = true }()

		_, err := svc.CreateSession(ctx, enrollment.CreateSessionRequest{
			EmployeeID: &employeeID,
		}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeConsentRequired {
				t.Errorf("expected code %s, got %s", httpx.CodeConsentRequired, appErr.Code)
			}
		}
	})

	t.Run("fails when reindex in progress", func(t *testing.T) {
		repo.reindexing = true
		defer func() { repo.reindexing = false }()

		_, err := svc.CreateSession(ctx, enrollment.CreateSessionRequest{
			EmployeeID: &employeeID,
		}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeReindexInProgress {
				t.Errorf("expected code %s, got %s", httpx.CodeReindexInProgress, appErr.Code)
			}
		}
	})

	t.Run("fails when active draft already exists", func(t *testing.T) {
		repo.activeDraft = &enrollment.Session{
			ID:         uuid.New(),
			EmployeeID: employeeID,
			Status:     "draft",
		}
		defer func() { repo.activeDraft = nil }()

		_, err := svc.CreateSession(ctx, enrollment.CreateSessionRequest{
			EmployeeID: &employeeID,
		}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeConflict {
				t.Errorf("expected code %s, got %s", httpx.CodeConflict, appErr.Code)
			}
		}
	})
}

func TestUploadPhoto_QualityAndLimits(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	store, _ := storage.NewLocalStore(t.TempDir())
	engine := inference.NewFakeClient()
	consent := &mockConsent{hasConsent: true}

	svc := enrollment.NewService(repo, store, engine, nil, consent, nil, nil)

	callerID := uuid.New()
	employeeID := uuid.New()
	sessID := uuid.New()

	session := &enrollment.Session{
		ID:             sessID,
		EmployeeID:     employeeID,
		CreatedBy:      callerID,
		Status:         "draft",
		Mode:           "replace",
		ModelVersion:   "buffalo_l",
		RequiredPhotos: 3,
		MaxPhotos:      5,
		ExpiresAt:      time.Now().Add(15 * time.Minute),
		CreatedAt:      time.Now(),
	}
	repo.sessions[sessID] = session

	dummyImage := []byte("fake-jpeg-binary-data-for-testing")

	t.Run("successful upload", func(t *testing.T) {
		resp, err := svc.UploadPhoto(ctx, sessID, dummyImage, "image/jpeg", "web_camera", callerID, callerID, true)
		if err != nil {
			t.Fatalf("unexpected upload error: %v", err)
		}
		if resp == nil || resp.ID == uuid.Nil {
			t.Fatalf("expected valid photo response")
		}
		if resp.QualityScore < 0.8 {
			t.Errorf("expected good quality score, got %f", resp.QualityScore)
		}
	})

	t.Run("fails on duplicate photo within session", func(t *testing.T) {
		_, err := svc.UploadPhoto(ctx, sessID, dummyImage, "image/jpeg", "web_camera", callerID, callerID, true)
		if err == nil {
			t.Fatal("expected duplicate error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeDuplicatePhoto {
				t.Errorf("expected code %s, got %s", httpx.CodeDuplicatePhoto, appErr.Code)
			}
		}
	})

	t.Run("fails when low quality / not usable", func(t *testing.T) {
		engine.SetOverride(func(imageBytes []byte) (*inference.EmbedResult, error) {
			return &inference.EmbedResult{
				Usable:       false,
				QualityScore: 0.45,
				Hints:        []string{"too_blurry", "poor_lighting"},
			}, nil
		})
		defer engine.SetOverride(nil)

		_, err := svc.UploadPhoto(ctx, sessID, []byte("different-photo-bytes"), "image/jpeg", "web_camera", callerID, callerID, true)
		if err == nil {
			t.Fatal("expected quality error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeFaceNotUsable {
				t.Errorf("expected code %s, got %s", httpx.CodeFaceNotUsable, appErr.Code)
			}
		}
	})
}

func TestCommitSession_Flow(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepo()
	store, _ := storage.NewLocalStore(t.TempDir())
	engine := inference.NewFakeClient()
	consent := &mockConsent{hasConsent: true}

	svc := enrollment.NewService(repo, store, engine, nil, consent, nil, nil)

	callerID := uuid.New()
	employeeID := uuid.New()
	sessID := uuid.New()

	session := &enrollment.Session{
		ID:             sessID,
		EmployeeID:     employeeID,
		CreatedBy:      callerID,
		Status:         "draft",
		Mode:           "replace",
		ModelVersion:   "buffalo_l",
		RequiredPhotos: 3,
		MaxPhotos:      5,
		ExpiresAt:      time.Now().Add(15 * time.Minute),
		CreatedAt:      time.Now(),
	}
	repo.sessions[sessID] = session

	t.Run("fails when photo count below required", func(t *testing.T) {
		// Only upload 1 photo
		_, err := svc.UploadPhoto(ctx, sessID, []byte("photo-1"), "image/jpeg", "web_camera", callerID, callerID, true)
		if err != nil {
			t.Fatalf("upload photo 1 failed: %v", err)
		}

		_, err = svc.CommitSession(ctx, sessID, enrollment.CommitSessionRequest{}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected incomplete photos error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeEnrollmentIncomplete {
				t.Errorf("expected code %s, got %s", httpx.CodeEnrollmentIncomplete, appErr.Code)
			}
		}
	})

	t.Run("fails when duplicate face detected without override", func(t *testing.T) {
		// Upload photo 2 and 3 to reach 3 photos
		_, _ = svc.UploadPhoto(ctx, sessID, []byte("photo-2"), "image/jpeg", "web_camera", callerID, callerID, true)
		_, _ = svc.UploadPhoto(ctx, sessID, []byte("photo-3"), "image/jpeg", "web_camera", callerID, callerID, true)

		// Set mock duplicates
		otherEmpID := uuid.New()
		repo.duplicates = []enrollment.DuplicateMatch{
			{EmployeeID: otherEmpID, Similarity: 0.92},
		}

		_, err := svc.CommitSession(ctx, sessID, enrollment.CommitSessionRequest{}, callerID, callerID, true)
		if err == nil {
			t.Fatal("expected duplicate face error, got nil")
		}
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			if appErr.Code != httpx.CodeFaceBelongsToAnotherEmployee {
				t.Errorf("expected code %s, got %s", httpx.CodeFaceBelongsToAnotherEmployee, appErr.Code)
			}
		}
	})

	t.Run("successful commit with force duplicate override", func(t *testing.T) {
		reason := "Twin sibling approved by HR"
		resp, err := svc.CommitSession(ctx, sessID, enrollment.CommitSessionRequest{
			ForceDuplicate: true,
			Reason:         &reason,
		}, callerID, callerID, true)
		if err != nil {
			t.Fatalf("commit with force duplicate failed: %v", err)
		}
		if resp == nil {
			t.Fatalf("expected non-nil response")
		}
		if len(resp.CreatedReferenceIDs) != 3 {
			t.Errorf("expected 3 created references, got %d", len(resp.CreatedReferenceIDs))
		}
	})
}
