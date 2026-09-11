package location

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/geo"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/settings"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OfficeLocation represents an office location domain model (Plan/05-Fase4.md § 2.2).
type OfficeLocation struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	RadiusMeter int        `json:"radius_meter"`
	IsActive    bool       `json:"is_active"`
	Address     *string    `json:"address,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreateLocationInput represents payload to create a new office location.
type CreateLocationInput struct {
	Name        string   `json:"name"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	RadiusMeter *int     `json:"radius_meter"`
	IsActive    *bool    `json:"is_active"`
	Address     *string  `json:"address"`

	// Aliases for lat/lng per API conventions
	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

// UpdateLocationInput represents payload to update an existing office location.
type UpdateLocationInput struct {
	Name        *string  `json:"name"`
	Latitude    *float64 `json:"latitude"`
	Longitude   *float64 `json:"longitude"`
	RadiusMeter *int     `json:"radius_meter"`
	IsActive    *bool    `json:"is_active"`
	Address     *string  `json:"address"`

	Lat *float64 `json:"lat"`
	Lng *float64 `json:"lng"`
}

// Service provides management of office locations.
type Service struct {
	db              *pgxpool.Pool
	settingsService *settings.Service
	audit           *audit.Recorder
}

// NewService creates a new office location service.
func NewService(db *pgxpool.Pool, settingsService *settings.Service, audit *audit.Recorder) *Service {
	return &Service{
		db:              db,
		settingsService: settingsService,
		audit:           audit,
	}
}

// List returns office locations filtered by is_active and include_deleted.
func (s *Service) List(ctx context.Context, isActive *bool, includeDeleted bool) ([]OfficeLocation, error) {
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	if !includeDeleted {
		conditions = append(conditions, "deleted_at IS NULL")
	}

	if isActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *isActive)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT id, name, latitude, longitude, radius_meter, is_active, address, deleted_at, created_at, updated_at
		FROM office_locations
		%s
		ORDER BY is_active DESC, name ASC
	`, whereClause)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying office locations: %w", err)
	}
	defer rows.Close()

	var locations []OfficeLocation
	for rows.Next() {
		var loc OfficeLocation
		if err := rows.Scan(
			&loc.ID,
			&loc.Name,
			&loc.Latitude,
			&loc.Longitude,
			&loc.RadiusMeter,
			&loc.IsActive,
			&loc.Address,
			&loc.DeletedAt,
			&loc.CreatedAt,
			&loc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning office location: %w", err)
		}
		locations = append(locations, loc)
	}

	if locations == nil {
		locations = []OfficeLocation{}
	}

	return locations, nil
}

// GetByID returns an office location by ID.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*OfficeLocation, error) {
	const q = `
		SELECT id, name, latitude, longitude, radius_meter, is_active, address, deleted_at, created_at, updated_at
		FROM office_locations
		WHERE id = $1 AND deleted_at IS NULL
	`
	var loc OfficeLocation
	err := s.db.QueryRow(ctx, q, id).Scan(
		&loc.ID,
		&loc.Name,
		&loc.Latitude,
		&loc.Longitude,
		&loc.RadiusMeter,
		&loc.IsActive,
		&loc.Address,
		&loc.DeletedAt,
		&loc.CreatedAt,
		&loc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.NewAppError(httpx.CodeNotFound, "Office location not found")
		}
		return nil, fmt.Errorf("getting office location: %w", err)
	}
	return &loc, nil
}

// GetActiveLocations returns all active locations as geo.OfficeLocation for geofence evaluation.
func (s *Service) GetActiveLocations(ctx context.Context) ([]geo.OfficeLocation, error) {
	const q = `
		SELECT id, name, latitude, longitude, radius_meter, is_active, COALESCE(address, '')
		FROM office_locations
		WHERE is_active = true AND deleted_at IS NULL
	`
	rows, err := s.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("querying active office locations: %w", err)
	}
	defer rows.Close()

	var list []geo.OfficeLocation
	for rows.Next() {
		var item geo.OfficeLocation
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Latitude,
			&item.Longitude,
			&item.RadiusMeter,
			&item.IsActive,
			&item.Address,
		); err != nil {
			return nil, fmt.Errorf("scanning active office location: %w", err)
		}
		list = append(list, item)
	}
	return list, nil
}

// Create validates and creates a new office location.
func (s *Service) Create(ctx context.Context, in CreateLocationInput) (*OfficeLocation, error) {
	var fieldErrors []httpx.FieldError

	name := strings.TrimSpace(in.Name)
	if name == "" {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "Name is required"})
	} else if len(name) > 100 {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "Name must not exceed 100 characters"})
	}

	lat := in.Latitude
	if lat == nil {
		lat = in.Lat
	}
	if lat == nil {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "latitude", Message: "Latitude is required"})
	} else if *lat < -90.0 || *lat > 90.0 {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "latitude", Message: "Latitude must be between -90 and 90"})
	}

	lng := in.Longitude
	if lng == nil {
		lng = in.Lng
	}
	if lng == nil {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "longitude", Message: "Longitude is required"})
	} else if *lng < -180.0 || *lng > 180.0 {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "longitude", Message: "Longitude must be between -180 and 180"})
	}

	radius := 100
	if s.settingsService != nil {
		radius = s.settingsService.GetInt(ctx, "max_distance_meter", 100)
	}
	if in.RadiusMeter != nil {
		radius = *in.RadiusMeter
	}
	if radius < 10 || radius > 50000 {
		fieldErrors = append(fieldErrors, httpx.FieldError{Field: "radius_meter", Message: "Radius must be between 10 and 50000 meters"})
	}

	if len(fieldErrors) > 0 {
		return nil, httpx.NewValidationError("Validation failed", fieldErrors)
	}

	isActive := true
	if in.IsActive != nil {
		isActive = *in.IsActive
	}

	var address *string
	if in.Address != nil {
		trimmed := strings.TrimSpace(*in.Address)
		if trimmed != "" {
			address = &trimmed
		}
	}

	const q = `
		INSERT INTO office_locations (name, latitude, longitude, radius_meter, is_active, address)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, latitude, longitude, radius_meter, is_active, address, deleted_at, created_at, updated_at
	`
	var loc OfficeLocation
	err := s.db.QueryRow(ctx, q, name, *lat, *lng, radius, isActive, address).Scan(
		&loc.ID,
		&loc.Name,
		&loc.Latitude,
		&loc.Longitude,
		&loc.RadiusMeter,
		&loc.IsActive,
		&loc.Address,
		&loc.DeletedAt,
		&loc.CreatedAt,
		&loc.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, httpx.NewAppError(httpx.CodeConflict, "Office location name already in use")
		}
		return nil, fmt.Errorf("inserting office location: %w", err)
	}

	return &loc, nil
}

// Update updates an office location with guard against deactivating the last active location.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateLocationInput) (*OfficeLocation, *OfficeLocation, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock existing row
	const lockQ = `
		SELECT id, name, latitude, longitude, radius_meter, is_active, address, deleted_at, created_at, updated_at
		FROM office_locations
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	var existing OfficeLocation
	err = tx.QueryRow(ctx, lockQ, id).Scan(
		&existing.ID,
		&existing.Name,
		&existing.Latitude,
		&existing.Longitude,
		&existing.RadiusMeter,
		&existing.IsActive,
		&existing.Address,
		&existing.DeletedAt,
		&existing.CreatedAt,
		&existing.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, httpx.NewAppError(httpx.CodeNotFound, "Office location not found")
		}
		return nil, nil, fmt.Errorf("locking office location: %w", err)
	}

	var fieldErrors []httpx.FieldError

	name := existing.Name
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "Name cannot be empty"})
		} else if len(trimmed) > 100 {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "name", Message: "Name must not exceed 100 characters"})
		} else {
			name = trimmed
		}
	}

	lat := existing.Latitude
	latIn := in.Latitude
	if latIn == nil {
		latIn = in.Lat
	}
	if latIn != nil {
		if *latIn < -90.0 || *latIn > 90.0 {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "latitude", Message: "Latitude must be between -90 and 90"})
		} else {
			lat = *latIn
		}
	}

	lng := existing.Longitude
	lngIn := in.Longitude
	if lngIn == nil {
		lngIn = in.Lng
	}
	if lngIn != nil {
		if *lngIn < -180.0 || *lngIn > 180.0 {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "longitude", Message: "Longitude must be between -180 and 180"})
		} else {
			lng = *lngIn
		}
	}

	radius := existing.RadiusMeter
	if in.RadiusMeter != nil {
		if *in.RadiusMeter < 10 || *in.RadiusMeter > 50000 {
			fieldErrors = append(fieldErrors, httpx.FieldError{Field: "radius_meter", Message: "Radius must be between 10 and 50000 meters"})
		} else {
			radius = *in.RadiusMeter
		}
	}

	if len(fieldErrors) > 0 {
		return nil, nil, httpx.NewValidationError("Validation failed", fieldErrors)
	}

	isActive := existing.IsActive
	if in.IsActive != nil {
		isActive = *in.IsActive
		// Guard: if deactivating (true -> false), check if this is the last active location
		if existing.IsActive && !isActive {
			const countQ = `
				SELECT count(*) FROM office_locations
				WHERE is_active = true AND deleted_at IS NULL
				FOR UPDATE
			`
			var activeCount int
			if err := tx.QueryRow(ctx, countQ).Scan(&activeCount); err != nil {
				return nil, nil, fmt.Errorf("counting active locations: %w", err)
			}
			if activeCount <= 1 {
				return nil, nil, httpx.NewAppError(httpx.CodeConflict, "Cannot deactivate the last active office location")
			}
		}
	}

	address := existing.Address
	if in.Address != nil {
		trimmed := strings.TrimSpace(*in.Address)
		if trimmed != "" {
			address = &trimmed
		} else {
			address = nil
		}
	}

	const updateQ = `
		UPDATE office_locations
		SET name = $1, latitude = $2, longitude = $3, radius_meter = $4, is_active = $5, address = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, name, latitude, longitude, radius_meter, is_active, address, deleted_at, created_at, updated_at
	`
	var updated OfficeLocation
	err = tx.QueryRow(ctx, updateQ, name, lat, lng, radius, isActive, address, id).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Latitude,
		&updated.Longitude,
		&updated.RadiusMeter,
		&updated.IsActive,
		&updated.Address,
		&updated.DeletedAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, nil, httpx.NewAppError(httpx.CodeConflict, "Office location name already in use")
		}
		return nil, nil, fmt.Errorf("updating office location: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("committing update: %w", err)
	}

	return &existing, &updated, nil
}

// Delete soft-deletes an office location with guard against deleting the last active location.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	const lockQ = `
		SELECT id, name, is_active
		FROM office_locations
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`
	var (
		locID    uuid.UUID
		name     string
		isActive bool
	)
	err = tx.QueryRow(ctx, lockQ, id).Scan(&locID, &name, &isActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", httpx.NewAppError(httpx.CodeNotFound, "Office location not found")
		}
		return "", fmt.Errorf("locking location for delete: %w", err)
	}

	if isActive {
		const countQ = `
			SELECT count(*) FROM office_locations
			WHERE is_active = true AND deleted_at IS NULL
			FOR UPDATE
		`
		var activeCount int
		if err := tx.QueryRow(ctx, countQ).Scan(&activeCount); err != nil {
			return "", fmt.Errorf("counting active locations: %w", err)
		}
		if activeCount <= 1 {
			return "", httpx.NewAppError(httpx.CodeConflict, "Cannot delete the last active office location")
		}
	}

	const deleteQ = `
		UPDATE office_locations
		SET is_active = false, deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	if _, err := tx.Exec(ctx, deleteQ, id); err != nil {
		return "", fmt.Errorf("soft deleting office location: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("committing delete: %w", err)
	}

	return name, nil
}

// Handler provides HTTP endpoints for office locations.
type Handler struct {
	service *Service
	audit   *audit.Recorder
}

// NewHandler creates a new office location handler.
func NewHandler(service *Service, audit *audit.Recorder) *Handler {
	return &Handler{service: service, audit: audit}
}

// List handles GET /api/v1/office-locations (#66).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var isActive *bool
	if val := q.Get("is_active"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			isActive = &b
		}
	}
	includeDeleted := false
	if val := q.Get("include_deleted"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			includeDeleted = b
		}
	}

	locations, err := h.service.List(r.Context(), isActive, includeDeleted)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Failed to list office locations"))
		return
	}
	httpx.OK(w, locations)
}

// Create handles POST /api/v1/office-locations (#67).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Invalid request payload"))
		return
	}

	loc, err := h.service.Create(r.Context(), in)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Failed to create office location"))
		return
	}

	if h.audit != nil {
		resID := loc.ID.String()
		_ = h.audit.RecordFromRequest(r, "location.create", "office_location", &resID, map[string]any{
			"name":         loc.Name,
			"latitude":     loc.Latitude,
			"longitude":    loc.Longitude,
			"radius_meter": loc.RadiusMeter,
			"is_active":    loc.IsActive,
		})
	}

	httpx.Created(w, loc)
}

// GetByID handles GET /api/v1/office-locations/{id} (#68).
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "Invalid office location ID"))
		return
	}

	loc, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Failed to get office location"))
		return
	}
	httpx.OK(w, loc)
}

// Update handles PATCH/PUT /api/v1/office-locations/{id} (#69).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "Invalid office location ID"))
		return
	}

	var in UpdateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "Invalid request payload"))
		return
	}

	existing, loc, err := h.service.Update(r.Context(), id, in)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Failed to update office location"))
		return
	}

	if h.audit != nil {
		resID := loc.ID.String()
		_ = h.audit.RecordFromRequest(r, "location.update", "office_location", &resID, map[string]any{
			"old": map[string]any{
				"name":         existing.Name,
				"latitude":     existing.Latitude,
				"longitude":    existing.Longitude,
				"radius_meter": existing.RadiusMeter,
				"is_active":    existing.IsActive,
			},
			"new": map[string]any{
				"name":         loc.Name,
				"latitude":     loc.Latitude,
				"longitude":    loc.Longitude,
				"radius_meter": loc.RadiusMeter,
				"is_active":    loc.IsActive,
			},
		})
	}

	httpx.OK(w, loc)
}

// Delete handles DELETE /api/v1/office-locations/{id} (#70).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeNotFound, "Invalid office location ID"))
		return
	}

	name, err := h.service.Delete(r.Context(), id)
	if err != nil {
		var appErr *httpx.AppError
		if errors.As(err, &appErr) {
			httpx.Fail(r.Context(), w, appErr)
			return
		}
		httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "Failed to delete office location"))
		return
	}

	if h.audit != nil {
		resID := id.String()
		_ = h.audit.RecordFromRequest(r, "location.delete", "office_location", &resID, map[string]any{
			"name": name,
		})
	}

	httpx.OK(w, map[string]string{"message": "Office location deleted"})
}

// RegisterRoutes registers endpoints #66-#70 with Chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/office-locations", func(r chi.Router) {
		r.With(rbac.RequirePermission("location.read")).Get("/", h.List)
		r.With(rbac.RequireAnyPermission("location.create", "location.manage")).Post("/", h.Create)
		r.Route("/{id}", func(r chi.Router) {
			r.With(rbac.RequirePermission("location.read")).Get("/", h.GetByID)
			r.With(rbac.RequireAnyPermission("location.update", "location.manage")).Patch("/", h.Update)
			r.With(rbac.RequireAnyPermission("location.update", "location.manage")).Put("/", h.Update)
			r.With(rbac.RequireAnyPermission("location.delete", "location.manage")).Delete("/", h.Delete)
		})
	})
}
