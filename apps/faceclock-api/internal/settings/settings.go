package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
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
		case "attendance.max_distance_meter":
			if num < 10 || num > 10000 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance.max_distance_meter must be between 10 and 10000")
			}
		case "attendance.late_tolerance_minutes":
			if num < 0 || num > 120 {
				return httpx.NewAppError(httpx.CodeValidationError, "attendance.late_tolerance_minutes must be between 0 and 120")
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
		case "attendance.work_start_time", "attendance.work_end_time":
			if _, err := time.Parse("15:04", str); err != nil {
				return httpx.NewAppError(httpx.CodeValidationError, fmt.Sprintf("setting %q must be in HH:MM format", key))
			}
		case "face.liveness_strictness":
			if str != "low" && str != "standard" && str != "high" {
				return httpx.NewAppError(httpx.CodeValidationError, "face.liveness_strictness must be one of: low, standard, high")
			}
		}
	}
	return nil
}

// Handler handles HTTP requests for settings.
type Handler struct {
	svc   *Service
	audit *audit.Recorder
}

// NewHandler creates a settings handler.
func NewHandler(svc *Service, audit *audit.Recorder) *Handler {
	return &Handler{svc: svc, audit: audit}
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
