package enrollment

import (
	"time"

	"github.com/google/uuid"
)

type CreateSessionRequest struct {
	EmployeeID *uuid.UUID `json:"employee_id,omitempty"`
	Mode       string     `json:"mode,omitempty"` // "replace" (default) or "append"
}

type SessionPhotoItem struct {
	ID            uuid.UUID `json:"id"`
	Position      int       `json:"position"`
	QualityScore  float32   `json:"quality_score"`
	DetScore      float32   `json:"det_score"`
	PhotoBytes    int       `json:"photo_bytes"`
	CaptureSource string    `json:"capture_source"`
	CreatedAt     time.Time `json:"created_at"`
}

type SessionResponse struct {
	ID             uuid.UUID          `json:"id"`
	EmployeeID     uuid.UUID          `json:"employee_id"`
	Status         string             `json:"status"`
	Mode           string             `json:"mode"`
	RequiredPhotos int                `json:"required_photos"`
	MaxPhotos      int                `json:"max_photos"`
	ModelVersion   string             `json:"model_version"`
	Photos         []SessionPhotoItem `json:"photos"`
	ExpiresAt      time.Time          `json:"expires_at"`
}

type UploadPhotoResponse struct {
	ID             uuid.UUID `json:"id"`
	Position       int       `json:"position"`
	QualityScore   float32   `json:"quality_score"`
	DetScore       float32   `json:"det_score"`
	Hints          []string  `json:"hints"`
	PhotoBytes     int       `json:"photo_bytes"`
	AcceptedCount  int       `json:"accepted_count"`
	RequiredPhotos int       `json:"required_photos"`
	CanCommit      bool      `json:"can_commit"`
}

type CommitSessionRequest struct {
	ForceDuplicate bool    `json:"force_duplicate"`
	Reason         *string `json:"reason"`
}

type CommitSessionResponse struct {
	SessionID               uuid.UUID   `json:"session_id"`
	EmployeeID              uuid.UUID   `json:"employee_id"`
	CommittedAt             time.Time   `json:"committed_at"`
	ModelVersion            string      `json:"model_version"`
	CreatedReferenceIDs     []uuid.UUID `json:"created_reference_ids"`
	DeactivatedReferenceIDs []uuid.UUID `json:"deactivated_reference_ids"`
	ActiveReferenceCount    int         `json:"active_reference_count"`
	MeanQualityScore        float32     `json:"mean_quality_score"`
}
