package attendance

import (
	"time"

	"github.com/google/uuid"
)

// ClockType denotes whether the action is check-in or check-out.
type ClockType string

const (
	ClockTypeIn  ClockType = "in"
	ClockTypeOut ClockType = "out"
)

// Status represents the verification outcome of the attendance record.
type Status string

const (
	StatusSuccess       Status = "success"
	StatusFailed        Status = "failed"
	StatusPendingReview Status = "pending_review"
)

// Record represents a single recorded clock-in or clock-out event.
type Record struct {
	ID              uuid.UUID `json:"id"`
	EmployeeID      uuid.UUID `json:"employee_id"`
	Type            ClockType `json:"type"`
	Status          Status    `json:"status"`
	PhotoKey        string    `json:"photo_key"`
	SimilarityScore *float64  `json:"similarity_score,omitempty"`
	Distance        *float64  `json:"distance,omitempty"`
	ModelVersion    string    `json:"model_version"`
	Notes           *string   `json:"notes,omitempty"`
	RecordedAt      time.Time `json:"recorded_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// ClockParams holds the input parameters for performing clock-in/out.
type ClockParams struct {
	EmployeeID  *uuid.UUID `json:"employee_id,omitempty"`
	PhotoBytes  []byte     `json:"-"`
	ContentType string     `json:"-"`
	Notes       *string    `json:"notes,omitempty"`
}

// VerificationResult encapsulates the biometric distance calculation details.
type VerificationResult struct {
	IsMatch         bool    `json:"is_match"`
	SimilarityScore float64 `json:"similarity_score"`
	Distance        float64 `json:"distance"`
	Threshold       float64 `json:"threshold"`
}
