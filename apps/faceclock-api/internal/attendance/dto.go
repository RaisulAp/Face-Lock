package attendance

import (
	"time"

	"github.com/google/uuid"
)

// ClockRequest carries all payload data for check-in and check-out requests.
type ClockRequest struct {
	PhotoBytes       []byte
	PhotoMime        string
	Latitude         *float64
	Longitude        *float64
	Accuracy         *float64
	ClientReportedAt time.Time
	LocationIsMocked bool
	Notes            *string
	IdempotencyKey   *string
	AllowFallback    bool
	FallbackReason   *string
	FallbackNote     *string
	RequestID        string
	IP               string
	UserAgent        string
}

// EmployeeAttendanceDTO is the sanitized, safe attendance representation for employees.
// In accordance with Plan/05-Fase4.md § 2.5 (Anti-Score Leakage):
// - similarity_score is NEVER exposed.
// - threshold is NEVER exposed.
// - model_version is NEVER exposed.
// - distance_meter is NEVER exposed (only in_radius boolean).
type EmployeeAttendanceDTO struct {
	ID              uuid.UUID        `json:"id"`
	EmployeeID      uuid.UUID        `json:"employee_id"`
	WorkDate        string           `json:"work_date"`
	Type            ClockType        `json:"type"`
	Status          AttendanceStatus `json:"status"`
	Method          AttendanceMethod `json:"method"`
	ServerTimestamp time.Time        `json:"server_timestamp"`
	OfficeName      *string          `json:"office_name,omitempty"`
	InRadius        bool             `json:"in_radius"`
	FallbackUsed    bool             `json:"fallback_used"`
	FallbackReason  *string          `json:"fallback_reason,omitempty"`
	IsLate          bool             `json:"is_late,omitempty"`
	LateMinutes     int              `json:"late_minutes,omitempty"`
	IsEarlyLeave    bool             `json:"is_early_leave,omitempty"`
	EarlyMinutes    int              `json:"early_leave_minutes,omitempty"`
	ReviewNotes     *string          `json:"review_notes,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

// AdminAttendanceDTO provides the comprehensive attendance telemetry for managers, HR, and admins.
type AdminAttendanceDTO struct {
	ID                uuid.UUID        `json:"id"`
	EmployeeID        uuid.UUID        `json:"employee_id"`
	EmployeeName      string           `json:"employee_name,omitempty"`
	EmployeeNumber    string           `json:"employee_number,omitempty"`
	Department        *string          `json:"department,omitempty"`
	WorkDate          string           `json:"work_date"`
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
	MatchedOfficeName *string          `json:"matched_office_name,omitempty"`
	LocationIsMocked  bool             `json:"location_is_mocked"`
	MatchedSimilarity *float64         `json:"matched_similarity,omitempty"`
	ThresholdUsed     *float64         `json:"threshold_used,omitempty"`
	ModelVersion      *string          `json:"model_version,omitempty"`
	QualityScore      *float64         `json:"quality_score,omitempty"`
	PhotoKey          *string          `json:"photo_key,omitempty"`
	PhotoPurgedAt     *time.Time       `json:"photo_purged_at,omitempty"`
	PhotoSHA256       *string          `json:"photo_sha256,omitempty"`
	LivenessPassed    *bool            `json:"liveness_passed,omitempty"`
	LivenessSupported bool             `json:"liveness_supported"`
	FallbackReason    *string          `json:"fallback_reason,omitempty"`
	FallbackNote      *string          `json:"fallback_note,omitempty"`
	ReviewedBy        *uuid.UUID       `json:"reviewed_by,omitempty"`
	ReviewedByName    *string          `json:"reviewed_by_name,omitempty"`
	ReviewedAt        *time.Time       `json:"reviewed_at,omitempty"`
	ReviewNotes       *string          `json:"review_notes,omitempty"`
	WaitingHours      *float64         `json:"waiting_hours,omitempty"`
	IdempotencyKey    *string          `json:"idempotency_key,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

// OfficeSummaryDTO provides basic office information for geofence validation on client.
type OfficeSummaryDTO struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	RadiusMeters float64   `json:"radius_meters"`
	Address      *string   `json:"address,omitempty"`
}

// AttendanceContextResponse supplies today's schedule, current status, and geofence locations for employee.
type AttendanceContextResponse struct {
	ServerTime       time.Time              `json:"server_time"`
	WorkDate         string                 `json:"work_date"`
	TodayCheckIn     *EmployeeAttendanceDTO `json:"today_check_in,omitempty"`
	TodayCheckOut    *EmployeeAttendanceDTO `json:"today_check_out,omitempty"`
	CanCheckIn       bool                   `json:"can_check_in"`
	CanCheckOut      bool                   `json:"can_check_out"`
	HasEnrolledFace  bool                   `json:"has_enrolled_face"`
	WorkStartTime    string                 `json:"work_start_time"`
	WorkEndTime      string                 `json:"work_end_time"`
	LateToleranceMin int                    `json:"late_tolerance_minutes"`
	EarlyLeaveMin    int                    `json:"early_leave_tolerance_minutes"`
	ActiveOffices    []OfficeSummaryDTO     `json:"active_offices"`
}

// ReviewRequest handles single record review.
type ReviewRequest struct {
	Action          string  `json:"action"` // "approve" or "reject"
	ReviewNote      string  `json:"review_note"`
	ReviewNotes     string  `json:"review_notes"`
	RejectionReason *string `json:"rejection_reason,omitempty"`
}

// BulkReviewItemRequest represents an individual item in a bulk review payload.
type BulkReviewItemRequest struct {
	ID         uuid.UUID `json:"id"`
	Action     string    `json:"action"` // "approve" or "reject"
	ReviewNote string    `json:"review_note"`
}

// BulkReviewRequest handles bulk review of multiple pending records (max 50 items).
type BulkReviewRequest struct {
	Items           []BulkReviewItemRequest `json:"items,omitempty"`
	IDs             []uuid.UUID             `json:"ids,omitempty"`
	Action          string                  `json:"action,omitempty"`
	ReviewNotes     string                  `json:"review_notes,omitempty"`
	RejectionReason *string                 `json:"rejection_reason,omitempty"`
}

// BulkReviewError details a single item failure.
type BulkReviewError struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// BulkReviewResultItem represents the outcome of reviewing one item in a batch.
type BulkReviewResultItem struct {
	ID     uuid.UUID         `json:"id"`
	Status *AttendanceStatus `json:"status,omitempty"`
	Error  *BulkReviewError  `json:"error,omitempty"`
}

// BulkReviewResponse summarizes the bulk review operation.
type BulkReviewResponse struct {
	Processed int                    `json:"processed"`
	Succeeded int                    `json:"succeeded"`
	Failed    int                    `json:"failed"`
	Results   []BulkReviewResultItem `json:"results"`
}

// ReviewerInfo provides reviewer identity details.
type ReviewerInfo struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name,omitempty"`
}

// ApproveResponse is returned on successful attendance approval (#62).
type ApproveResponse struct {
	ID         uuid.UUID        `json:"id"`
	Status     AttendanceStatus `json:"status"`
	ReviewedBy ReviewerInfo     `json:"reviewed_by"`
	ReviewedAt time.Time        `json:"reviewed_at"`
}

// RejectResponse is returned on successful attendance rejection (#63).
type RejectResponse struct {
	ID               uuid.UUID        `json:"id"`
	Status           AttendanceStatus `json:"status"`
	EmployeeCanRetry bool             `json:"employee_can_retry"`
	ReviewedBy       ReviewerInfo     `json:"reviewed_by"`
	ReviewedAt       time.Time        `json:"reviewed_at"`
}

// AttendanceStatsResponse returns summary aggregate statistics.
type AttendanceStatsResponse struct {
	StartDate       string `json:"start_date"`
	EndDate         string `json:"end_date"`
	TotalRecords    int    `json:"total_records"`
	ApprovedCount   int    `json:"approved_count"`
	PendingCount    int    `json:"pending_count"`
	RejectedCount   int    `json:"rejected_count"`
	CheckInCount    int    `json:"check_in_count"`
	CheckOutCount   int    `json:"check_out_count"`
	LateCount       int    `json:"late_count"`
	EarlyLeaveCount int    `json:"early_leave_count"`
	FallbackCount   int    `json:"fallback_count"`
}

// AttendanceFilter carries search and filter criteria for attendance records.
type AttendanceFilter struct {
	Page         int
	PerPage      int
	EmployeeID   *uuid.UUID
	Department   string
	StartDate    string // YYYY-MM-DD
	EndDate      string // YYYY-MM-DD
	WorkDate     string // YYYY-MM-DD
	Status       string
	Type         string
	Method       string
	Search       string
	IsPending    bool
	OnlyFallback bool
}

// AttemptFilter carries search and filter criteria for attendance telemetry attempts.
type AttemptFilter struct {
	Page       int
	PerPage    int
	EmployeeID *uuid.UUID
	Outcome    string
	Type       string
	StartDate  string
	EndDate    string
}
