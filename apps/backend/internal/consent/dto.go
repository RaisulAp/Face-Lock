package consent

import (
	"time"

	"github.com/google/uuid"
)

type ConsentDocumentResponse struct {
	Version     string     `json:"version"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	ContentHash string     `json:"content_hash"`
	PublishedAt *time.Time `json:"published_at"`
}

type ConsentMeResponse struct {
	Status           string     `json:"status"`
	DocumentVersion  string     `json:"document_version"`
	GrantedAt        *time.Time `json:"granted_at"`
	IsCurrentVersion bool       `json:"is_current_version"`
	Method           string     `json:"method"`
}

type GrantConsentRequest struct {
	DocumentVersion string `json:"document_version" validate:"required"`
	Agreed          bool   `json:"agreed"`
}

type GrantConsentResponse struct {
	Status          string    `json:"status"`
	DocumentVersion string    `json:"document_version"`
	GrantedAt       time.Time `json:"granted_at"`
}

type WithdrawConsentRequest struct {
	Reason string `json:"reason" validate:"required,min=3"`
}

type WithdrawConsentResponse struct {
	Status                    string    `json:"status"`
	WithdrawnAt               time.Time `json:"withdrawn_at"`
	DeactivatedReferenceCount int       `json:"deactivated_reference_count"`
	AttendanceModeHint        string    `json:"attendance_mode_hint"`
}

type RecordAdminConsentRequest struct {
	DocumentVersion string     `json:"document_version" validate:"required"`
	SignedAt        *time.Time `json:"signed_at,omitempty"`
	Note            string     `json:"note,omitempty"`
}

type EmployeeConsentResponse struct {
	Status           string     `json:"status"`
	DocumentVersion  string     `json:"document_version"`
	GrantedAt        *time.Time `json:"granted_at"`
	Method           string     `json:"method"`
	RecordedBy       *uuid.UUID `json:"recorded_by,omitempty"`
	WithdrawnAt      *time.Time `json:"withdrawn_at,omitempty"`
	WithdrawnReason  *string    `json:"withdrawn_reason,omitempty"`
	IsCurrentVersion bool       `json:"is_current_version"`
}
