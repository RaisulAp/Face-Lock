package attendance

import (
	"time"

	"github.com/google/uuid"
)

// ClockType represents the direction of attendance.
type ClockType string

const (
	ClockTypeCheckIn  ClockType = "check_in"
	ClockTypeCheckOut ClockType = "check_out"
)

// AttendanceStatus represents the approval state of an attendance record.
type AttendanceStatus string

const (
	StatusApproved      AttendanceStatus = "approved"
	StatusRejected      AttendanceStatus = "rejected"
	StatusPendingReview AttendanceStatus = "pending_review"
)

// AttendanceMethod represents how the attendance verification was performed.
type AttendanceMethod string

const (
	MethodFaceVerified    AttendanceMethod = "face_verified"
	MethodFallbackManual  AttendanceMethod = "fallback_manual"
	MethodFallbackOffline AttendanceMethod = "fallback_offline"
	MethodSystemTimeout   AttendanceMethod = "system_timeout"
)

// FallbackReason represents standard documented reasons for fallback attendance.
type FallbackReason string

const (
	FallbackCameraFailure    FallbackReason = "camera_hardware_failure"
	FallbackEngineFailure    FallbackReason = "engine_unreachable"
	FallbackLighting         FallbackReason = "lighting_condition"
	FallbackFaceUnrecognized FallbackReason = "face_unrecognized"
	FallbackManualOverride   FallbackReason = "manual_override"
)

// Attendance represents the core domain model corresponding to table attendances.
type Attendance struct {
	ID                uuid.UUID        `json:"id"`
	EmployeeID        uuid.UUID        `json:"employee_id"`
	WorkDate          string           `json:"work_date"` // YYYY-MM-DD
	Type              ClockType        `json:"type"`
	Status            AttendanceStatus `json:"status"`
	Method            AttendanceMethod `json:"method"`
	ServerTimestamp   time.Time        `json:"server_timestamp"`
	ClientReportedAt  time.Time        `json:"client_reported_at"`
	ClockSkewSeconds  int              `json:"clock_skew_seconds"`
	Latitude          *float64         `json:"latitude,omitempty"`
	Longitude         *float64         `json:"longitude,omitempty"`
	DistanceMeter     *float64         `json:"distance_meter,omitempty"`
	MatchedOfficeID   *uuid.UUID       `json:"matched_office_id,omitempty"`
	LocationIsMocked  bool             `json:"location_is_mocked"`
	MatchedSimilarity *float64         `json:"matched_similarity,omitempty"`
	ThresholdUsed     *float64         `json:"threshold_used,omitempty"`
	ModelVersion      *string          `json:"model_version,omitempty"`
	QualityScore      *float64         `json:"quality_score,omitempty"`
	PhotoKey          *string          `json:"photo_key,omitempty"`
	PhotoPurgedAt     *time.Time       `json:"photo_purged_at,omitempty"`
	PhotoSHA256       *string          `json:"photo_sha256,omitempty"`
	PhotoBytes        *int             `json:"photo_bytes,omitempty"`
	PhotoMime         *string          `json:"photo_mime,omitempty"`
	LivenessPassed    *bool            `json:"liveness_passed,omitempty"`
	LivenessSupported bool             `json:"liveness_supported"`
	FallbackReason    *string          `json:"fallback_reason,omitempty"`
	FallbackNote      *string          `json:"fallback_note,omitempty"`
	ReviewedBy        *uuid.UUID       `json:"reviewed_by,omitempty"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
	ReviewNotes       *string          `json:"review_notes,omitempty"`
	IdempotencyKey    *string          `json:"idempotency_key,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`

	// Relational joins
	EmployeeName      string  `json:"employee_name,omitempty"`
	EmployeeNumber    string  `json:"employee_number,omitempty"`
	Department        *string `json:"department,omitempty"`
	MatchedOfficeName *string `json:"matched_office_name,omitempty"`
	ReviewedByName    *string `json:"reviewed_by_name,omitempty"`
}

// AttendanceAttempt represents the telemetry log row in attendance_attempts.
type AttendanceAttempt struct {
	ID                int64      `json:"id"`
	EmployeeID        uuid.UUID  `json:"employee_id"`
	Type              ClockType  `json:"type"`
	ServerTimestamp   time.Time  `json:"server_timestamp"`
	WorkDate          string     `json:"work_date"`
	Outcome           string     `json:"outcome"`
	MatchedSimilarity *float64   `json:"matched_similarity,omitempty"`
	ThresholdUsed     *float64   `json:"threshold_used,omitempty"`
	ModelVersion      *string    `json:"model_version,omitempty"`
	QualityScore      *float64   `json:"quality_score,omitempty"`
	Hints             []string   `json:"hints,omitempty"`
	GeofenceStatus    *string    `json:"geofence_status,omitempty"`
	DistanceMeter     *float64   `json:"distance_meter,omitempty"`
	AllowFallback     bool       `json:"allow_fallback"`
	AttendanceID      *uuid.UUID `json:"attendance_id,omitempty"`
	FailureReason     *string    `json:"failure_reason,omitempty"`
	RequestID         *string    `json:"request_id,omitempty"`
	IP                *string    `json:"ip,omitempty"`
	UserAgent         *string    `json:"user_agent,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`

	// Joins
	EmployeeName   string `json:"employee_name,omitempty"`
	EmployeeNumber string `json:"employee_number,omitempty"`
}
