package attendance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/geo"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/storage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrPhotoPurged indicates an attendance photo has already been removed by the retention policy.
var ErrPhotoPurged = errors.New("photo purged")

// Service orchestrates the complete Attendance Engine workflow.
type Service struct {
	db         *pgxpool.Pool
	faceEngine face.FaceEngine
	store      storage.Store
	settings   *settings.Service
}

// NewService constructs a new attendance Service.
func NewService(db *pgxpool.Pool, engine face.FaceEngine, store storage.Store, setSvc *settings.Service) *Service {
	return &Service{
		db:         db,
		faceEngine: engine,
		store:      store,
		settings:   setSvc,
	}
}

// SetDependencies allows updating dependencies for testing or dynamic wiring.
func (s *Service) SetDependencies(engine face.FaceEngine, store storage.Store, setSvc *settings.Service) {
	s.faceEngine = engine
	s.store = store
	s.settings = setSvc
}

// Clock processes check-in or check-out with complete geofence, biometrics, anti-replay, and schedule rules.
func (s *Service) Clock(ctx context.Context, principal *rbac.Principal, clockType ClockType, req ClockRequest) (*Attendance, *EmployeeAttendanceDTO, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}
	empID := *principal.EmployeeID

	// 1. Fetch employee details
	const fetchEmpQ = `
		SELECT id, employee_number, full_name, department, employment_status, deleted_at
		FROM employees
		WHERE id = $1
	`
	var (
		empNumber, fullName, empStatus string
		department                     *string
		deletedAt                      *time.Time
	)
	err := s.db.QueryRow(ctx, fetchEmpQ, empID).Scan(
		&empID,
		&empNumber,
		&fullName,
		&department,
		&empStatus,
		&deletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || deletedAt != nil {
			return nil, nil, httpx.NewAppError(httpx.CodeNotFound, "employee not found")
		}
		return nil, nil, fmt.Errorf("fetching employee: %w", err)
	}

	if empStatus != "active" {
		return nil, nil, httpx.NewAppError(httpx.CodeEmployeeInactive, "inactive employee cannot perform attendance")
	}

	// 2. Resolve timezone and work date
	tzStr := "Asia/Jakarta"
	if s.settings != nil {
		tzStr = s.settings.GetString(ctx, "company_timezone", "Asia/Jakarta")
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	cutoffHour := 4
	if s.settings != nil {
		cutoffHour = s.settings.GetInt(ctx, "workday_cutoff_hour", 4)
	}

	now := time.Now().In(loc)
	workDate := ComputeWorkDate(now, loc, cutoffHour).Format("2006-01-02")

	// 3. Rate limiting / Abuse Prevention (Rule B12)
	maxFailed := 5
	windowSec := 300
	if s.settings != nil {
		maxFailed = s.settings.GetInt(ctx, "attendance_max_failed_attempts", 5)
		windowSec = s.settings.GetInt(ctx, "attendance_failed_attempt_window_seconds", 300)
	}

	if exceeded, count, err := CheckFailedAttemptsRateLimit(ctx, s.db, empID, windowSec, maxFailed); err == nil && exceeded {
		reason := fmt.Sprintf("Too many failed attempts (%d/%d) in %d seconds", count, maxFailed, windowSec)
		_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
			EmployeeID:      empID,
			Type:            clockType,
			ServerTimestamp: now,
			WorkDate:        workDate,
			Outcome:         "rate_limited",
			FailureReason:   &reason,
			RequestID:       &req.RequestID,
			IP:              &req.IP,
			UserAgent:       &req.UserAgent,
		})
		return nil, nil, httpx.NewAppError(httpx.CodeTooManyFailedAttempts, reason)
	}

	// 4. Idempotency Check
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		existing, err := s.getByEmployeeWorkDateAndIdempotency(ctx, empID, workDate, *req.IdempotencyKey)
		if err == nil && existing != nil {
			// Idempotent replay: return existing record
			dto := s.mapToEmployeeDTO(existing, loc)
			return existing, dto, nil
		}
	}

	// 5. Clock skew evaluation
	maxSkewSec := 300
	if s.settings != nil {
		maxSkewSec = s.settings.GetInt(ctx, "attendance_max_skew_seconds", 300)
	}
	clientReported := req.ClientReportedAt
	if clientReported.IsZero() {
		clientReported = now
	}
	skewSec, _ := EvaluateClockSkew(clientReported, now, maxSkewSec)

	// 6. Geofencing Evaluation (Rule B3)
	geofenceEnabled := true
	if s.settings != nil {
		geofenceEnabled = s.settings.GetBool(ctx, "attendance_geofence_enabled", true)
	}

	var (
		matchedOfficeID   *uuid.UUID
		matchedOfficeName *string
		distanceMeter     *float64
		geofenceStatus    = "disabled"
		inRadius          = true
	)

	if geofenceEnabled {
		if req.Latitude == nil || req.Longitude == nil {
			failReason := "GPS coordinates are required"
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "rule_rejected",
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeLocationRequired, "GPS coordinates are required for attendance")
		}

		if req.Accuracy != nil && *req.Accuracy > 100.0 {
			failReason := fmt.Sprintf("GPS accuracy is too low: %.1fm (max 100m)", *req.Accuracy)
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "rule_rejected",
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeLocationInaccurate, failReason)
		}

		allowMock := false
		if s.settings != nil {
			allowMock = s.settings.GetBool(ctx, "attendance_allow_mock_location", false)
		}
		if req.LocationIsMocked && !allowMock {
			failReason := "Mock location provider detected"
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "rule_rejected",
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeOutsideGeofence, "mock location is not allowed")
		}

		offices, err := s.getActiveOffices(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("fetching active offices for geofence: %w", err)
		}

		geoResult := geo.Evaluate(*req.Latitude, *req.Longitude, offices)
		if geoResult.OfficeID != nil {
			matchedOfficeID = geoResult.OfficeID
			matchedOfficeName = &geoResult.OfficeName
			d := geoResult.DistanceMeter
			distanceMeter = &d
		}

		if !geoResult.Inside {
			geofenceStatus = "outside"
			inRadius = false
			failReason := fmt.Sprintf("Outside office geofence (distance: %.1fm)", geoResult.DistanceMeter)
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "geofence_rejected",
				GeofenceStatus:  &geofenceStatus,
				DistanceMeter:   distanceMeter,
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeOutsideGeofence, "you are outside the authorized office geofence")
		}

		geofenceStatus = "inside"
		inRadius = true
	}

	// 7. Check Anti-Replay Photo (Rule B4 / § 2.7c)
	if len(req.PhotoBytes) == 0 {
		return nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "attendance photo is required")
	}

	windowDays := 7
	if s.settings != nil {
		windowDays = s.settings.GetInt(ctx, "attendance.duplicate_photo_window_days", 7)
	}

	photoSHA := ComputePhotoSHA256(req.PhotoBytes)
	var existingPhotoAttID uuid.UUID
	const checkDupQ = `
		SELECT id FROM attendances
		WHERE employee_id = $1 AND photo_sha256 = $2 AND created_at > NOW() - make_interval(days => $3)
		LIMIT 1
	`
	err = s.db.QueryRow(ctx, checkDupQ, empID, photoSHA, windowDays).Scan(&existingPhotoAttID)
	if err == nil {
		failReason := "Photo has already been used for attendance by this employee"
		_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
			EmployeeID:      empID,
			Type:            clockType,
			ServerTimestamp: now,
			WorkDate:        workDate,
			Outcome:         "duplicate_photo",
			FailureReason:   &failReason,
			RequestID:       &req.RequestID,
			IP:              &req.IP,
			UserAgent:       &req.UserAgent,
		})
		return nil, nil, httpx.NewAppError(httpx.CodeDuplicatePhoto, "attendance photo has already been used (anti-replay check failed)")
	}

	// 8. Enforce Schedule & Sequencing Rules (Rule B9)
	if clockType == ClockTypeCheckIn {
		var existingCheckInID uuid.UUID
		const checkExistingIn = `
			SELECT id FROM attendances
			WHERE employee_id = $1 AND work_date = $2 AND type = 'check_in' AND status <> 'rejected'
			LIMIT 1
		`
		err = s.db.QueryRow(ctx, checkExistingIn, empID, workDate).Scan(&existingCheckInID)
		if err == nil {
			failReason := "Employee already checked in today"
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "rule_rejected",
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeAlreadyCheckedIn, "employee has already checked in for work date "+workDate)
		}
	}

	if clockType == ClockTypeCheckOut {
		var existingCheckOutID uuid.UUID
		const checkExistingOut = `
			SELECT id FROM attendances
			WHERE employee_id = $1 AND work_date = $2 AND type = 'check_out' AND status <> 'rejected'
			LIMIT 1
		`
		err = s.db.QueryRow(ctx, checkExistingOut, empID, workDate).Scan(&existingCheckOutID)
		if err == nil {
			failReason := "Employee already checked out today"
			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         "rule_rejected",
				FailureReason:   &failReason,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeAlreadyCheckedOut, "employee has already checked out for work date "+workDate)
		}

		// Must have check-in
		reqCheckIn := true
		if s.settings != nil {
			reqCheckIn = s.settings.GetBool(ctx, "attendance_checkout_requires_checkin", true)
		}
		var checkInTime time.Time
		var checkInID uuid.UUID
		const fetchInQ = `
			SELECT id, server_timestamp
			FROM attendances
			WHERE employee_id = $1 AND work_date = $2 AND type = 'check_in' AND status <> 'rejected'
			ORDER BY server_timestamp DESC
			LIMIT 1
		`
		err = s.db.QueryRow(ctx, fetchInQ, empID, workDate).Scan(&checkInID, &checkInTime)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) && reqCheckIn {
				failReason := "Cannot check out without check-in"
				_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
					EmployeeID:      empID,
					Type:            clockType,
					ServerTimestamp: now,
					WorkDate:        workDate,
					Outcome:         "rule_rejected",
					FailureReason:   &failReason,
					RequestID:       &req.RequestID,
					IP:              &req.IP,
					UserAgent:       &req.UserAgent,
				})
				return nil, nil, httpx.NewAppError(httpx.CodeCheckoutWithoutCheckin, "cannot check out without a valid check-in for today")
			}
		} else {
			// Check minimum interval
			minInterval := 5
			if s.settings != nil {
				minInterval = s.settings.GetInt(ctx, "attendance_checkout_min_interval_minutes", 5)
			}
			if minInterval > 0 && now.Sub(checkInTime) < time.Duration(minInterval)*time.Minute {
				failReason := fmt.Sprintf("Check out attempted too soon (< %d mins from check-in)", minInterval)
				_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
					EmployeeID:      empID,
					Type:            clockType,
					ServerTimestamp: now,
					WorkDate:        workDate,
					Outcome:         "rule_rejected",
					FailureReason:   &failReason,
					RequestID:       &req.RequestID,
					IP:              &req.IP,
					UserAgent:       &req.UserAgent,
				})
				return nil, nil, httpx.NewAppError(httpx.CodeCheckoutTooSoon, fmt.Sprintf("cannot check out within %d minutes of checking in", minInterval))
			}
		}
	}

	// 9. Face Verification & Fail-Secure Fallback (Rule B2 & D17)
	hasEnrolled, err := s.hasEnrolledFace(ctx, empID)
	if err != nil {
		return nil, nil, fmt.Errorf("checking enrolled face: %w", err)
	}
	if !hasEnrolled {
		failReason := "Employee has no enrolled face reference"
		_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
			EmployeeID:      empID,
			Type:            clockType,
			ServerTimestamp: now,
			WorkDate:        workDate,
			Outcome:         "no_reference",
			FailureReason:   &failReason,
			RequestID:       &req.RequestID,
			IP:              &req.IP,
			UserAgent:       &req.UserAgent,
		})
		return nil, nil, httpx.NewAppError(httpx.CodeFaceNotEnrolled, "employee does not have an enrolled face reference")
	}

	var (
		status            AttendanceStatus = StatusApproved
		method            AttendanceMethod = MethodFaceVerified
		matchedSimilarity *float64
		thresholdUsed     *float64
		modelVersion      *string
		qualityScore      *float64
		hints             []string
		livenessPassed    *bool
		fallbackReason    *string
		fallbackNote      *string
	)

	defaultThreshold := 0.60
	if s.faceEngine != nil {
		defaultThreshold = float64(s.faceEngine.Threshold())
	}
	if s.settings != nil {
		defaultThreshold = s.settings.GetFloat(ctx, "face.similarity_threshold", defaultThreshold)
	}
	thresholdUsed = &defaultThreshold

	fallbackAllowed := false
	if s.settings != nil {
		fallbackAllowed = s.settings.GetBool(ctx, "attendance_fallback_enabled", false)
	}

	// Analyze face using engine
	if s.faceEngine == nil {
		return nil, nil, httpx.NewAppError(httpx.CodeFaceServiceNotConfigured, "face engine is not configured")
	}

	detResult, err := s.faceEngine.DetectAndEmbed(ctx, req.PhotoBytes)
	if err != nil || detResult == nil || !detResult.IsUsable || len(detResult.Embedding) == 0 {
		// Face detection / quality issue
		if req.AllowFallback && fallbackAllowed {
			status = StatusPendingReview
			method = MethodFallbackManual
			fbReason := string(FallbackCameraFailure)
			if req.FallbackReason != nil && *req.FallbackReason != "" {
				fbReason = *req.FallbackReason
			}
			fallbackReason = &fbReason
			fallbackNote = req.FallbackNote
		} else {
			outcome := "face_not_usable"
			failMsg := "attendance photo does not meet quality requirements"
			if errors.Is(err, face.ErrNoFaceDetected) {
				failMsg = "no face detected in attendance photo"
			} else if errors.Is(err, face.ErrMultipleFaces) {
				failMsg = "multiple faces detected; ensure only one person is in frame"
			} else if detResult != nil && len(detResult.Hints) > 0 {
				failMsg = strings.Join(detResult.Hints, ", ")
				hints = detResult.Hints
			}

			_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
				EmployeeID:      empID,
				Type:            clockType,
				ServerTimestamp: now,
				WorkDate:        workDate,
				Outcome:         outcome,
				ThresholdUsed:   thresholdUsed,
				Hints:           hints,
				GeofenceStatus:  &geofenceStatus,
				DistanceMeter:   distanceMeter,
				FailureReason:   &failMsg,
				RequestID:       &req.RequestID,
				IP:              &req.IP,
				UserAgent:       &req.UserAgent,
			})
			return nil, nil, httpx.NewAppError(httpx.CodeFaceNotUsable, failMsg)
		}
	} else {
		// Face analysis succeeded
		qScore := float64(detResult.QualityScore)
		qualityScore = &qScore
		modVer := detResult.ModelVersion
		if modVer == "" {
			modVer = s.faceEngine.ModelVersion()
		}
		modelVersion = &modVer
		hints = detResult.Hints

		// Query similarity using pgvector
		liveVectorStr := face.VectorToPGVector(detResult.Embedding)
		sim, err := s.matchBestSimilarity(ctx, empID, liveVectorStr)
		if err != nil {
			return nil, nil, fmt.Errorf("evaluating face similarity: %w", err)
		}

		matchedSimilarity = &sim

		if sim < *thresholdUsed {
			// Biometric mismatch
			if req.AllowFallback && fallbackAllowed {
				status = StatusPendingReview
				method = MethodFallbackManual
				fbReason := string(FallbackFaceUnrecognized)
				if req.FallbackReason != nil && *req.FallbackReason != "" {
					fbReason = *req.FallbackReason
				}
				fallbackReason = &fbReason
				fallbackNote = req.FallbackNote
			} else {
				failReason := fmt.Sprintf("Face similarity %.4f is below threshold %.2f", sim, *thresholdUsed)
				_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
					EmployeeID:        empID,
					Type:              clockType,
					ServerTimestamp:   now,
					WorkDate:          workDate,
					Outcome:           "below_threshold",
					MatchedSimilarity: matchedSimilarity,
					ThresholdUsed:     thresholdUsed,
					ModelVersion:      modelVersion,
					QualityScore:      qualityScore,
					Hints:             hints,
					GeofenceStatus:    &geofenceStatus,
					DistanceMeter:     distanceMeter,
					FailureReason:     &failReason,
					RequestID:         &req.RequestID,
					IP:                &req.IP,
					UserAgent:         &req.UserAgent,
				})
				return nil, nil, httpx.NewAppError(httpx.CodeFaceMismatch, "face verification failed: biometric match did not meet threshold")
			}
		} else {
			// Matched!
			status = StatusApproved
			method = MethodFaceVerified
		}
	}

	// 10. Persist Photo to Object Store
	if s.store == nil {
		return nil, nil, httpx.NewAppError(httpx.CodeInternalError, "storage provider not configured")
	}

	attendanceID := uuid.New()
	targetPhotoKey := storage.AttendancePhotoKey(empID, string(clockType), attendanceID)
	photoMime := req.PhotoMime
	if photoMime == "" {
		photoMime = "image/jpeg"
	}
	storedKey, err := s.store.Put(ctx, targetPhotoKey, bytes.NewReader(req.PhotoBytes), photoMime)
	if err != nil {
		return nil, nil, fmt.Errorf("storing attendance photo: %w", err)
	}
	if storedKey != "" {
		targetPhotoKey = storedKey
	}
	photoBytesCount := len(req.PhotoBytes)

	// 11. Insert Attendance Record
	const insertAttQ = `
		INSERT INTO attendances (
			id, employee_id, work_date, type, status, method,
			server_timestamp, client_reported_at, clock_skew_seconds,
			latitude, longitude, distance_meter, matched_office_id, location_is_mocked,
			matched_similarity, threshold_used, model_version, quality_score,
			photo_key, photo_sha256, photo_bytes, photo_mime,
			liveness_passed, liveness_supported,
			fallback_reason, fallback_note,
			idempotency_key, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12, $13, $14,
			$15, $16, $17, $18,
			$19, $20, $21, $22,
			$23, $24,
			$25, $26,
			$27, NOW(), NOW()
		)
		RETURNING id, created_at, updated_at
	`

	att := Attendance{
		ID:                attendanceID,
		EmployeeID:        empID,
		WorkDate:          workDate,
		Type:              clockType,
		Status:            status,
		Method:            method,
		ServerTimestamp:   now,
		ClientReportedAt:  clientReported,
		ClockSkewSeconds:  skewSec,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		DistanceMeter:     distanceMeter,
		MatchedOfficeID:   matchedOfficeID,
		LocationIsMocked:  req.LocationIsMocked,
		MatchedSimilarity: matchedSimilarity,
		ThresholdUsed:     thresholdUsed,
		ModelVersion:      modelVersion,
		QualityScore:      qualityScore,
		PhotoKey:          &targetPhotoKey,
		PhotoSHA256:       &photoSHA,
		PhotoBytes:        &photoBytesCount,
		PhotoMime:         &photoMime,
		LivenessPassed:    livenessPassed,
		LivenessSupported: false,
		FallbackReason:    fallbackReason,
		FallbackNote:      fallbackNote,
		IdempotencyKey:    req.IdempotencyKey,
		EmployeeName:      fullName,
		EmployeeNumber:    empNumber,
		Department:        department,
		MatchedOfficeName: matchedOfficeName,
	}

	err = s.db.QueryRow(ctx, insertAttQ,
		att.ID, att.EmployeeID, att.WorkDate, string(att.Type), string(att.Status), string(att.Method),
		att.ServerTimestamp, att.ClientReportedAt, att.ClockSkewSeconds,
		att.Latitude, att.Longitude, att.DistanceMeter, att.MatchedOfficeID, att.LocationIsMocked,
		att.MatchedSimilarity, att.ThresholdUsed, att.ModelVersion, att.QualityScore,
		att.PhotoKey, att.PhotoSHA256, att.PhotoBytes, att.PhotoMime,
		att.LivenessPassed, att.LivenessSupported,
		att.FallbackReason, att.FallbackNote,
		att.IdempotencyKey,
	).Scan(&att.ID, &att.CreatedAt, &att.UpdatedAt)

	if err != nil {
		// Clean up photo from store if DB insert fails
		_ = s.store.Delete(ctx, targetPhotoKey)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "one_checkin_per_day") {
				return nil, nil, httpx.NewAppError(httpx.CodeAlreadyCheckedIn, "employee has already checked in for work date "+workDate)
			}
			if strings.Contains(pgErr.ConstraintName, "one_checkout_per_day") {
				return nil, nil, httpx.NewAppError(httpx.CodeAlreadyCheckedOut, "employee has already checked out for work date "+workDate)
			}
		}
		return nil, nil, fmt.Errorf("inserting attendance: %w", err)
	}

	// 12. Log Successful Attempt into attendance_attempts
	outcome := "matched"
	if status == StatusPendingReview {
		outcome = "manual_mode"
	}
	_ = RecordAttempt(ctx, s.db, AttendanceAttempt{
		EmployeeID:        empID,
		Type:              clockType,
		ServerTimestamp:   now,
		WorkDate:          workDate,
		Outcome:           outcome,
		MatchedSimilarity: matchedSimilarity,
		ThresholdUsed:     thresholdUsed,
		ModelVersion:      modelVersion,
		QualityScore:      qualityScore,
		Hints:             hints,
		GeofenceStatus:    &geofenceStatus,
		DistanceMeter:     distanceMeter,
		AllowFallback:     req.AllowFallback,
		AttendanceID:      &att.ID,
		RequestID:         &req.RequestID,
		IP:                &req.IP,
		UserAgent:         &req.UserAgent,
	})

	// 13. Map to EmployeeAttendanceDTO
	dto := s.mapToEmployeeDTO(&att, loc)
	dto.InRadius = inRadius

	return &att, dto, nil
}

// GetContext retrieves today's context, schedule, and geofence locations for employee (#55).
func (s *Service) GetContext(ctx context.Context, principal *rbac.Principal) (*AttendanceContextResponse, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}
	empID := *principal.EmployeeID

	tzStr := "Asia/Jakarta"
	if s.settings != nil {
		tzStr = s.settings.GetString(ctx, "company_timezone", "Asia/Jakarta")
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	cutoffHour := 4
	workStartTime := "08:00"
	workEndTime := "17:00"
	lateTolerance := 15
	earlyTolerance := 15
	if s.settings != nil {
		cutoffHour = s.settings.GetInt(ctx, "workday_cutoff_hour", 4)
		workStartTime = s.settings.GetString(ctx, "attendance_work_start_time", "08:00")
		workEndTime = s.settings.GetString(ctx, "attendance_work_end_time", "17:00")
		lateTolerance = s.settings.GetInt(ctx, "attendance_late_tolerance_minutes", 15)
		earlyTolerance = s.settings.GetInt(ctx, "attendance_early_leave_tolerance_minutes", 15)
	}

	now := time.Now().In(loc)
	workDate := ComputeWorkDate(now, loc, cutoffHour).Format("2006-01-02")

	todayIn, todayOut, err := s.GetMyToday(ctx, principal)
	if err != nil {
		return nil, err
	}

	hasFace, err := s.hasEnrolledFace(ctx, empID)
	if err != nil {
		return nil, err
	}

	offices, err := s.getActiveOffices(ctx)
	if err != nil {
		return nil, err
	}

	officeDTOs := make([]OfficeSummaryDTO, 0, len(offices))
	for _, o := range offices {
		officeDTOs = append(officeDTOs, OfficeSummaryDTO{
			ID:           o.ID,
			Name:         o.Name,
			Latitude:     o.Latitude,
			Longitude:    o.Longitude,
			RadiusMeters: float64(o.RadiusMeter),
		})
	}

	canCheckIn := todayIn == nil
	canCheckOut := todayIn != nil && todayOut == nil

	return &AttendanceContextResponse{
		ServerTime:       now,
		WorkDate:         workDate,
		TodayCheckIn:     todayIn,
		TodayCheckOut:    todayOut,
		CanCheckIn:       canCheckIn,
		CanCheckOut:      canCheckOut,
		HasEnrolledFace:  hasFace,
		WorkStartTime:    workStartTime,
		WorkEndTime:      workEndTime,
		LateToleranceMin: lateTolerance,
		EarlyLeaveMin:    earlyTolerance,
		ActiveOffices:    officeDTOs,
	}, nil
}

// GetMyToday returns check-in and check-out records for current employee today (#56).
func (s *Service) GetMyToday(ctx context.Context, principal *rbac.Principal) (*EmployeeAttendanceDTO, *EmployeeAttendanceDTO, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, nil, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}
	empID := *principal.EmployeeID

	tzStr := "Asia/Jakarta"
	if s.settings != nil {
		tzStr = s.settings.GetString(ctx, "company_timezone", "Asia/Jakarta")
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	cutoffHour := 4
	if s.settings != nil {
		cutoffHour = s.settings.GetInt(ctx, "workday_cutoff_hour", 4)
	}

	now := time.Now().In(loc)
	workDate := ComputeWorkDate(now, loc, cutoffHour).Format("2006-01-02")

	const q = `
		SELECT a.id, a.employee_id, a.work_date, a.type, a.status, a.method,
		       a.server_timestamp, a.client_reported_at, a.clock_skew_seconds,
		       a.latitude, a.longitude, a.distance_meter, a.matched_office_id, a.location_is_mocked,
		       a.matched_similarity, a.threshold_used, a.model_version, a.quality_score,
		       a.photo_key, a.photo_purged_at, a.photo_sha256, a.photo_bytes, a.photo_mime,
		       a.liveness_passed, a.liveness_supported,
		       a.fallback_reason, a.fallback_note,
		       a.reviewed_by, a.reviewed_at, a.review_notes, a.idempotency_key,
		       a.created_at, a.updated_at,
		       e.full_name, e.employee_number, e.department,
		       o.name as matched_office_name
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		LEFT JOIN office_locations o ON o.id = a.matched_office_id
		WHERE a.employee_id = $1 AND a.work_date = $2 AND a.status <> 'rejected'
		ORDER BY a.server_timestamp ASC
	`
	rows, err := s.db.Query(ctx, q, empID, workDate)
	if err != nil {
		return nil, nil, fmt.Errorf("querying today's attendance: %w", err)
	}
	defer rows.Close()

	var todayIn, todayOut *EmployeeAttendanceDTO
	for rows.Next() {
		att, err := s.scanAttendanceRow(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scanning attendance row: %w", err)
		}
		dto := s.mapToEmployeeDTO(att, loc)
		if att.Type == ClockTypeCheckIn && todayIn == nil {
			todayIn = dto
		} else if att.Type == ClockTypeCheckOut && todayOut == nil {
			todayOut = dto
		}
	}

	return todayIn, todayOut, nil
}

// GetMyHistory returns paginated attendance history for the logged-in employee (#57).
func (s *Service) GetMyHistory(ctx context.Context, principal *rbac.Principal, filter AttendanceFilter) ([]EmployeeAttendanceDTO, int, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, 0, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}
	empID := *principal.EmployeeID
	filter.EmployeeID = &empID

	tzStr := "Asia/Jakarta"
	if s.settings != nil {
		tzStr = s.settings.GetString(ctx, "company_timezone", "Asia/Jakarta")
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	records, total, err := s.queryAttendanceRecords(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]EmployeeAttendanceDTO, 0, len(records))
	for _, r := range records {
		dtos = append(dtos, *s.mapToEmployeeDTO(&r, loc))
	}

	return dtos, total, nil
}

// GetMyHistoryByID returns a single attendance detail for the logged-in employee (#58).
func (s *Service) GetMyHistoryByID(ctx context.Context, principal *rbac.Principal, id uuid.UUID) (*EmployeeAttendanceDTO, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}
	empID := *principal.EmployeeID

	att, err := s.getAttendanceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if att.EmployeeID != empID {
		return nil, httpx.NewAppError(httpx.CodeForbidden, "access denied: attendance record belongs to another employee")
	}

	tzStr := "Asia/Jakarta"
	if s.settings != nil {
		tzStr = s.settings.GetString(ctx, "company_timezone", "Asia/Jakarta")
	}
	loc, err := time.LoadLocation(tzStr)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	return s.mapToEmployeeDTO(att, loc), nil
}

// GetAdminRecords queries all attendance records with admin/manager filters (#59).
func (s *Service) GetAdminRecords(ctx context.Context, filter AttendanceFilter) ([]AdminAttendanceDTO, int, error) {
	records, total, err := s.queryAttendanceRecords(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]AdminAttendanceDTO, 0, len(records))
	for _, r := range records {
		dtos = append(dtos, *s.mapToAdminDTO(&r))
	}

	return dtos, total, nil
}

// GetAdminRecordByID retrieves complete attendance record with full telemetry (#60).
func (s *Service) GetAdminRecordByID(ctx context.Context, id uuid.UUID) (*AdminAttendanceDTO, error) {
	att, err := s.getAttendanceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.mapToAdminDTO(att), nil
}

// ReviewRecord reviews a pending attendance record (Rule B10 & B11) (#61).
func (s *Service) ReviewRecord(ctx context.Context, principal *rbac.Principal, id uuid.UUID, req ReviewRequest) (*AdminAttendanceDTO, error) {
	if principal == nil {
		return nil, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required")
	}

	att, err := s.getAttendanceByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Rule B10: Only pending_review can be reviewed
	if att.Status != StatusPendingReview {
		return nil, httpx.NewAppError(httpx.CodeAttendanceAlreadyReviewed, fmt.Sprintf("cannot review record with status %q", att.Status))
	}

	// Rule B11: Prevent self-review (An employee cannot review their own record)
	if principal.EmployeeID != nil && *principal.EmployeeID == att.EmployeeID {
		return nil, httpx.NewAppError(httpx.CodeSelfReviewDenied, "you cannot review your own attendance record")
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	var newStatus AttendanceStatus
	switch action {
	case "approve":
		newStatus = StatusApproved
	case "reject":
		newStatus = StatusRejected
	default:
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "action must be either 'approve' or 'reject'")
	}

	note := strings.TrimSpace(req.ReviewNote)
	if note == "" {
		note = strings.TrimSpace(req.ReviewNotes)
	}
	if action == "reject" && req.RejectionReason != nil && *req.RejectionReason != "" {
		if note != "" {
			note = *req.RejectionReason + " | " + note
		} else {
			note = *req.RejectionReason
		}
	}

	if action == "reject" {
		if len(note) < 3 || len(note) > 500 {
			return nil, httpx.NewAppError(httpx.CodeValidationError, "review_note is required when rejecting (3-500 characters)")
		}
	}

	const updateQ = `
		UPDATE attendances
		SET status = $1, reviewed_by = $2, reviewed_at = NOW(), review_notes = $3, updated_at = NOW()
		WHERE id = $4
	`
	_, err = s.db.Exec(ctx, updateQ, string(newStatus), principal.UserID, note, id)
	if err != nil {
		return nil, fmt.Errorf("updating attendance review: %w", err)
	}

	return s.GetAdminRecordByID(ctx, id)
}

// BulkReviewRecords reviews multiple pending attendance records in batch (#64).
func (s *Service) BulkReviewRecords(ctx context.Context, principal *rbac.Principal, req BulkReviewRequest) (*BulkReviewResponse, error) {
	if principal == nil {
		return nil, httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required")
	}

	items := req.Items
	if len(items) == 0 && len(req.IDs) > 0 {
		for _, id := range req.IDs {
			items = append(items, BulkReviewItemRequest{
				ID:         id,
				Action:     req.Action,
				ReviewNote: req.ReviewNotes,
			})
		}
	}

	if len(items) == 0 {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "items list cannot be empty")
	}
	if len(items) > 50 {
		return nil, httpx.NewAppError(httpx.CodeBadRequest, "maximum 50 items per bulk review batch")
	}

	resp := &BulkReviewResponse{
		Processed: len(items),
		Results:   make([]BulkReviewResultItem, 0, len(items)),
	}

	for _, item := range items {
		resItem := BulkReviewResultItem{ID: item.ID}
		singleReq := ReviewRequest{
			Action:     item.Action,
			ReviewNote: item.ReviewNote,
		}
		dto, err := s.ReviewRecord(ctx, principal, item.ID, singleReq)
		if err != nil {
			resp.Failed++
			code := string(httpx.CodeBadRequest)
			msg := err.Error()
			if appErr, ok := err.(*httpx.AppError); ok {
				code = string(appErr.Code)
				msg = appErr.Message
			}
			resItem.Error = &BulkReviewError{
				Code:    code,
				Message: msg,
			}
		} else {
			resp.Succeeded++
			resItem.Status = &dto.Status
		}
		resp.Results = append(resp.Results, resItem)
	}

	return resp, nil
}

// GetAttendancePhoto opens an attendance photo stream, enforcing data retention policies and ownership (#59).
func (s *Service) GetAttendancePhoto(ctx context.Context, principal *rbac.Principal, id uuid.UUID) (io.ReadCloser, string, error) {
	if principal == nil {
		return nil, "", httpx.NewAppError(httpx.CodeUnauthenticated, "authentication required")
	}

	att, err := s.getAttendanceByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	isAdmin := principal.HasPermission(rbac.PermAttendanceReadAll)
	if !isAdmin {
		if principal.EmployeeID == nil || *principal.EmployeeID != att.EmployeeID {
			return nil, "", httpx.NewAppError(httpx.CodeNotFound, "attendance record not found")
		}
	}

	// Rule: If purged by retention worker, return ErrPhotoPurged -> HTTP 410 Gone (REV-ERR-04)
	if att.PhotoPurgedAt != nil {
		return nil, "", ErrPhotoPurged
	}

	// If photo key is set and store is available, read from storage
	if att.PhotoKey != nil && *att.PhotoKey != "" && s.store != nil {
		rdr, err := s.store.Get(ctx, *att.PhotoKey)
		if err == nil {
			mime := "image/jpeg"
			if att.PhotoMime != nil && *att.PhotoMime != "" {
				mime = *att.PhotoMime
			}
			return rdr, mime, nil
		}
	}

	return nil, "", httpx.NewAppError(httpx.CodeNotFound, "attendance photo not available")
}

// GetTeamRecords queries attendance records for the team of the requesting manager (#63).
func (s *Service) GetTeamRecords(ctx context.Context, principal *rbac.Principal, filter AttendanceFilter) ([]AdminAttendanceDTO, int, error) {
	if principal == nil || principal.EmployeeID == nil {
		return nil, 0, httpx.NewAppError(httpx.CodeBadRequest, "user is not associated with an employee profile")
	}

	// Resolve manager's department
	var dept *string
	err := s.db.QueryRow(ctx, "SELECT department FROM employees WHERE id = $1", *principal.EmployeeID).Scan(&dept)
	if err != nil {
		return nil, 0, fmt.Errorf("resolving manager department: %w", err)
	}

	if dept == nil || *dept == "" {
		return []AdminAttendanceDTO{}, 0, nil
	}

	filter.Department = *dept
	return s.GetAdminRecords(ctx, filter)
}

// GetStats returns summary aggregate metrics for attendance in a date range (#64).
func (s *Service) GetStats(ctx context.Context, startDate, endDate string) (*AttendanceStatsResponse, error) {
	const q = `
		SELECT
			count(*) as total,
			count(*) FILTER (WHERE status = 'approved') as approved,
			count(*) FILTER (WHERE status = 'pending_review') as pending,
			count(*) FILTER (WHERE status = 'rejected') as rejected,
			count(*) FILTER (WHERE type = 'check_in') as check_ins,
			count(*) FILTER (WHERE type = 'check_out') as check_outs,
			count(*) FILTER (WHERE method <> 'face_verified') as fallbacks
		FROM attendances
		WHERE ($1 = '' OR work_date >= $1::date)
		  AND ($2 = '' OR work_date <= $2::date)
	`
	var (
		stats                                                    AttendanceStatsResponse
		total, approved, pending, rejected, ins, outs, fallbacks int
	)
	err := s.db.QueryRow(ctx, q, startDate, endDate).Scan(
		&total, &approved, &pending, &rejected, &ins, &outs, &fallbacks,
	)
	if err != nil {
		return nil, fmt.Errorf("aggregating attendance stats: %w", err)
	}

	stats.StartDate = startDate
	stats.EndDate = endDate
	stats.TotalRecords = total
	stats.ApprovedCount = approved
	stats.PendingCount = pending
	stats.RejectedCount = rejected
	stats.CheckInCount = ins
	stats.CheckOutCount = outs
	stats.FallbackCount = fallbacks

	return &stats, nil
}

// GetAttempts returns audit telemetry attempts (#65).
func (s *Service) GetAttempts(ctx context.Context, filter AttemptFilter) ([]AttendanceAttempt, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var where []string
	var args []any
	idx := 1

	if filter.EmployeeID != nil {
		where = append(where, fmt.Sprintf("a.employee_id = $%d", idx))
		args = append(args, *filter.EmployeeID)
		idx++
	}
	if filter.Outcome != "" {
		where = append(where, fmt.Sprintf("a.outcome = $%d", idx))
		args = append(args, filter.Outcome)
		idx++
	}
	if filter.Type != "" {
		where = append(where, fmt.Sprintf("a.type = $%d", idx))
		args = append(args, filter.Type)
		idx++
	}
	if filter.StartDate != "" {
		where = append(where, fmt.Sprintf("a.work_date >= $%d::date", idx))
		args = append(args, filter.StartDate)
		idx++
	}
	if filter.EndDate != "" {
		where = append(where, fmt.Sprintf("a.work_date <= $%d::date", idx))
		args = append(args, filter.EndDate)
		idx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	countQ := fmt.Sprintf("SELECT count(*) FROM attendance_attempts a %s", whereClause)
	var total int
	if err := s.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting attendance attempts: %w", err)
	}

	queryQ := fmt.Sprintf(`
		SELECT a.id, a.employee_id, a.type, a.server_timestamp, a.work_date, a.outcome,
		       a.matched_similarity, a.threshold_used, a.model_version, a.quality_score,
		       a.hints, a.geofence_status, a.distance_meter, a.allow_fallback,
		       a.attendance_id, a.failure_reason, a.request_id, a.ip, a.user_agent, a.created_at,
		       e.full_name, e.employee_number
		FROM attendance_attempts a
		JOIN employees e ON e.id = a.employee_id
		%s
		ORDER BY a.server_timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, idx, idx+1)
	args = append(args, perPage, offset)

	rows, err := s.db.Query(ctx, queryQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying attendance attempts: %w", err)
	}
	defer rows.Close()

	var attempts []AttendanceAttempt
	for rows.Next() {
		var (
			att       AttendanceAttempt
			clockType string
			workDate  time.Time
		)
		if err := rows.Scan(
			&att.ID, &att.EmployeeID, &clockType, &att.ServerTimestamp, &workDate, &att.Outcome,
			&att.MatchedSimilarity, &att.ThresholdUsed, &att.ModelVersion, &att.QualityScore,
			&att.Hints, &att.GeofenceStatus, &att.DistanceMeter, &att.AllowFallback,
			&att.AttendanceID, &att.FailureReason, &att.RequestID, &att.IP, &att.UserAgent, &att.CreatedAt,
			&att.EmployeeName, &att.EmployeeNumber,
		); err != nil {
			return nil, 0, fmt.Errorf("scanning attendance attempt: %w", err)
		}
		att.Type = ClockType(clockType)
		att.WorkDate = workDate.Format("2006-01-02")
		attempts = append(attempts, att)
	}

	if attempts == nil {
		attempts = []AttendanceAttempt{}
	}

	return attempts, total, nil
}

func (s *Service) getAttendanceByID(ctx context.Context, id uuid.UUID) (*Attendance, error) {
	const q = `
		SELECT a.id, a.employee_id, a.work_date, a.type, a.status, a.method,
		       a.server_timestamp, a.client_reported_at, a.clock_skew_seconds,
		       a.latitude, a.longitude, a.distance_meter, a.matched_office_id, a.location_is_mocked,
		       a.matched_similarity, a.threshold_used, a.model_version, a.quality_score,
		       a.photo_key, a.photo_purged_at, a.photo_sha256, a.photo_bytes, a.photo_mime,
		       a.liveness_passed, a.liveness_supported,
		       a.fallback_reason, a.fallback_note,
		       a.reviewed_by, a.reviewed_at, a.review_notes, a.idempotency_key,
		       a.created_at, a.updated_at,
		       e.full_name, e.employee_number, e.department,
		       o.name as matched_office_name
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		LEFT JOIN office_locations o ON o.id = a.matched_office_id
		WHERE a.id = $1
		LIMIT 1
	`
	row := s.db.QueryRow(ctx, q, id)
	att, err := s.scanAttendanceRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "attendance record not found")
		}
		return nil, fmt.Errorf("fetching attendance by id: %w", err)
	}
	return att, nil
}

func (s *Service) queryAttendanceRecords(ctx context.Context, filter AttendanceFilter) ([]Attendance, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var where []string
	var args []any
	idx := 1

	if filter.EmployeeID != nil {
		where = append(where, fmt.Sprintf("a.employee_id = $%d", idx))
		args = append(args, *filter.EmployeeID)
		idx++
	}
	if filter.Department != "" {
		where = append(where, fmt.Sprintf("e.department = $%d", idx))
		args = append(args, filter.Department)
		idx++
	}
	if filter.StartDate != "" {
		where = append(where, fmt.Sprintf("a.work_date >= $%d::date", idx))
		args = append(args, filter.StartDate)
		idx++
	}
	if filter.EndDate != "" {
		where = append(where, fmt.Sprintf("a.work_date <= $%d::date", idx))
		args = append(args, filter.EndDate)
		idx++
	}
	if filter.WorkDate != "" {
		where = append(where, fmt.Sprintf("a.work_date = $%d::date", idx))
		args = append(args, filter.WorkDate)
		idx++
	}
	if filter.Status != "" {
		where = append(where, fmt.Sprintf("a.status = $%d", idx))
		args = append(args, filter.Status)
		idx++
	}
	if filter.Type != "" {
		where = append(where, fmt.Sprintf("a.type = $%d", idx))
		args = append(args, filter.Type)
		idx++
	}
	if filter.Method != "" {
		where = append(where, fmt.Sprintf("a.method = $%d", idx))
		args = append(args, filter.Method)
		idx++
	}
	if filter.IsPending {
		where = append(where, "a.status = 'pending_review'")
	}
	if filter.OnlyFallback {
		where = append(where, "a.method <> 'face_verified'")
	}
	if filter.Search != "" {
		where = append(where, fmt.Sprintf("(e.full_name ILIKE $%d OR e.employee_number ILIKE $%d)", idx, idx))
		args = append(args, "%"+filter.Search+"%")
		idx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	countQ := fmt.Sprintf(`
		SELECT count(*)
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		%s
	`, whereClause)
	var total int
	if err := s.db.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting attendances: %w", err)
	}

	orderClause := "ORDER BY a.server_timestamp DESC"
	if filter.IsPending {
		orderClause = "ORDER BY a.server_timestamp ASC"
	}

	queryQ := fmt.Sprintf(`
		SELECT a.id, a.employee_id, a.work_date, a.type, a.status, a.method,
		       a.server_timestamp, a.client_reported_at, a.clock_skew_seconds,
		       a.latitude, a.longitude, a.distance_meter, a.matched_office_id, a.location_is_mocked,
		       a.matched_similarity, a.threshold_used, a.model_version, a.quality_score,
		       a.photo_key, a.photo_purged_at, a.photo_sha256, a.photo_bytes, a.photo_mime,
		       a.liveness_passed, a.liveness_supported,
		       a.fallback_reason, a.fallback_note,
		       a.reviewed_by, a.reviewed_at, a.review_notes, a.idempotency_key,
		       a.created_at, a.updated_at,
		       e.full_name, e.employee_number, e.department,
		       o.name as matched_office_name
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		LEFT JOIN office_locations o ON o.id = a.matched_office_id
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, idx, idx+1)
	args = append(args, perPage, offset)

	rows, err := s.db.Query(ctx, queryQ, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying attendances: %w", err)
	}
	defer rows.Close()

	var records []Attendance
	for rows.Next() {
		att, err := s.scanAttendanceRow(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning attendance record: %w", err)
		}
		records = append(records, *att)
	}

	if records == nil {
		records = []Attendance{}
	}

	return records, total, nil
}

func (s *Service) getByEmployeeWorkDateAndIdempotency(ctx context.Context, empID uuid.UUID, workDate, key string) (*Attendance, error) {
	const q = `
		SELECT a.id, a.employee_id, a.work_date, a.type, a.status, a.method,
		       a.server_timestamp, a.client_reported_at, a.clock_skew_seconds,
		       a.latitude, a.longitude, a.distance_meter, a.matched_office_id, a.location_is_mocked,
		       a.matched_similarity, a.threshold_used, a.model_version, a.quality_score,
		       a.photo_key, a.photo_purged_at, a.photo_sha256, a.photo_bytes, a.photo_mime,
		       a.liveness_passed, a.liveness_supported,
		       a.fallback_reason, a.fallback_note,
		       a.reviewed_by, a.reviewed_at, a.review_notes, a.idempotency_key,
		       a.created_at, a.updated_at,
		       e.full_name, e.employee_number, e.department,
		       o.name as matched_office_name
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		LEFT JOIN office_locations o ON o.id = a.matched_office_id
		WHERE a.employee_id = $1 AND a.work_date = $2 AND a.idempotency_key = $3
		LIMIT 1
	`
	row := s.db.QueryRow(ctx, q, empID, workDate, key)
	return s.scanAttendanceRow(row)
}

func (s *Service) hasEnrolledFace(ctx context.Context, empID uuid.UUID) (bool, error) {
	// Check face_references first
	var refCount int
	err := s.db.QueryRow(ctx, "SELECT count(*) FROM face_references WHERE employee_id = $1 AND is_active = true", empID).Scan(&refCount)
	if err == nil && refCount > 0 {
		return true, nil
	}

	// Check employees.face_embedding
	var hasEmb bool
	err = s.db.QueryRow(ctx, "SELECT (face_embedding IS NOT NULL) FROM employees WHERE id = $1", empID).Scan(&hasEmb)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return hasEmb, nil
}

func (s *Service) matchBestSimilarity(ctx context.Context, empID uuid.UUID, vectorStr string) (float64, error) {
	const refQ = `
		SELECT 1 - (embedding <=> $1::vector) AS similarity
		FROM face_references
		WHERE employee_id = $2 AND is_active = true
		ORDER BY embedding <=> $1::vector ASC
		LIMIT 1
	`
	var sim float64
	err := s.db.QueryRow(ctx, refQ, vectorStr, empID).Scan(&sim)
	if err == nil {
		return sim, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}

	// Fallback to employees table
	const empQ = `
		SELECT 1 - (face_embedding <=> $1::vector) AS similarity
		FROM employees
		WHERE id = $2 AND face_embedding IS NOT NULL
	`
	err = s.db.QueryRow(ctx, empQ, vectorStr, empID).Scan(&sim)
	if err != nil {
		return 0, err
	}
	return sim, nil
}

func (s *Service) getActiveOffices(ctx context.Context) ([]geo.OfficeLocation, error) {
	const q = `
		SELECT id, name, latitude, longitude, radius_meter, is_active
		FROM office_locations
		WHERE is_active = true AND deleted_at IS NULL
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var offices []geo.OfficeLocation
	for rows.Next() {
		var o geo.OfficeLocation
		if err := rows.Scan(&o.ID, &o.Name, &o.Latitude, &o.Longitude, &o.RadiusMeter, &o.IsActive); err != nil {
			return nil, err
		}
		offices = append(offices, o)
	}
	return offices, nil
}

func (s *Service) mapToEmployeeDTO(att *Attendance, loc *time.Location) *EmployeeAttendanceDTO {
	startTimeStr := "08:00"
	lateTolerance := 15
	endTimeStr := "17:00"
	earlyTolerance := 15
	if s.settings != nil {
		ctx := context.Background()
		startTimeStr = s.settings.GetString(ctx, "attendance_work_start_time", "08:00")
		lateTolerance = s.settings.GetInt(ctx, "attendance_late_tolerance_minutes", 15)
		endTimeStr = s.settings.GetString(ctx, "attendance_work_end_time", "17:00")
		earlyTolerance = s.settings.GetInt(ctx, "attendance_early_leave_tolerance_minutes", 15)
	}

	dto := &EmployeeAttendanceDTO{
		ID:              att.ID,
		EmployeeID:      att.EmployeeID,
		WorkDate:        att.WorkDate,
		Type:            att.Type,
		Status:          att.Status,
		Method:          att.Method,
		ServerTimestamp: att.ServerTimestamp,
		OfficeName:      att.MatchedOfficeName,
		InRadius:        att.DistanceMeter != nil,
		FallbackUsed:    att.Method != MethodFaceVerified,
		FallbackReason:  att.FallbackReason,
		ReviewNotes:     att.ReviewNotes,
		CreatedAt:       att.CreatedAt,
	}

	if att.Type == ClockTypeCheckIn {
		dto.IsLate, dto.LateMinutes = EvaluateLate(att.ServerTimestamp, loc, startTimeStr, lateTolerance)
	} else if att.Type == ClockTypeCheckOut {
		dto.IsEarlyLeave, dto.EarlyMinutes = EvaluateEarlyLeave(att.ServerTimestamp, loc, endTimeStr, earlyTolerance)
	}

	return dto
}

func (s *Service) mapToAdminDTO(att *Attendance) *AdminAttendanceDTO {
	var waitingHours *float64
	if att.Status == StatusPendingReview {
		wh := math.Round(time.Since(att.ServerTimestamp).Hours()*10) / 10
		waitingHours = &wh
	}

	return &AdminAttendanceDTO{
		ID:                att.ID,
		EmployeeID:        att.EmployeeID,
		EmployeeName:      att.EmployeeName,
		EmployeeNumber:    att.EmployeeNumber,
		Department:        att.Department,
		WorkDate:          att.WorkDate,
		Type:              att.Type,
		Status:            att.Status,
		Method:            att.Method,
		ServerTimestamp:   att.ServerTimestamp,
		ClientReportedAt:  att.ClientReportedAt,
		ClockSkewSeconds:  att.ClockSkewSeconds,
		Latitude:          att.Latitude,
		Longitude:         att.Longitude,
		DistanceMeter:     att.DistanceMeter,
		MatchedOfficeID:   att.MatchedOfficeID,
		MatchedOfficeName: att.MatchedOfficeName,
		LocationIsMocked:  att.LocationIsMocked,
		MatchedSimilarity: att.MatchedSimilarity,
		ThresholdUsed:     att.ThresholdUsed,
		ModelVersion:      att.ModelVersion,
		QualityScore:      att.QualityScore,
		PhotoKey:          att.PhotoKey,
		PhotoPurgedAt:     att.PhotoPurgedAt,
		PhotoSHA256:       att.PhotoSHA256,
		LivenessPassed:    att.LivenessPassed,
		LivenessSupported: att.LivenessSupported,
		FallbackReason:    att.FallbackReason,
		FallbackNote:      att.FallbackNote,
		ReviewedBy:        att.ReviewedBy,
		ReviewedByName:    att.ReviewedByName,
		ReviewedAt:        att.ReviewedAt,
		ReviewNotes:       att.ReviewNotes,
		WaitingHours:      waitingHours,
		IdempotencyKey:    att.IdempotencyKey,
		CreatedAt:         att.CreatedAt,
		UpdatedAt:         att.UpdatedAt,
	}
}

func (s *Service) scanAttendanceRow(row pgx.Row) (*Attendance, error) {
	var (
		att                       Attendance
		clockType, status, method string
		workDate                  time.Time
	)

	err := row.Scan(
		&att.ID, &att.EmployeeID, &workDate, &clockType, &status, &method,
		&att.ServerTimestamp, &att.ClientReportedAt, &att.ClockSkewSeconds,
		&att.Latitude, &att.Longitude, &att.DistanceMeter, &att.MatchedOfficeID, &att.LocationIsMocked,
		&att.MatchedSimilarity, &att.ThresholdUsed, &att.ModelVersion, &att.QualityScore,
		&att.PhotoKey, &att.PhotoPurgedAt, &att.PhotoSHA256, &att.PhotoBytes, &att.PhotoMime,
		&att.LivenessPassed, &att.LivenessSupported,
		&att.FallbackReason, &att.FallbackNote,
		&att.ReviewedBy, &att.ReviewedAt, &att.ReviewNotes, &att.IdempotencyKey,
		&att.CreatedAt, &att.UpdatedAt,
		&att.EmployeeName, &att.EmployeeNumber, &att.Department,
		&att.MatchedOfficeName,
	)
	if err != nil {
		return nil, err
	}

	att.WorkDate = workDate.Format("2006-01-02")
	att.Type = ClockType(clockType)
	att.Status = AttendanceStatus(status)
	att.Method = AttendanceMethod(method)

	return &att, nil
}
