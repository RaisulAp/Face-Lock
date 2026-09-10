package consent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/google/uuid"
)

type Service struct {
	repo     Repository
	cache    *Cache
	settings *settings.Service
	audit    *audit.Recorder
}

func NewService(repo Repository, cache *Cache, settings *settings.Service, audit *audit.Recorder) *Service {
	return &Service{
		repo:     repo,
		cache:    cache,
		settings: settings,
		audit:    audit,
	}
}

// GetActiveDocument retrieves the active consent document for display.
func (s *Service) GetActiveDocument(ctx context.Context) (*ConsentDocumentResponse, error) {
	doc, err := s.repo.GetActiveDocument(ctx)
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "no active consent document found")
		}
		return nil, fmt.Errorf("consent svc: get active document: %w", err)
	}

	return &ConsentDocumentResponse{
		Version:     doc.Version,
		Title:       doc.Title,
		Body:        doc.Body,
		ContentHash: doc.ContentHash,
		PublishedAt: doc.PublishedAt,
	}, nil
}

// GetCurrentDocumentVersion returns the version string of the active consent document.
func (s *Service) GetCurrentDocumentVersion(ctx context.Context) (string, error) {
	doc, err := s.GetActiveDocument(ctx)
	if err != nil {
		return "", err
	}
	return doc.Version, nil
}

// GetMyConsent retrieves the current authenticated employee's consent status.
func (s *Service) GetMyConsent(ctx context.Context, employeeID uuid.UUID) (*ConsentMeResponse, error) {
	activeDoc, docErr := s.repo.GetActiveDocument(ctx)

	consent, err := s.repo.GetLatestConsentByEmployeeID(ctx, employeeID)
	if err != nil {
		if errors.Is(err, ErrConsentNotFound) {
			res := &ConsentMeResponse{
				Status: "none",
			}
			if docErr == nil && activeDoc != nil {
				res.DocumentVersion = activeDoc.Version
			}
			return res, nil
		}
		return nil, fmt.Errorf("consent svc: get latest consent: %w", err)
	}

	isCurrent := false
	if docErr == nil && activeDoc != nil && consent.DocumentVersion == activeDoc.Version {
		isCurrent = true
	}

	res := &ConsentMeResponse{
		Status:           consent.Status,
		DocumentVersion:  consent.DocumentVersion,
		GrantedAt:        &consent.GrantedAt,
		IsCurrentVersion: isCurrent,
		Method:           consent.Method,
	}
	return res, nil
}

// GrantConsent records a self-serve biometric consent from the employee.
func (s *Service) GrantConsent(
	ctx context.Context,
	employeeID uuid.UUID,
	req GrantConsentRequest,
	ip, userAgent string,
	actorUserID *uuid.UUID,
) (*GrantConsentResponse, error) {
	if !req.Agreed {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "agreed must be true to grant consent")
	}

	doc, err := s.repo.GetDocumentByVersion(ctx, req.DocumentVersion)
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			return nil, httpx.NewAppError(httpx.CodeConsentVersionOutdated, fmt.Sprintf("consent document version %q not found", req.DocumentVersion))
		}
		return nil, fmt.Errorf("consent svc: get document: %w", err)
	}

	if !doc.IsActive {
		return nil, httpx.NewAppError(httpx.CodeConsentVersionOutdated, "cannot grant consent on inactive or deprecated document version")
	}

	// Double check if employee already has active consent
	hasActive, err := s.repo.HasActiveConsent(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("consent svc: check active consent: %w", err)
	}
	if hasActive {
		return nil, httpx.NewAppError(httpx.CodeConsentAlreadyGranted, "biometric consent has already been granted")
	}

	c := &BiometricConsent{
		ID:              uuid.New(),
		EmployeeID:      employeeID,
		DocumentVersion: doc.Version,
		ContentHash:     doc.ContentHash,
		Method:          "self_web",
		RecordedBy:      actorUserID,
	}
	if ip != "" {
		c.IP = &ip
	}
	if userAgent != "" {
		c.UserAgent = &userAgent
	}

	if err := s.repo.GrantConsent(ctx, c); err != nil {
		if errors.Is(err, ErrConsentAlreadyGranted) {
			return nil, httpx.NewAppError(httpx.CodeConsentAlreadyGranted, "biometric consent has already been granted")
		}
		return nil, fmt.Errorf("consent svc: grant consent: %w", err)
	}

	s.cache.Set(employeeID, true)

	if s.audit != nil {
		resID := c.ID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			ActorUserID:  actorUserID,
			Action:       "consent.granted",
			ResourceType: "consent",
			ResourceID:   &resID,
			Metadata: map[string]any{
				"employee_id":      employeeID.String(),
				"document_version": doc.Version,
				"content_hash":     doc.ContentHash,
				"method":           "self_web",
			},
			IP:        ip,
			UserAgent: userAgent,
		})
	}

	return &GrantConsentResponse{
		Status:          c.Status,
		DocumentVersion: c.DocumentVersion,
		GrantedAt:       c.GrantedAt,
	}, nil
}

// WithdrawConsent withdraws an employee's biometric consent, which deactivates their face references.
func (s *Service) WithdrawConsent(
	ctx context.Context,
	employeeID uuid.UUID,
	req WithdrawConsentRequest,
	ip, userAgent string,
	actorUserID *uuid.UUID,
) (*WithdrawConsentResponse, error) {
	if req.Reason == "" {
		return nil, httpx.NewAppError(httpx.CodeValidationError, "withdrawal reason is required")
	}

	deactivatedCount, err := s.repo.WithdrawConsent(ctx, employeeID, req.Reason)
	if err != nil {
		if errors.Is(err, ErrConsentNotFound) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "no active consent found to withdraw")
		}
		return nil, fmt.Errorf("consent svc: withdraw: %w", err)
	}

	s.cache.Set(employeeID, false)

	if s.audit != nil {
		empStr := employeeID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			ActorUserID:  actorUserID,
			Action:       "consent.withdrawn",
			ResourceType: "consent",
			ResourceID:   &empStr,
			Metadata: map[string]any{
				"employee_id":           employeeID.String(),
				"reason":                req.Reason,
				"deactivated_ref_count": deactivatedCount,
			},
			IP:        ip,
			UserAgent: userAgent,
		})

		if deactivatedCount > 0 {
			_ = s.audit.Record(ctx, audit.LogEntry{
				ActorUserID:  actorUserID,
				Action:       "face.references.deactivated",
				ResourceType: "face_reference",
				ResourceID:   &empStr,
				Metadata: map[string]any{
					"employee_id": employeeID.String(),
					"reason":      "consent_withdrawn",
					"count":       deactivatedCount,
				},
				IP:        ip,
				UserAgent: userAgent,
			})
		}
	}

	return &WithdrawConsentResponse{
		Status:                    "withdrawn",
		WithdrawnAt:               time.Now().UTC(),
		DeactivatedReferenceCount: deactivatedCount,
		AttendanceModeHint:        "manual",
	}, nil
}

// GetEmployeeConsent retrieves consent status for a specific employee (Admin view).
func (s *Service) GetEmployeeConsent(ctx context.Context, employeeID uuid.UUID) (*EmployeeConsentResponse, error) {
	activeDoc, _ := s.repo.GetActiveDocument(ctx)

	consent, err := s.repo.GetLatestConsentByEmployeeID(ctx, employeeID)
	if err != nil {
		if errors.Is(err, ErrConsentNotFound) {
			res := &EmployeeConsentResponse{
				Status: "none",
			}
			if activeDoc != nil {
				res.DocumentVersion = activeDoc.Version
			}
			return res, nil
		}
		return nil, fmt.Errorf("consent svc: get employee consent: %w", err)
	}

	isCurrent := false
	if activeDoc != nil && consent.DocumentVersion == activeDoc.Version {
		isCurrent = true
	}

	return &EmployeeConsentResponse{
		Status:           consent.Status,
		DocumentVersion:  consent.DocumentVersion,
		GrantedAt:        &consent.GrantedAt,
		Method:           consent.Method,
		RecordedBy:       consent.RecordedBy,
		WithdrawnAt:      consent.WithdrawnAt,
		WithdrawnReason:  consent.WithdrawnReason,
		IsCurrentVersion: isCurrent,
	}, nil
}

// AdminRecordConsent allows an HR/Admin to record offline consent on behalf of an employee.
func (s *Service) AdminRecordConsent(
	ctx context.Context,
	employeeID uuid.UUID,
	req RecordAdminConsentRequest,
	ip, userAgent string,
	adminUserID uuid.UUID,
) (*EmployeeConsentResponse, error) {
	doc, err := s.repo.GetDocumentByVersion(ctx, req.DocumentVersion)
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			return nil, httpx.NewAppError(httpx.CodeConsentVersionOutdated, fmt.Sprintf("consent document version %q not found", req.DocumentVersion))
		}
		return nil, fmt.Errorf("consent svc: get document: %w", err)
	}

	if !doc.IsActive {
		return nil, httpx.NewAppError(httpx.CodeConsentVersionOutdated, "cannot record consent on inactive document version")
	}

	// Check if already active
	hasActive, err := s.repo.HasActiveConsent(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("consent svc: check active: %w", err)
	}
	if hasActive {
		return nil, httpx.NewAppError(httpx.CodeConsentAlreadyGranted, "employee already has active biometric consent")
	}

	grantedAt := time.Now().UTC()
	if req.SignedAt != nil {
		grantedAt = *req.SignedAt
	}

	c := &BiometricConsent{
		ID:              uuid.New(),
		EmployeeID:      employeeID,
		DocumentVersion: doc.Version,
		ContentHash:     doc.ContentHash,
		Method:          "admin_recorded",
		GrantedAt:       grantedAt,
		RecordedBy:      &adminUserID,
	}
	if ip != "" {
		c.IP = &ip
	}
	if userAgent != "" {
		c.UserAgent = &userAgent
	}

	if err := s.repo.GrantConsent(ctx, c); err != nil {
		if errors.Is(err, ErrConsentAlreadyGranted) {
			return nil, httpx.NewAppError(httpx.CodeConsentAlreadyGranted, "employee already has active biometric consent")
		}
		return nil, fmt.Errorf("consent svc: grant consent by admin: %w", err)
	}

	s.cache.Set(employeeID, true)

	if s.audit != nil {
		resID := c.ID.String()
		_ = s.audit.Record(ctx, audit.LogEntry{
			ActorUserID:  &adminUserID,
			Action:       "consent.admin_recorded",
			ResourceType: "consent",
			ResourceID:   &resID,
			Metadata: map[string]any{
				"employee_id":      employeeID.String(),
				"document_version": doc.Version,
				"content_hash":     doc.ContentHash,
				"note":             req.Note,
				"method":           "admin_recorded",
			},
			IP:        ip,
			UserAgent: userAgent,
		})
	}

	return &EmployeeConsentResponse{
		Status:           "granted",
		DocumentVersion:  doc.Version,
		GrantedAt:        &c.GrantedAt,
		Method:           "admin_recorded",
		RecordedBy:       &adminUserID,
		IsCurrentVersion: true,
	}, nil
}

// HasActiveConsent checks whether an employee has active consent, checking cache and settings.
func (s *Service) HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	// 1. Check if consent requirement is enabled in app_settings
	if s.settings != nil {
		required := s.settings.GetBool(ctx, "face.consent_required", true)
		if !required {
			return true, nil
		}
	}

	// 2. Check cache
	if hasConsent, ok := s.cache.Get(employeeID); ok {
		return hasConsent, nil
	}

	// 3. Query DB
	hasConsent, err := s.repo.HasActiveConsent(ctx, employeeID)
	if err != nil {
		return false, err
	}

	// 4. Populate cache
	s.cache.Set(employeeID, hasConsent)
	return hasConsent, nil
}

// ComputeHash computes SHA256 hex string of document content.
func ComputeHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
