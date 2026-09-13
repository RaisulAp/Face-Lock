package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AppSetting represents an application setting entry.
type AppSetting struct {
	Key         string     `json:"key"`
	Value       any        `json:"value"`
	ValueType   string     `json:"value_type"`
	Description *string    `json:"description,omitempty"`
	IsPublic    bool       `json:"is_public"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Service provides access to reading and writing app_settings.
type Service struct {
	db *pgxpool.Pool
}

// NewService creates a settings service.
func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// GetAll returns all settings.
func (s *Service) GetAll(ctx context.Context) ([]AppSetting, error) {
	const q = `
		SELECT key, value, value_type, description, is_public, updated_by, updated_at
		FROM app_settings
		ORDER BY key ASC
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying app_settings: %w", err)
	}
	defer rows.Close()

	var list []AppSetting
	for rows.Next() {
		var (
			item   AppSetting
			rawVal []byte
		)
		if err := rows.Scan(
			&item.Key,
			&rawVal,
			&item.ValueType,
			&item.Description,
			&item.IsPublic,
			&item.UpdatedBy,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning app_setting: %w", err)
		}

		if err := json.Unmarshal(rawVal, &item.Value); err != nil {
			item.Value = string(rawVal)
		}

		list = append(list, item)
	}

	if list == nil {
		list = []AppSetting{}
	}

	return list, nil
}

// GetPublic returns only settings marked as is_public = true.
func (s *Service) GetPublic(ctx context.Context) ([]AppSetting, error) {
	const q = `
		SELECT key, value, value_type, description, is_public, updated_by, updated_at
		FROM app_settings
		WHERE is_public = true
		ORDER BY key ASC
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying public app_settings: %w", err)
	}
	defer rows.Close()

	var list []AppSetting
	for rows.Next() {
		var (
			item   AppSetting
			rawVal []byte
		)
		if err := rows.Scan(
			&item.Key,
			&rawVal,
			&item.ValueType,
			&item.Description,
			&item.IsPublic,
			&item.UpdatedBy,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning public app_setting: %w", err)
		}

		if err := json.Unmarshal(rawVal, &item.Value); err != nil {
			item.Value = string(rawVal)
		}

		list = append(list, item)
	}

	if list == nil {
		list = []AppSetting{}
	}

	return list, nil
}

// GetInt retrieves a numeric setting as integer with a default fallback.
func (s *Service) GetInt(ctx context.Context, key string, defaultVal int) int {
	if s == nil || s.db == nil {
		return defaultVal
	}
	const q = `SELECT value FROM app_settings WHERE key = $1`
	var rawVal []byte
	if err := s.db.QueryRow(ctx, q, key).Scan(&rawVal); err != nil {
		return defaultVal
	}

	var numVal float64
	if err := json.Unmarshal(rawVal, &numVal); err == nil {
		return int(numVal)
	}

	var strVal string
	if err := json.Unmarshal(rawVal, &strVal); err == nil {
		if i, err := strconv.Atoi(strVal); err == nil {
			return i
		}
	}

	return defaultVal
}

// GetBool retrieves a boolean setting with a default fallback.
func (s *Service) GetBool(ctx context.Context, key string, defaultVal bool) bool {
	if s == nil || s.db == nil {
		return defaultVal
	}
	const q = `SELECT value FROM app_settings WHERE key = $1`
	var rawVal []byte
	if err := s.db.QueryRow(ctx, q, key).Scan(&rawVal); err != nil {
		return defaultVal
	}

	var boolVal bool
	if err := json.Unmarshal(rawVal, &boolVal); err == nil {
		return boolVal
	}

	var strVal string
	if err := json.Unmarshal(rawVal, &strVal); err == nil {
		if b, err := strconv.ParseBool(strVal); err == nil {
			return b
		}
	}

	return defaultVal
}

// GetFloat retrieves a floating-point numeric setting with a default fallback.
func (s *Service) GetFloat(ctx context.Context, key string, defaultVal float64) float64 {
	if s == nil || s.db == nil {
		return defaultVal
	}
	const q = `SELECT value FROM app_settings WHERE key = $1`
	var rawVal []byte
	if err := s.db.QueryRow(ctx, q, key).Scan(&rawVal); err != nil {
		return defaultVal
	}

	var numVal float64
	if err := json.Unmarshal(rawVal, &numVal); err == nil {
		return numVal
	}

	var strVal string
	if err := json.Unmarshal(rawVal, &strVal); err == nil {
		if f, err := strconv.ParseFloat(strVal, 64); err == nil {
			return f
		}
	}

	return defaultVal
}

// GetString retrieves a string setting with a default fallback.
func (s *Service) GetString(ctx context.Context, key string, defaultVal string) string {
	if s == nil || s.db == nil {
		return defaultVal
	}
	const q = `SELECT value FROM app_settings WHERE key = $1`
	var rawVal []byte
	if err := s.db.QueryRow(ctx, q, key).Scan(&rawVal); err != nil {
		return defaultVal
	}

	var strVal string
	if err := json.Unmarshal(rawVal, &strVal); err == nil {
		return strVal
	}

	return string(rawVal)
}

// UpdateBatch updates a list of settings after validating types.
func (s *Service) UpdateBatch(ctx context.Context, actorID *uuid.UUID, updates map[string]any) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for k, v := range updates {
		// Get expected value_type
		const typeQ = `SELECT value_type FROM app_settings WHERE key = $1`
		var valType string
		if err := tx.QueryRow(ctx, typeQ, k).Scan(&valType); err != nil {
			return httpx.NewAppError(httpx.CodeNotFound, fmt.Sprintf("setting %q not found", k))
		}

		// Validate value type
		switch valType {
		case "number":
			switch v.(type) {
			case float64, float32, int, int32, int64:
			default:
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a number", k))
			}
		case "boolean":
			if _, ok := v.(bool); !ok {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a boolean", k))
			}
		case "string":
			if _, ok := v.(string); !ok {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a string", k))
			}
		case "json":
			// Any valid JSON type is fine
		}

		valJSON, err := json.Marshal(v)
		if err != nil {
			return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("invalid json value for %q", k))
		}

		const updateQ = `
			UPDATE app_settings
			SET value = $1, updated_by = $2, updated_at = NOW()
			WHERE key = $3
		`
		if _, err := tx.Exec(ctx, updateQ, valJSON, actorID, k); err != nil {
			return fmt.Errorf("updating setting %q: %w", k, err)
		}
	}

	return tx.Commit(ctx)
}

// GetByKey returns a single setting by key.
func (s *Service) GetByKey(ctx context.Context, key string) (*AppSetting, error) {
	const q = `
		SELECT key, value, value_type, description, is_public, updated_by, updated_at
		FROM app_settings
		WHERE key = $1
	`
	var (
		item   AppSetting
		rawVal []byte
	)
	err := s.db.QueryRow(ctx, q, key).Scan(
		&item.Key,
		&rawVal,
		&item.ValueType,
		&item.Description,
		&item.IsPublic,
		&item.UpdatedBy,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, httpx.NewAppError(httpx.CodeNotFound, fmt.Sprintf("setting %q not found", key))
	}

	if err := json.Unmarshal(rawVal, &item.Value); err != nil {
		item.Value = string(rawVal)
	}

	return &item, nil
}

// UpdateSingle validates and updates a single setting by key, returning old and new setting.
func (s *Service) UpdateSingle(ctx context.Context, actorID *uuid.UUID, key string, val any) (*AppSetting, *AppSetting, error) {
	old, err := s.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	// Validate value against value_type and per-key rules
	if err := validateSettingValue(key, old.ValueType, val); err != nil {
		return nil, nil, err
	}

	valJSON, err := json.Marshal(val)
	if err != nil {
		return nil, nil, httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("invalid json value for %q", key))
	}

	const updateQ = `
		UPDATE app_settings
		SET value = $1, updated_by = $2, updated_at = NOW()
		WHERE key = $3
	`
	if _, err := s.db.Exec(ctx, updateQ, valJSON, actorID, key); err != nil {
		return nil, nil, fmt.Errorf("updating setting %q: %w", key, err)
	}

	updated, err := s.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	return old, updated, nil
}

func validateSettingValue(key, valType string, v any) error {
	switch valType {
	case "number":
		var num float64
		switch n := v.(type) {
		case float64:
			num = n
		case float32:
			num = float64(n)
		case int:
			num = float64(n)
		case int32:
			num = float64(n)
		case int64:
			num = float64(n)
		default:
			return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a number", key))
		}

		switch key {
		case "face.similarity_threshold":
			if num < 0.0 || num > 1.0 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.similarity_threshold must be between 0.0 and 1.0")
			}
		case "face.min_reference_photos":
			if num < 1 || num > 5 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.min_reference_photos must be between 1 and 5")
			}
		case "face.max_reference_photos":
			if num < 1 || num > 10 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.max_reference_photos must be between 1 and 10")
			}
		case "face.enrollment_session_ttl_minutes":
			if num < 5 || num > 60 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.enrollment_session_ttl_minutes must be between 5 and 60")
			}
		case "face.min_quality_score":
			if num < 0.0 || num > 1.0 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.min_quality_score must be between 0.0 and 1.0")
			}
		case "face.duplicate_threshold":
			if num < 0.0 || num > 1.0 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.duplicate_threshold must be between 0.0 and 1.0")
			}
		case "face.retention_days_after_resign":
			if num < 1 || num > 3650 {
				return httpx.NewAppError(httpx.CodeValidationError, "face.retention_days_after_resign must be between 1 and 3650")
			}
		case "attendance.max_distance_meter", "attendance_geofence_radius_meters":
			if num < 10 || num > 10000 {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("%s must be between 10 and 10000", key))
			}
		case "attendance.late_tolerance_minutes", "attendance_late_tolerance_minutes":
			if num < 0 || num > 240 {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("%s must be between 0 and 240", key))
			}
		case "attendance_early_leave_tolerance_minutes", "attendance_overtime_minimum_minutes", "attendance_checkout_min_interval_minutes":
			if num < 0 || num > 240 {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("%s must be between 0 and 240", key))
			}
		case "attendance_photo_retention_days":
			if num < 1 || num > 3650 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance_photo_retention_days must be between 1 and 3650")
			}
		case "attendance_max_skew_seconds":
			if num < 10 || num > 3600 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance_max_skew_seconds must be between 10 and 3600")
			}
		case "attendance_fallback_max_per_month":
			if num < 0 || num > 31 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance_fallback_max_per_month must be between 0 and 31")
			}
		case "attendance_max_failed_attempts":
			if num < 1 || num > 20 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance_max_failed_attempts must be between 1 and 20")
			}
		case "attendance_failed_attempt_window_seconds":
			if num < 10 || num > 3600 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance_failed_attempt_window_seconds must be between 10 and 3600")
			}
		case "workday_cutoff_hour":
			if num < 0 || num > 23 {
				return httpx.NewAppError(httpx.CodeValidationError, "workday_cutoff_hour must be between 0 and 23")
			}
		case "security.max_failed_login":
			if num < 3 || num > 10 {
				return httpx.NewAppError(httpx.CodeValidationError, "security.max_failed_login must be between 3 and 10")
			}
		case "security.lockout_minutes":
			if num < 1 || num > 60 {
				return httpx.NewAppError(httpx.CodeValidationError, "security.lockout_minutes must be between 1 and 60")
			}
		case "security.require_password_change_days":
			if num < 0 || num > 365 {
				return httpx.NewAppError(httpx.CodeValidationError, "security.require_password_change_days must be between 0 and 365")
			}
		case "storage.retention_days":
			if num < 30 || num > 3650 {
				return httpx.NewAppError(httpx.CodeValidationError, "storage.retention_days must be between 30 and 3650")
			}
		case "attendance.export_max_rows":
			if num < 100 || num > 1000000 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance.export_max_rows must be between 100 and 1000000")
			}
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a boolean", key))
		}
	case "string":
		str, ok := v.(string)
		if !ok {
			return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q expects a string", key))
		}

		switch key {
		case "attendance.work_start_time", "attendance.work_end_time",
			"attendance_work_start_time", "attendance_work_end_time", "attendance_auto_checkout_time":
			if _, err := time.Parse("15:04", str); err != nil {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q must be in HH:MM format", key))
			}
		case "company_timezone":
			if _, err := time.LoadLocation(str); err != nil {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q must be a valid IANA timezone (e.g. Asia/Jakarta)", key))
			}
		case "face.liveness_strictness":
			if str != "low" && str != "standard" && str != "high" {
				return httpx.NewAppError(httpx.CodeValidationError, "face.liveness_strictness must be one of: low, standard, high")
			}
		}
	}
	return nil
}

// FaceQualityThresholdItem represents a single face quality threshold compared with active inference.
type FaceQualityThresholdItem struct {
	Key                  string   `json:"key"`
	AppSettingsValue     float64  `json:"app_settings_value"`
	InferenceActiveValue *float64 `json:"inference_active_value"`
	InSync               *bool    `json:"in_sync"`
}

// FaceQualityStatusResponse represents the payload of GET /api/v1/settings/face-quality-status (#73).
type FaceQualityStatusResponse struct {
	CheckedAt          *time.Time                 `json:"checked_at"`
	InSync             *bool                      `json:"in_sync"`
	InferenceAvailable bool                       `json:"inference_available"`
	InferenceReady     bool                       `json:"inference_ready"`
	ModelName          string                     `json:"model_name,omitempty"`
	Thresholds         []FaceQualityThresholdItem `json:"thresholds"`
}

// Handler handles HTTP requests for settings.
type Handler struct {
	svc       *Service
	audit     *audit.Recorder
	inference inference.FaceEngine
}

// NewHandler creates a settings handler with optional inference engine.
func NewHandler(svc *Service, audit *audit.Recorder, inf ...inference.FaceEngine) *Handler {
	h := &Handler{svc: svc, audit: audit}
	if len(inf) > 0 && inf[0] != nil {
		h.inference = inf[0]
	}
	return h
}

// SetInference configures the inference engine client for live drift checks.
func (h *Handler) SetInference(inf inference.FaceEngine) {
	h.inference = inf
}

// FaceQualityStatus handles GET /api/v1/settings/face-quality-status (#73 - REV-EP-11 / K-02).
func (h *Handler) FaceQualityStatus(w http.ResponseWriter, r *http.Request) {
	keys := []string{
		"face.min_det_score",
		"face.min_blur_var",
		"face.min_brightness",
		"face.max_brightness",
		"face.min_face_ratio",
		"face.max_abs_yaw",
		"face.max_abs_pitch",
	}

	defaultValues := map[string]float64{
		"face.min_det_score":  0.60,
		"face.min_blur_var":   40.0,
		"face.min_brightness": 55.0,
		"face.max_brightness": 215.0,
		"face.min_face_ratio": 0.18,
		"face.max_abs_yaw":    0.35,
		"face.max_abs_pitch":  0.30,
	}

	var (
		checkedAt     *time.Time
		overallInSync *bool
		thresholds    []FaceQualityThresholdItem
	)

	// Live check with strict 2-second timeout budget
	var readyData *inference.ReadyData
	if h.inference != nil {
		checkCtx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if rd, err := h.inference.Ready(checkCtx); err == nil && rd != nil {
			readyData = rd
		}
	}

	if readyData != nil {
		now := time.Now().UTC()
		checkedAt = &now
		allSync := true

		activeValues := map[string]float64{
			"face.min_det_score":  readyData.QualityThresholds.MinDetScore,
			"face.min_blur_var":   readyData.QualityThresholds.MinBlurVar,
			"face.min_brightness": readyData.QualityThresholds.MinBrightness,
			"face.max_brightness": readyData.QualityThresholds.MaxBrightness,
			"face.min_face_ratio": readyData.QualityThresholds.MinFaceRatio,
			"face.max_abs_yaw":    readyData.QualityThresholds.MaxAbsYaw,
			"face.max_abs_pitch":  readyData.QualityThresholds.MaxAbsPitch,
		}

		for _, k := range keys {
			appVal := h.svc.GetFloat(r.Context(), k, defaultValues[k])
			infVal := activeValues[k]
			inSync := math.Abs(appVal-infVal) < 0.0001
			if !inSync {
				allSync = false
			}

			valCopy := infVal
			syncCopy := inSync
			thresholds = append(thresholds, FaceQualityThresholdItem{
				Key:                  k,
				AppSettingsValue:     appVal,
				InferenceActiveValue: &valCopy,
				InSync:               &syncCopy,
			})
		}
		overallInSync = &allSync
	} else {
		// Inference unreachable or timed out within 2s -> return 200 with null active values
		checkedAt = nil
		overallInSync = nil
		for _, k := range keys {
			appVal := h.svc.GetFloat(r.Context(), k, defaultValues[k])
			thresholds = append(thresholds, FaceQualityThresholdItem{
				Key:                  k,
				AppSettingsValue:     appVal,
				InferenceActiveValue: nil,
				InSync:               nil,
			})
		}
	}

	res := FaceQualityStatusResponse{
		CheckedAt:  checkedAt,
		InSync:     overallInSync,
		Thresholds: thresholds,
	}
	if readyData != nil {
		res.InferenceAvailable = true
		res.InferenceReady = true
		res.ModelName = readyData.ModelName
	}
	httpx.OK(w, res)
}

// List handles GET /api/v1/settings
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, ok := rbac.GetPrincipal(r.Context())
	if ok && p != nil && p.HasPermission("settings.update") {
		list, err := h.svc.GetAll(r.Context())
		if err != nil {
			httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to load settings"))
			return
		}
		httpx.OK(w, list)
		return
	}

	list, err := h.svc.GetPublic(r.Context())
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to load settings"))
		return
	}
	httpx.OK(w, list)
}

// GetByKey handles GET /api/v1/settings/{key}
func (h *Handler) GetByKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	item, err := h.svc.GetByKey(r.Context(), key)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to load setting"))
		return
	}

	p, _ := rbac.GetPrincipal(r.Context())
	if !item.IsPublic && (p == nil || !p.HasPermission("settings.update")) {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, fmt.Sprintf("setting %q not found", key)))
		return
	}

	httpx.OK(w, item)
}

// UpdateSettingRequest payload for updating a single setting.
type UpdateSettingRequest struct {
	Value any `json:"value"`
}

// Update handles PUT /api/v1/settings/{key}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req UpdateSettingRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	p, _ := rbac.GetPrincipal(r.Context())
	var actorID *uuid.UUID
	if p != nil {
		actorID = &p.UserID
	}

	oldSetting, updatedSetting, err := h.svc.UpdateSingle(r.Context(), actorID, key, req.Value)
	if err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update setting"))
		return
	}

	_ = h.audit.RecordFromRequest(r, "settings.updated", "app_settings", &key, map[string]any{
		"old_value": oldSetting.Value,
		"new_value": updatedSetting.Value,
	})

	httpx.OK(w, updatedSetting)
}

// GetAll handles GET /api/v1/settings (legacy alias)
func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	h.List(w, r)
}

// GetPublic handles GET /api/v1/settings/public
func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetPublic(r.Context())
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to load public settings"))
		return
	}
	httpx.OK(w, list)
}

// UpdateBatchRequest payload for updating settings.
type UpdateBatchRequest struct {
	Settings map[string]any `json:"settings" validate:"required"`
}

// UpdateBatch handles PUT /api/v1/settings
func (h *Handler) UpdateBatch(w http.ResponseWriter, r *http.Request) {
	var req UpdateBatchRequest
	if err := httpx.DecodeAndValidate(r, &req); err != nil {
		httpx.Fail(r.Context(), w, err)
		return
	}

	if len(req.Settings) == 0 {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeValidationError, "settings map cannot be empty"))
		return
	}

	p, _ := rbac.GetPrincipal(r.Context())
	var actorID *uuid.UUID
	if p != nil {
		actorID = &p.UserID
	}

	if err := h.svc.UpdateBatch(r.Context(), actorID, req.Settings); err != nil {
		if appErr, ok := err.(*httpx.AppError); ok {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to update settings"))
		return
	}

	_ = h.audit.RecordFromRequest(r, "settings.update", "app_settings", nil, map[string]any{
		"updated_keys": func() []string {
			keys := make([]string, 0, len(req.Settings))
			for k := range req.Settings {
				keys = append(keys, k)
			}
			return keys
		}(),
	})

	list, _ := h.svc.GetAll(r.Context())
	httpx.OK(w, list)
}
