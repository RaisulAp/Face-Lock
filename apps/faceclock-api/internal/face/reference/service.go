package reference

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
)

var (
	ErrPhotoPurged = errors.New("face_reference: photo purged by retention policy")
)

type ConsentDocVersionProvider interface {
	GetCurrentDocumentVersion(ctx context.Context) (string, error)
}

type Service struct {
	repo       Repository
	store      storage.Store
	settings   *settings.Service
	consentDoc ConsentDocVersionProvider
	audit      *audit.Recorder
	logger     *slog.Logger
}

func NewService(
	repo Repository,
	store storage.Store,
	settings *settings.Service,
	consentDoc ConsentDocVersionProvider,
	audit *audit.Recorder,
	logger *slog.Logger,
) *Service {
	return &Service{
		repo:       repo,
		store:      store,
		settings:   settings,
		consentDoc: consentDoc,
		audit:      audit,
		logger:     logger,
	}
}

func (s *Service) ListReferences(
	ctx context.Context,
	employeeID uuid.UUID,
	includeInactive bool,
	callerID, callerEmpID uuid.UUID,
	canReadAny bool,
) (*ReferenceListResponse, error) {
	if !canReadAny && employeeID != callerEmpID {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Karyawan tidak ditemukan")
	}

	exists, err := s.repo.EmployeeExists(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("check employee exists: %w", err)
	}
	if !exists {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Karyawan tidak ditemukan")
	}

	refs, err := s.repo.ListByEmployee(ctx, employeeID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list references: %w", err)
	}

	activeCount, err := s.repo.CountActive(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("count active references: %w", err)
	}

	requiredPhotos := s.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if requiredPhotos <= 0 {
		requiredPhotos = 3
	}

	items := make([]ReferenceItem, 0, len(refs))
	for _, r := range refs {
		var enrolledBy *EnrolledByUser
		if r.EnrolledBy != nil {
			email := ""
			if r.EnrolledByEmail != nil {
				email = *r.EnrolledByEmail
			}
			enrolledBy = &EnrolledByUser{
				ID:    *r.EnrolledBy,
				Email: email,
			}
		}

		item := ReferenceItem{
			ID:                r.ID,
			Position:          r.Position,
			QualityScore:      r.QualityScore,
			DetScore:          r.DetScore,
			ModelVersion:      r.ModelVersion,
			CaptureSource:     r.CaptureSource,
			IsActive:          r.IsActive,
			PhotoURL:          fmt.Sprintf("/api/v1/face/references/%s/photo", r.ID),
			EnrolledAt:        r.CreatedAt,
			EnrolledBy:        enrolledBy,
			DeactivatedAt:     r.DeactivatedAt,
			DeactivatedReason: r.DeactivatedReason,
		}
		items = append(items, item)
	}

	return &ReferenceListResponse{
		Data: items,
		Meta: ReferenceListMeta{
			ActiveCount:   activeCount,
			RequiredCount: requiredPhotos,
			IsEnrolled:    activeCount >= requiredPhotos,
		},
	}, nil
}

func (s *Service) GetPhoto(
	ctx context.Context,
	referenceID uuid.UUID,
	callerID, callerEmpID uuid.UUID,
	canReadAny bool,
) (io.ReadCloser, string, error) {
	ref, err := s.repo.GetByID(ctx, referenceID)
	if err != nil {
		if errors.Is(err, ErrReferenceNotFound) {
			return nil, "", httpx.NewAppError(httpx.CodeNotFound, "Foto referensi tidak ditemukan")
		}
		return nil, "", fmt.Errorf("get reference: %w", err)
	}

	// 410 Gone if photo purged by retention
	if ref.PhotoPurgedAt != nil {
		return nil, "", ErrPhotoPurged
	}

	// Permission guard
	if !canReadAny && ref.EmployeeID != callerEmpID {
		return nil, "", httpx.NewAppError(httpx.CodeNotFound, "Foto referensi tidak ditemukan")
	}

	// Audit log when viewing other employee's photo
	if canReadAny && ref.EmployeeID != callerEmpID && s.audit != nil {
		empIDStr := ref.EmployeeID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.reference.photo_viewed",
			ActorUserID:  &callerID,
			ResourceType: "face_reference",
			ResourceID:   &empIDStr,
			Metadata: map[string]any{
				"reference_id": referenceID.String(),
				"employee_id":  empIDStr,
			},
		})
	}

	reader, err := s.store.Get(ctx, ref.PhotoKey)
	if err != nil {
		return nil, "", fmt.Errorf("read photo from store: %w", err)
	}

	mimeType := ref.PhotoMIME
	if mimeType == "" {
		mimeType = "image/jpeg"
	}

	return reader, mimeType, nil
}

func (s *Service) DeactivateReference(
	ctx context.Context,
	referenceID uuid.UUID,
	req DeactivateReferenceRequest,
	callerID uuid.UUID,
) (*DeactivateReferenceResponse, error) {
	if req.IsActive {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "Mengaktifkan kembali referensi tidak diizinkan. Lakukan enrollment baru jika diperlukan")
	}

	ref, err := s.repo.GetByID(ctx, referenceID)
	if err != nil {
		if errors.Is(err, ErrReferenceNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Referensi wajah tidak ditemukan")
		}
		return nil, fmt.Errorf("get reference: %w", err)
	}

	if !ref.IsActive {
		return nil, httpx.NewAppError(httpx.CodeConflict, "Referensi wajah sudah dalam keadaan tidak aktif")
	}

	activeCount, err := s.repo.CountActive(ctx, ref.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf("count active: %w", err)
	}

	minPhotos := s.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if minPhotos <= 0 {
		minPhotos = 3
	}

	if (activeCount-1) < minPhotos && !req.AllowBelowMinimum {
		return nil, httpx.NewAppError(httpx.CodeEnrollmentIncomplete, fmt.Sprintf("Penonaktifan akan membuat referensi aktif karyawan di bawah batas minimum (%d). Sertakan allow_below_minimum=true untuk mengonfirmasi", minPhotos))
	}

	reason := "manual_deactivation"
	if req.Reason != nil && strings.TrimSpace(*req.Reason) != "" {
		reason = strings.TrimSpace(*req.Reason)
	}

	updated, err := s.repo.Deactivate(ctx, referenceID, reason)
	if err != nil {
		return nil, fmt.Errorf("deactivate reference: %w", err)
	}

	remainingCount, _ := s.repo.CountActive(ctx, ref.EmployeeID)

	if s.audit != nil {
		refIDStr := referenceID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.reference.deactivated",
			ActorUserID:  &callerID,
			ResourceType: "face_reference",
			ResourceID:   &refIDStr,
			Metadata: map[string]any{
				"reference_id":           refIDStr,
				"employee_id":            ref.EmployeeID.String(),
				"reason":                 reason,
				"remaining_active_count": remainingCount,
			},
		})
	}

	return &DeactivateReferenceResponse{
		ReferenceID:          updated.ID,
		IsActive:             updated.IsActive,
		DeactivatedAt:        *updated.DeactivatedAt,
		DeactivatedReason:    updated.DeactivatedReason,
		RemainingActiveCount: remainingCount,
	}, nil
}

func (s *Service) DeleteFaceData(
	ctx context.Context,
	employeeID uuid.UUID,
	reason string,
	callerID uuid.UUID,
) (*DeleteFaceDataResponse, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "Alasan (reason) penghapusan data wajah wajib diisi")
	}

	exists, err := s.repo.EmployeeExists(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("check employee exists: %w", err)
	}
	if !exists {
		return nil, httpx.NewAppError(httpx.CodeNotFound, "Karyawan tidak ditemukan")
	}

	deletedCount, photoKeys, err := s.repo.DeleteAllForEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("delete all face data in db: %w", err)
	}

	// Delete from storage
	photosDeleted := 0
	for _, k := range photoKeys {
		if err := s.store.Delete(ctx, k); err == nil {
			photosDeleted++
		}
	}

	if s.audit != nil {
		empIDStr := employeeID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			Action:       "face.data.erased",
			ActorUserID:  &callerID,
			ResourceType: "employee",
			ResourceID:   &empIDStr,
			Metadata: map[string]any{
				"employee_id":     empIDStr,
				"reason":          reason,
				"deleted_count":   deletedCount,
				"photos_deleted":  photosDeleted,
				"consent_status":  "withdrawn",
				"attendance_mode": "manual",
			},
		})
	}

	return &DeleteFaceDataResponse{
		DeletedReferenceCount: deletedCount,
		PhotosDeleted:         photosDeleted,
		ConsentStatus:         "withdrawn",
	}, nil
}

func (s *Service) GetEnrollmentStatus(
	ctx context.Context,
	employeeID uuid.UUID,
) (*EnrollmentStatusResponse, error) {
	currentDocVersion := "2026-09-v1"
	if s.consentDoc != nil {
		if ver, err := s.consentDoc.GetCurrentDocumentVersion(ctx); err == nil && ver != "" {
			currentDocVersion = ver
		}
	}

	summary, err := s.repo.GetEmployeeFaceSummary(ctx, employeeID, currentDocVersion)
	if err != nil {
		if errors.Is(err, ErrEmployeeNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Karyawan tidak ditemukan")
		}
		return nil, fmt.Errorf("get employee face summary: %w", err)
	}

	requiredPhotos := s.settings.GetInt(ctx, "face.min_reference_photos", 3)
	if requiredPhotos <= 0 {
		requiredPhotos = 3
	}
	maxPhotos := s.settings.GetInt(ctx, "face.max_reference_photos", 5)
	if maxPhotos <= 0 {
		maxPhotos = 5
	}

	currentModelVer := s.settings.GetString(ctx, "face.model_version", "buffalo_l")
	modelMatches := (summary.ActiveRefCount > 0 && summary.LatestModelVer == currentModelVer)
	isEnrolled := summary.ActiveRefCount >= requiredPhotos
	needsReEnrollment := (isEnrolled && !modelMatches)

	return &EnrollmentStatusResponse{
		Consent: ConsentStatusSummary{
			Status:           summary.ConsentStatus,
			DocumentVersion:  summary.ConsentDocVer,
			IsCurrentVersion: summary.IsCurrentDocVer,
		},
		AttendanceMode:       summary.AttendanceMode,
		IsEnrolled:           isEnrolled,
		ActiveReferenceCount: summary.ActiveRefCount,
		RequiredPhotos:       requiredPhotos,
		MaxPhotos:            maxPhotos,
		ModelVersionMatches:  modelMatches,
		NeedsReEnrollment:    needsReEnrollment,
		DraftSessionID:       summary.DraftSessionID,
	}, nil
}
