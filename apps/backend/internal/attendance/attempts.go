package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RecordAttempt logs an attendance attempt to the attendance_attempts telemetry table.
func RecordAttempt(ctx context.Context, db *pgxpool.Pool, att AttendanceAttempt) error {
	hintsJSON, _ := json.Marshal(att.Hints)
	if len(att.Hints) == 0 || string(hintsJSON) == "null" {
		hintsJSON = []byte("[]")
	}

	var ipVal *string
	if att.IP != nil && *att.IP != "" {
		host, _, err := net.SplitHostPort(*att.IP)
		if err != nil {
			host = *att.IP
		}
		if parsed := net.ParseIP(host); parsed != nil {
			cleanIP := parsed.String()
			ipVal = &cleanIP
		}
	}

	const q = `
		INSERT INTO attendance_attempts (
			employee_id, type, server_timestamp, work_date, outcome,
			matched_similarity, threshold_used, model_version, quality_score,
			hints, geofence_status, distance_meter, allow_fallback,
			attendance_id, failure_reason, request_id, ip, user_agent, created_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16, $17, $18, NOW()
		)
	`
	_, err := db.Exec(ctx, q,
		att.EmployeeID,
		string(att.Type),
		att.ServerTimestamp,
		att.WorkDate,
		att.Outcome,
		att.MatchedSimilarity,
		att.ThresholdUsed,
		att.ModelVersion,
		att.QualityScore,
		hintsJSON,
		att.GeofenceStatus,
		att.DistanceMeter,
		att.AllowFallback,
		att.AttendanceID,
		att.FailureReason,
		att.RequestID,
		ipVal,
		att.UserAgent,
	)
	if err != nil {
		log.Printf("ERROR inserting attendance attempt: %v", err)
		return fmt.Errorf("inserting attendance attempt: %w", err)
	}
	return nil
}

// CheckFailedAttemptsRateLimit checks Rule B12:
// Rejects when failed attempts for an employee exceed maxFailed within windowSeconds.
func CheckFailedAttemptsRateLimit(ctx context.Context, db *pgxpool.Pool, employeeID uuid.UUID, windowSeconds int, maxFailed int) (bool, int, error) {
	if maxFailed <= 0 || windowSeconds <= 0 {
		return false, 0, nil
	}

	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)
	const q = `
		SELECT count(*)
		FROM attendance_attempts
		WHERE employee_id = $1
		  AND server_timestamp >= $2
		  AND outcome NOT IN ('success', 'fallback_used')
	`
	var count int
	err := db.QueryRow(ctx, q, employeeID, since).Scan(&count)
	if err != nil {
		return false, 0, fmt.Errorf("checking failed attempts rate limit: %w", err)
	}

	return count >= maxFailed, count, nil
}
