package enrollment

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
)

type ConsentChecker interface {
	HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error)
}

type Service struct {
	repo     Repository
	store    storage.Store
	engine   inference.FaceEngine
	settings *settings.Service
	consent  ConsentChecker
	audit    *audit.Recorder
	logger   *slog.Logger
}

func NewService(
	repo Repository,
	store storage.Store,
	engine inference.FaceEngine,
	settings *settings.Service,
	consent ConsentChecker,
	audit *audit.Recorder,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:     repo,
		store:    store,
		engine:   engine,
		settings: settings,
		consent:  consent,
		audit:    audit,
		logger:   logger,
	}
}

func (s *Service) CreateSession(
	ctx context.Context,
	req CreateSessionRequest,
	callerID, callerEmployeeID uuid.UUID,
	canEnrollAny bool,
) (*SessionResponse, error) {
	var targetEmployeeID uuid.UUID
	if req.EmployeeID != nil && *req.EmployeeID != uuid.Nil {
		if !canEnrollAny && *req.EmployeeID != callerEmployeeID {
			return nil, httpx.NewAppError(httpx.CodeForbidden, "Akses ditolak: tidak memiliki izin face.enroll_any")
		}
		targetEmployeeID = *req.EmployeeID
	} else {
		if callerEmployeeID == uuid.Nil {
			return nil, httpx.NewAppError(httpx.CodeValidationError, "Akun ini tidak tertaut dengan data karyawan")
		}
		targetEmployeeID = callerEmployeeID
	}

	// Verify employee exists and is active
	exists, err := s.repo.EmployeeExists(ctx, targetEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("check employee: %w", err)
	}
	if !exists {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Data karyawan tidak ditemukan atau sudah nonaktif")
	}

	// Check model_version
	modelVersion := s.settings.GetString(ctx, "face.model_version", "buffalo_l")
	if modelVersion == "" || modelVersion == "unset" {
		return nil, httpx.NewAppError(httpx.CodeFaceServiceNotConfigured, "Layanan biometrik wajah belum dikonfigurasi (model belum dikalibrasi)")
	}

	// Check consent
	if s.settings.GetBool(ctx, "face.consent_required", true) && s.consent != nil {
		hasConsent, err := s.consent.HasActiveConsent(ctx, targetEmployeeID)
		if err != nil {
			return nil, fmt.Errorf("check consent: %w", err)
		}
		if !hasConsent {
			return nil, httpx.NewAppError(httpx.CodeConsentRequired, "Karyawan belum memberikan persetujuan (consent) pemrosesan data biometrik")
		}
	}

	// Check attendance_mode
	mode, err := s.repo.GetEmployeeAttendanceMode(ctx, targetEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("get attendance mode: %w", err)
	}
	if mode == "manual" {
		return nil, httpx.NewAppError(httpx.CodeAttendanceModeManual, "Karyawan berada pada jalur absensi manual")
	}

	// Check reindex running
	reindexRunning, err := s.repo.IsReindexRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("check reindex: %w", err)
	}
	if reindexRunning {
		return nil, httpx.NewAppError(httpx.CodeReindexInProgress, "Proses reindex wajah sedang berlangsung, silakan coba lagi nanti")
	}

	// Check existing draft session
	existingDraft, err := s.repo.GetActiveDraftSession(ctx, targetEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("check active draft: %w", err)
	}
	if existingDraft != nil {
		appErr := httpx.NewAppError(httpx.CodeConflict, "Sudah ada sesi enrollment draft yang masih aktif")
		appErr.Extra = map[string]any{
			"existing_session_id": existingDraft.ID.String(),
		}
		return nil, appErr
	}

	// Mode validation
	sessionMode := strings.ToLower(strings.TrimSpace(req.Mode))
	if sessionMode == "" {
		sessionMode = "replace"
	}
	if sessionMode != "replace" && sessionMode != "append" {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "Mode enrollment harus 'replace' atau 'append'")
	}

	maxPhotos := s.settings.GetInt(ctx, "face.max_reference_photos", 5)
	if maxPhotos <= 0 {
		maxPhotos = 5
	}
	minPhotos := s.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if minPhotos <= 0 {
		minPhotos = 3
	}

	if sessionMode == "append" {
		activeCount, err := s.repo.CountActiveReferences(ctx, targetEmployeeID)
		if err != nil {
			return nil, fmt.Errorf("count active references: %w", err)
		}
		if activeCount >= maxPhotos {
			return nil, httpx.NewAppError(httpx.CodeEnrollmentLimitReached, fmt.Sprintf("Jumlah foto referensi sudah mencapai batas maksimum (%d)", maxPhotos))
		}
	}

	sessionID := uuid.New()
	expiresAt := time.Now().Add(30 * time.Minute)

	session := &Session{
		ID:             sessionID,
		EmployeeID:     targetEmployeeID,
		CreatedBy:      callerID,
		Status:         "draft",
		Mode:           sessionMode,
		ModelVersion:   modelVersion,
		RequiredPhotos: minPhotos,
		MaxPhotos:      maxPhotos,
		ExpiresAt:      expiresAt,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &SessionResponse{
		ID:             session.ID,
		EmployeeID:     session.EmployeeID,
		Status:         session.Status,
		Mode:           session.Mode,
		RequiredPhotos: session.RequiredPhotos,
		MaxPhotos:      session.MaxPhotos,
		ModelVersion:   session.ModelVersion,
		Photos:         []SessionPhotoItem{},
		ExpiresAt:      session.ExpiresAt,
	}, nil
}

func (s *Service) GetSession(
	ctx context.Context,
	sessionID uuid.UUID,
	callerID, callerEmployeeID uuid.UUID,
	canReadAny bool,
) (*SessionResponse, error) {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
		}
		return nil, err
	}

	if !canReadAny && sess.EmployeeID != callerEmployeeID && sess.CreatedBy != callerID {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
	}

	if sess.Status == "draft" && sess.ExpiresAt.Before(time.Now()) {
		_ = s.repo.ExpireSession(ctx, sessionID)
		sess.Status = "expired"
	}

	photos, err := s.repo.ListPhotosBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list session photos: %w", err)
	}

	photoItems := make([]SessionPhotoItem, 0, len(photos))
	for _, p := range photos {
		photoItems = append(photoItems, SessionPhotoItem{
			ID:            p.ID,
			Position:      p.Position,
			QualityScore:  p.QualityScore,
			DetScore:      p.DetScore,
			PhotoBytes:    p.PhotoBytes,
			CaptureSource: p.CaptureSource,
			CreatedAt:     p.CreatedAt,
		})
	}

	return &SessionResponse{
		ID:             sess.ID,
		EmployeeID:     sess.EmployeeID,
		Status:         sess.Status,
		Mode:           sess.Mode,
		RequiredPhotos: sess.RequiredPhotos,
		MaxPhotos:      sess.MaxPhotos,
		ModelVersion:   sess.ModelVersion,
		Photos:         photoItems,
		ExpiresAt:      sess.ExpiresAt,
	}, nil
}

func (s *Service) UploadPhoto(
	ctx context.Context,
	sessionID uuid.UUID,
	imageBytes []byte,
	mimeType, captureSource string,
	callerID, callerEmployeeID uuid.UUID,
	canEnrollAny bool,
) (*UploadPhotoResponse, error) {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
		}
		return nil, err
	}

	if !canEnrollAny && sess.EmployeeID != callerEmployeeID && sess.CreatedBy != callerID {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
	}

	// Validate captureSource
	captureSource = strings.ToLower(strings.TrimSpace(captureSource))
	if captureSource != "web_camera" && captureSource != "mobile_camera" && captureSource != "admin_upload" {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "capture_source harus 'web_camera', 'mobile_camera', atau 'admin_upload'")
	}
	if captureSource == "admin_upload" && !canEnrollAny {
		return nil, httpx.NewAppError(httpx.CodeForbidden, "capture_source 'admin_upload' hanya diizinkan untuk admin")
	}

	// Check status
	if sess.Status != "draft" || sess.ExpiresAt.Before(time.Now()) {
		if sess.Status == "draft" {
			_ = s.repo.ExpireSession(ctx, sessionID)
		}
		return nil, httpx.NewAppError(httpx.CodeEnrollmentSessionExpired, "Sesi enrollment sudah kedaluwarsa atau tidak aktif")
	}

	// Check image bytes limit
	maxBytes := s.settings.GetInt(ctx, "face.max_image_bytes", 5242880)
	if maxBytes <= 0 {
		maxBytes = 5242880
	}
	if len(imageBytes) > maxBytes {
		return nil, httpx.NewAppError(httpx.CodePayloadTooLarge, fmt.Sprintf("Ukuran foto melebihi batas maksimum %d bytes", maxBytes))
	}

	// Check sha256 duplicate within session
	hasher := sha256.New()
	hasher.Write(imageBytes)
	photoHash := hex.EncodeToString(hasher.Sum(nil))

	hashExists, err := s.repo.PhotoHashExists(ctx, sessionID, photoHash)
	if err != nil {
		return nil, fmt.Errorf("check photo hash: %w", err)
	}
	if hashExists {
		return nil, httpx.NewAppError(httpx.CodeDuplicatePhoto, "Foto yang sama sudah diunggah pada sesi ini")
	}

	// Check photo limit
	photos, err := s.repo.ListPhotosBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list photos: %w", err)
	}
	if len(photos) >= sess.MaxPhotos {
		return nil, httpx.NewAppError(httpx.CodeEnrollmentLimitReached, fmt.Sprintf("Jumlah foto sudah mencapai batas maksimum sesi (%d)", sess.MaxPhotos))
	}

	// Inference: detect and embed
	detResult, err := s.engine.DetectAndEmbed(ctx, imageBytes)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, httpx.NewAppError(httpx.CodeUpstreamTimeout, "Layanan deteksi wajah melebihi batas waktu (timeout)")
		}
		return nil, httpx.NewAppError(httpx.CodeUpstreamError, "Gagal memproses deteksi wajah pada layanan upstream")
	}

	minQuality := float32(s.settings.GetFloat(ctx, "face.min_quality_score", 0.65))
	if minQuality <= 0 {
		minQuality = 0.65
	}

	// Gate quality
	if !detResult.Usable {
		appErr := httpx.NewAppError(httpx.CodeFaceNotUsable, "Foto belum memenuhi syarat kualitas")
		appErr.Extra = map[string]any{"hints": detResult.Hints}
		appErr.Details = []httpx.FieldError{{Field: "image", Message: "Foto belum memenuhi syarat"}}
		return nil, appErr
	}

	if detResult.QualityScore < minQuality {
		hints := append(detResult.Hints, "quality_below_reference_standard")
		appErr := httpx.NewAppError(httpx.CodeFaceNotUsable, "Kualitas foto di bawah standar referensi")
		appErr.Extra = map[string]any{"hints": hints}
		appErr.Details = []httpx.FieldError{{Field: "image", Message: "Kualitas foto di bawah standar referensi"}}
		return nil, appErr
	}

	if len(detResult.Embedding) == 0 {
		return nil, httpx.NewAppError(httpx.CodeUpstreamError, "Inference upstream tidak mengembalikan vektor embedding")
	}

	// Validate or detect MIME type
	if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
		detected := http.DetectContentType(imageBytes)
		if detected == "image/jpeg" || detected == "image/png" || detected == "image/webp" {
			mimeType = detected
		} else {
			return nil, httpx.NewAppError(httpx.CodeValidationError, "Format file bukan image/jpeg, image/png, atau image/webp")
		}
	}

	// Staging photo upload to storage
	photoID := uuid.New()
	stagingKey := storage.StagingPhotoKey(sessionID, photoID)

	_, err = s.store.Put(ctx, stagingKey, bytes.NewReader(imageBytes), mimeType)
	if err != nil {
		return nil, fmt.Errorf("save staging photo: %w", err)
	}

	position, err := s.repo.NextPosition(ctx, sessionID)
	if err != nil {
		_ = s.store.Delete(ctx, stagingKey)
		return nil, fmt.Errorf("next position: %w", err)
	}

	p := &Photo{
		ID:            photoID,
		SessionID:     sessionID,
		Position:      position,
		Embedding:     detResult.Embedding,
		QualityScore:  detResult.QualityScore,
		DetScore:      detResult.DetScore,
		PhotoKey:      stagingKey,
		PhotoSHA256:   photoHash,
		PhotoBytes:    len(imageBytes),
		PhotoMIME:     mimeType,
		CaptureSource: captureSource,
	}

	if err := s.repo.InsertPhoto(ctx, p); err != nil {
		_ = s.store.Delete(ctx, stagingKey)
		return nil, fmt.Errorf("insert photo record: %w", err)
	}

	acceptedCount := len(photos) + 1
	canCommit := acceptedCount >= sess.RequiredPhotos

	return &UploadPhotoResponse{
		ID:             photoID,
		Position:       position,
		QualityScore:   detResult.QualityScore,
		DetScore:       detResult.DetScore,
		Hints:          detResult.Hints,
		PhotoBytes:     len(imageBytes),
		AcceptedCount:  acceptedCount,
		RequiredPhotos: sess.RequiredPhotos,
		CanCommit:      canCommit,
	}, nil
}

func (s *Service) DeletePhoto(
	ctx context.Context,
	sessionID, photoID uuid.UUID,
	callerID, callerEmployeeID uuid.UUID,
	canEnrollAny bool,
) error {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
		}
		return err
	}

	if !canEnrollAny && sess.EmployeeID != callerEmployeeID && sess.CreatedBy != callerID {
		return httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
	}

	if sess.Status != "draft" {
		return httpx.NewAppError(httpx.CodeConflict, "Hanya foto pada sesi draft yang dapat dihapus")
	}

	photo, err := s.repo.GetPhotoByID(ctx, sessionID, photoID)
	if err != nil {
		if errors.Is(err, ErrPhotoNotFound) {
			return httpx.NewAppError(httpx.CodeNotFound, "Foto tidak ditemukan pada sesi ini")
		}
		return err
	}

	_ = s.store.Delete(ctx, photo.PhotoKey)
	if err := s.repo.DeletePhoto(ctx, sessionID, photoID); err != nil {
		return fmt.Errorf("delete photo repo: %w", err)
	}

	return nil
}

func (s *Service) CommitSession(
	ctx context.Context,
	sessionID uuid.UUID,
	req CommitSessionRequest,
	callerID, callerEmployeeID uuid.UUID,
	canEnrollAny bool,
) (*CommitSessionResponse, error) {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
		}
		return nil, err
	}

	if !canEnrollAny && sess.EmployeeID != callerEmployeeID && sess.CreatedBy != callerID {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
	}

	if req.ForceDuplicate {
		if !canEnrollAny {
			return nil, httpx.NewAppError(httpx.CodeForbidden, "Hanya admin pemegang izin face.enroll_any yang dapat melakukan override duplikat")
		}
		if req.Reason == nil || strings.TrimSpace(*req.Reason) == "" {
			return nil, httpx.NewAppError(httpx.CodeValidationError, "Alasan (reason) wajib diisi saat force_duplicate aktif")
		}
	}

	if sess.Status != "draft" || sess.ExpiresAt.Before(time.Now()) {
		if sess.Status == "draft" {
			_ = s.repo.ExpireSession(ctx, sessionID)
		}
		return nil, httpx.NewAppError(httpx.CodeEnrollmentSessionExpired, "Sesi enrollment sudah kedaluwarsa atau bukan status draft")
	}

	currentModelVersion := s.settings.GetString(ctx, "face.model_version", "buffalo_l")
	if sess.ModelVersion != currentModelVersion {
		return nil, httpx.NewAppError(httpx.CodeEnrollmentModelChanged, "Versi model biometrik telah diperbarui di tengah sesi enrollment")
	}

	// Consent re-check
	if s.settings.GetBool(ctx, "face.consent_required", true) && s.consent != nil {
		hasConsent, err := s.consent.HasActiveConsent(ctx, sess.EmployeeID)
		if err != nil {
			return nil, fmt.Errorf("check consent at commit: %w", err)
		}
		if !hasConsent {
			return nil, httpx.NewAppError(httpx.CodeConsentRequired, "Persetujuan consent biometrik telah dicabut sebelum sesi diselesaikan")
		}
	}

	// Reindex check
	reindexRunning, err := s.repo.IsReindexRunning(ctx)
	if err != nil {
		return nil, fmt.Errorf("check reindex at commit: %w", err)
	}
	if reindexRunning {
		return nil, httpx.NewAppError(httpx.CodeReindexInProgress, "Proses reindex sedang berlangsung, tidak dapat melakukan commit enrollment")
	}

	// Check photos count
	photos, err := s.repo.ListPhotosBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list staging photos for commit: %w", err)
	}
	if len(photos) < sess.RequiredPhotos {
		msg := fmt.Sprintf("Jumlah foto diterima (%d) belum mencukupi batas minimum (%d)", len(photos), sess.RequiredPhotos)
		appErr := httpx.NewAppError(httpx.CodeEnrollmentIncomplete, msg)
		appErr.Details = []httpx.FieldError{{Field: "photos", Message: msg}}
		return nil, appErr
	}

	// Duplicate face detection
	dupThreshold := float32(s.settings.GetFloat(ctx, "face.duplicate_threshold", 0.40))
	if dupThreshold <= 0 {
		dupThreshold = float32(s.settings.GetFloat(ctx, "face.similarity_threshold", 0.35)) + 0.05
	}

	empIDStr := sess.EmployeeID.String()
	for _, p := range photos {
		matches, err := s.repo.CheckDuplicateFaces(ctx, p.Embedding, sess.ModelVersion, sess.EmployeeID, dupThreshold)
		if err != nil {
			return nil, fmt.Errorf("duplicate check: %w", err)
		}
		if len(matches) > 0 {
			match := matches[0]
			if !req.ForceDuplicate {
				if s.audit != nil {
					_ = s.audit.Record(ctx, audit.LogEntry{
						Action:       "face.enrollment.duplicate_detected",
						ActorUserID:  &callerID,
						ResourceType: "employee",
						ResourceID:   &empIDStr,
						Metadata: map[string]any{
							"session_id":          sessionID.String(),
							"matched_employee_id": match.EmployeeID.String(),
							"similarity":          match.Similarity,
							"threshold":           dupThreshold,
						},
					})
				}
				return nil, httpx.NewAppError(httpx.CodeFaceBelongsToAnotherEmployee, "Wajah terdeteksi mirip dengan data biometrik karyawan lain")
			}

			// Force duplicate override audit
			if s.audit != nil {
				_ = s.audit.Record(ctx, audit.LogEntry{
					Action:       "face.enrollment.duplicate_override",
					ActorUserID:  &callerID,
					ResourceType: "employee",
					ResourceID:   &empIDStr,
					Metadata: map[string]any{
						"session_id":          sessionID.String(),
						"matched_employee_id": match.EmployeeID.String(),
						"similarity":          match.Similarity,
						"reason":              *req.Reason,
					},
				})
			}
		}
	}

	// Prepare permanent storage copy
	type photoMove struct {
		oldKey string
		newKey string
	}
	var moves []photoMove
	permPhotos := make([]Photo, len(photos))
	var totalQuality float32
	for i, p := range photos {
		refID := uuid.New()
		permKey := storage.ReferencePhotoKey(sess.EmployeeID, refID)
		if err := s.store.Copy(ctx, p.PhotoKey, permKey); err != nil {
			return nil, fmt.Errorf("copy photo to permanent storage: %w", err)
		}
		moves = append(moves, photoMove{oldKey: p.PhotoKey, newKey: permKey})

		cp := p
		cp.ID = refID
		cp.PhotoKey = permKey
		permPhotos[i] = cp
		totalQuality += p.QualityScore
	}

	// Execute DB commit
	createdIDs, deactivatedIDs, err := s.repo.CommitSession(ctx, sess, permPhotos, callerID)
	if err != nil {
		// Clean up newly copied permanent keys on DB failure
		for _, m := range moves {
			_ = s.store.Delete(ctx, m.newKey)
		}
		return nil, fmt.Errorf("commit session db: %w", err)
	}

	// Post-commit: delete staging objects
	for _, m := range moves {
		_ = s.store.Delete(ctx, m.oldKey)
	}

	meanQuality := totalQuality / float32(len(photos))

	if s.audit != nil {
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.enrollment.committed",
			ActorUserID:  &callerID,
			ResourceType: "employee",
			ResourceID:   &empIDStr,
			Metadata: map[string]any{
				"session_id":         sessionID.String(),
				"reference_count":    len(createdIDs),
				"mean_quality_score": meanQuality,
				"mode":               sess.Mode,
			},
		})
	}

	activeCount, _ := s.repo.CountActiveReferences(ctx, sess.EmployeeID)

	return &CommitSessionResponse{
		SessionID:               sessionID,
		EmployeeID:              sess.EmployeeID,
		CommittedAt:             time.Now(),
		ModelVersion:            sess.ModelVersion,
		CreatedReferenceIDs:     createdIDs,
		DeactivatedReferenceIDs: deactivatedIDs,
		ActiveReferenceCount:    activeCount,
		MeanQualityScore:        meanQuality,
	}, nil
}

func (s *Service) CancelSession(
	ctx context.Context,
	sessionID uuid.UUID,
	callerID, callerEmployeeID uuid.UUID,
	canEnrollAny bool,
) error {
	sess, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
		}
		return err
	}

	if !canEnrollAny && sess.EmployeeID != callerEmployeeID && sess.CreatedBy != callerID {
		return httpx.NewAppError(httpx.CodeNotFound, "Sesi enrollment tidak ditemukan")
	}

	if sess.Status != "draft" {
		return httpx.NewAppError(httpx.CodeConflict, "Hanya sesi draft yang dapat dibatalkan")
	}

	photos, _ := s.repo.ListPhotosBySession(ctx, sessionID)
	for _, p := range photos {
		_ = s.store.Delete(ctx, p.PhotoKey)
	}

	if err := s.repo.CancelSession(ctx, sessionID); err != nil {
		return fmt.Errorf("cancel session repo: %w", err)
	}

	return nil
}
