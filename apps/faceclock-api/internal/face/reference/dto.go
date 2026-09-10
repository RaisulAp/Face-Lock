package reference

import (
	"time"

	"github.com/google/uuid"
)

type EnrolledByUser struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type ReferenceItem struct {
	ID                uuid.UUID       `json:"id"`
	Position          int             `json:"position"`
	QualityScore      float32         `json:"quality_score"`
	DetScore          float32         `json:"det_score"`
	ModelVersion      string          `json:"model_version"`
	CaptureSource     string          `json:"capture_source"`
	IsActive          bool            `json:"is_active"`
	PhotoURL          string          `json:"photo_url"`
	EnrolledAt        time.Time       `json:"enrolled_at"`
	EnrolledBy        *EnrolledByUser `json:"enrolled_by,omitempty"`
	DeactivatedAt     *time.Time      `json:"deactivated_at,omitempty"`
	DeactivatedReason *string         `json:"deactivated_reason,omitempty"`
}

type ReferenceListMeta struct {
	ActiveCount   int  `json:"active_count"`
	RequiredCount int  `json:"required_count"`
	IsEnrolled    bool `json:"is_enrolled"`
}

type ReferenceListResponse struct {
	Data []ReferenceItem   `json:"data"`
	Meta ReferenceListMeta `json:"meta"`
}

type DeactivateReferenceRequest struct {
	IsActive          bool    `json:"is_active"`
	Reason            *string `json:"reason,omitempty"`
	AllowBelowMinimum bool    `json:"allow_below_minimum"`
}

type DeactivateReferenceResponse struct {
	ReferenceID          uuid.UUID `json:"reference_id"`
	IsActive             bool      `json:"is_active"`
	DeactivatedAt        time.Time `json:"deactivated_at"`
	DeactivatedReason    *string   `json:"deactivated_reason,omitempty"`
	RemainingActiveCount int       `json:"remaining_active_count"`
}

type DeleteFaceDataRequest struct {
	Reason string `json:"reason"`
}

type DeleteFaceDataResponse struct {
	DeletedReferenceCount int    `json:"deleted_reference_count"`
	PhotosDeleted         int    `json:"photos_deleted"`
	ConsentStatus         string `json:"consent_status"`
}

type ConsentStatusSummary struct {
	Status           string `json:"status"`
	DocumentVersion  string `json:"document_version"`
	IsCurrentVersion bool   `json:"is_current_version"`
}

type EnrollmentStatusResponse struct {
	Consent              ConsentStatusSummary `json:"consent"`
	AttendanceMode       string               `json:"attendance_mode"`
	IsEnrolled           bool                 `json:"is_enrolled"`
	ActiveReferenceCount int                  `json:"active_reference_count"`
	RequiredPhotos       int                  `json:"required_photos"`
	MaxPhotos            int                  `json:"max_photos"`
	ModelVersionMatches  bool                 `json:"model_version_matches"`
	NeedsReEnrollment    bool                 `json:"needs_re_enrollment"`
	DraftSessionID       *uuid.UUID           `json:"draft_session_id"`
}
