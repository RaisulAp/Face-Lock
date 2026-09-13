package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/audit"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/rbac"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestValidateSettingValue(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		valType string
		val     any
		wantErr bool
	}{
		// Similarity threshold (0.0 - 1.0)
		{
			name:    "valid similarity threshold",
			key:     "face.similarity_threshold",
			valType: "number",
			val:     0.75,
			wantErr: false,
		},
		{
			name:    "similarity threshold too high",
			key:     "face.similarity_threshold",
			valType: "number",
			val:     1.5,
			wantErr: true,
		},
		{
			name:    "similarity threshold negative",
			key:     "face.similarity_threshold",
			valType: "number",
			val:     -0.1,
			wantErr: true,
		},
		// Max distance meter (10 - 10000)
		{
			name:    "valid max distance meter",
			key:     "attendance.max_distance_meter",
			valType: "number",
			val:     100,
			wantErr: false,
		},
		{
			name:    "max distance meter too small",
			key:     "attendance.max_distance_meter",
			valType: "number",
			val:     5,
			wantErr: true,
		},
		{
			name:    "max distance meter too large",
			key:     "attendance.max_distance_meter",
			valType: "number",
			val:     20000,
			wantErr: true,
		},
		// Boolean
		{
			name:    "valid boolean true",
			key:     "attendance.require_location",
			valType: "boolean",
			val:     true,
			wantErr: false,
		},
		{
			name:    "invalid boolean string given",
			key:     "attendance.require_location",
			valType: "boolean",
			val:     "true",
			wantErr: true,
		},
		// Work time (HH:MM)
		{
			name:    "valid work start time",
			key:     "attendance.work_start_time",
			valType: "string",
			val:     "08:30",
			wantErr: false,
		},
		{
			name:    "invalid work start time format",
			key:     "attendance.work_start_time",
			valType: "string",
			val:     "8:30 AM",
			wantErr: true,
		},
		// Liveness strictness (low, standard, high)
		{
			name:    "valid liveness strictness standard",
			key:     "face.liveness_strictness",
			valType: "string",
			val:     "standard",
			wantErr: false,
		},
		{
			name:    "invalid liveness strictness",
			key:     "face.liveness_strictness",
			valType: "string",
			val:     "ultra",
			wantErr: true,
		},
		// Security settings
		{
			name:    "valid max failed login",
			key:     "security.max_failed_login",
			valType: "number",
			val:     5,
			wantErr: false,
		},
		{
			name:    "invalid max failed login",
			key:     "security.max_failed_login",
			valType: "number",
			val:     1,
			wantErr: true,
		},
		{
			name:    "valid retention days",
			key:     "storage.retention_days",
			valType: "number",
			val:     365,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSettingValue(tt.key, tt.valType, tt.val)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSettingValue(%s, %v) wantErr=%v, got=%v", tt.key, tt.val, tt.wantErr, err)
			}
			if err != nil {
				appErr, ok := err.(*httpx.AppError)
				if !ok || appErr.Code != httpx.CodeValidationError {
					t.Errorf("expected CodeValidationError, got %v", err)
				}
			}
		})
	}
}

func getTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://faceclock:devpassword123@localhost:5434/faceclock?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping db test, cannot connect to %s: %v", dsn, err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db test, ping failed: %v", err)
	}
	return pool
}

func TestSettingsServiceAndHandler(t *testing.T) {
	pool := getTestDB(t)
	svc := NewService(pool)
	auditRec := audit.NewRecorder(pool)
	handler := NewHandler(svc, auditRec)

	ctx := context.Background()

	// 1. GetAll
	all, err := svc.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(all) == 0 {
		t.Fatalf("expected seeded settings, got 0")
	}

	// 2. GetPublic
	pub, err := svc.GetPublic(ctx)
	if err != nil {
		t.Fatalf("GetPublic failed: %v", err)
	}
	for _, s := range pub {
		if !s.IsPublic {
			t.Errorf("expected only public settings in GetPublic, got key %s", s.Key)
		}
	}

	// 3. GetInt
	maxLogin := svc.GetInt(ctx, "security.max_failed_login", 5)
	if maxLogin <= 0 {
		t.Errorf("expected positive int for max_failed_login, got %d", maxLogin)
	}

	// Fallback for missing int
	defVal := svc.GetInt(ctx, "non.existent.key", 42)
	if defVal != 42 {
		t.Errorf("expected default value 42, got %d", defVal)
	}

	// 4. GetByKey
	setting, err := svc.GetByKey(ctx, "face.similarity_threshold")
	if err != nil {
		t.Fatalf("GetByKey failed: %v", err)
	}
	if setting.Key != "face.similarity_threshold" {
		t.Errorf("expected key face.similarity_threshold, got %s", setting.Key)
	}

	// 5. UpdateSingle
	var actorID uuid.UUID
	_ = pool.QueryRow(ctx, "SELECT id FROM users WHERE email = 'admin@faceclock.local'").Scan(&actorID)
	oldS, newS, err := svc.UpdateSingle(ctx, &actorID, "face.similarity_threshold", 0.82)
	if err != nil {
		t.Fatalf("UpdateSingle failed: %v", err)
	}
	if newS.Key != "face.similarity_threshold" {
		t.Errorf("expected key face.similarity_threshold, got %s", newS.Key)
	}
	_ = oldS

	// Revert UpdateSingle
	_, _, _ = svc.UpdateSingle(ctx, &actorID, "face.similarity_threshold", 0.75)

	// 6. UpdateSingle invalid value validation
	_, _, err = svc.UpdateSingle(ctx, &actorID, "face.similarity_threshold", 2.5)
	if err == nil {
		t.Fatalf("expected validation error on invalid similarity_threshold")
	}

	// 7. UpdateBatch
	err = svc.UpdateBatch(ctx, &actorID, map[string]any{
		"face.similarity_threshold": 0.80,
	})
	if err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	// Revert
	_ = svc.UpdateBatch(ctx, &actorID, map[string]any{
		"face.similarity_threshold": 0.75,
	})

	// 8. Handler List (public unauthenticated)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for public List, got %d", w.Code)
	}

	// 9. Handler List (authenticated with settings.update permission)
	adminPrincipal := &rbac.Principal{
		UserID:      actorID,
		Roles:       []string{"super_admin"},
		Permissions: map[string]struct{}{"settings.update": {}},
	}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil).WithContext(rbac.WithPrincipal(context.Background(), adminPrincipal))
	w = httptest.NewRecorder()
	handler.List(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin List, got %d", w.Code)
	}

	// 10. Handler GetByKey (admin authenticated for private setting)
	rCtx := chi.NewRouteContext()
	rCtx.URLParams.Add("key", "face.similarity_threshold")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/face.similarity_threshold", nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetByKey(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GetByKey admin, got %d", w.Code)
	}

	// Unauthenticated on public setting
	if len(pub) > 0 {
		pubKey := pub[0].Key
		rCtxPub := chi.NewRouteContext()
		rCtxPub.URLParams.Add("key", pubKey)
		req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/"+pubKey, nil).
			WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtxPub))
		w = httptest.NewRecorder()
		handler.GetByKey(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200 for public GetByKey (%s), got %d", pubKey, w.Code)
		}
	}

	// 11. Handler UpdateSingle
	reqBody, _ := json.Marshal(UpdateSettingRequest{Value: 0.78})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings/face.similarity_threshold", bytes.NewReader(reqBody)).WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for Handler.Update, got %d", w.Code)
	}

	// 12. Handler UpdateBatch
	batchBody, _ := json.Marshal(UpdateBatchRequest{Settings: map[string]any{"face.similarity_threshold": 0.75}})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(batchBody)).WithContext(rbac.WithPrincipal(context.Background(), adminPrincipal))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.UpdateBatch(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for Handler.UpdateBatch, got %d", w.Code)
	}

	// 13. Handler GetPublic and GetAll aliases
	w = httptest.NewRecorder()
	handler.GetPublic(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/public", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GetPublic, got %d", w.Code)
	}

	// 14. Error handling in handlers
	// GetByKey non-existent
	rCtxNone := chi.NewRouteContext()
	rCtxNone.URLParams.Add("key", "does.not.exist")
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/does.not.exist", nil).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtxNone))
	w = httptest.NewRecorder()
	handler.GetByKey(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent key, got %d", w.Code)
	}

	// GetByKey private setting unauthenticated
	req = httptest.NewRequest(http.MethodGet, "/api/v1/settings/face.similarity_threshold", nil).
		WithContext(context.WithValue(context.Background(), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.GetByKey(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unauthenticated private setting, got %d", w.Code)
	}

	// Update with invalid JSON
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings/face.similarity_threshold", bytes.NewReader([]byte(`{bad`))).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in update, got %d", w.Code)
	}

	// Update with validation error
	badValBody, _ := json.Marshal(UpdateSettingRequest{Value: 99.9})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings/face.similarity_threshold", bytes.NewReader(badValBody)).
		WithContext(context.WithValue(rbac.WithPrincipal(context.Background(), adminPrincipal), chi.RouteCtxKey, rCtx))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Update(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid value in update, got %d", w.Code)
	}

	// UpdateBatch with invalid JSON
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader([]byte(`{bad`))).
		WithContext(rbac.WithPrincipal(context.Background(), adminPrincipal))
	w = httptest.NewRecorder()
	handler.UpdateBatch(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad json in update batch, got %d", w.Code)
	}

	// UpdateBatch with empty settings
	emptyBody, _ := json.Marshal(UpdateBatchRequest{Settings: map[string]any{}})
	req = httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewReader(emptyBody)).
		WithContext(rbac.WithPrincipal(context.Background(), adminPrincipal))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.UpdateBatch(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty settings in update batch, got %d", w.Code)
	}

	// GetAll alias
	w = httptest.NewRecorder()
	handler.GetAll(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for GetAll, got %d", w.Code)
	}
}
