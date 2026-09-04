// Package health implements /healthz and /readyz (Fase 0 § 4).
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/inference"
	"github.com/faceclock/faceclock/apps/faceclock-api/internal/platform/postgres"
)

// perCheckTimeout matches Fase 0 § 4: "setiap check punya timeout sendiri 2
// detik; total /readyz tidak boleh lebih dari 5 detik."
const perCheckTimeout = 2 * time.Second

// Handler holds every dependency /readyz needs to check.
type Handler struct {
	DB        *pgxpool.Pool
	Inference *inference.Client
}

// Healthz never touches a dependency (Fase 0 § 4 / E3): the process being
// alive is the only thing being reported.
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]string{"status": "ok"})
}

// Readyz checks database and inference connectivity, each bounded to
// perCheckTimeout, and returns 503 with status "degraded" if either check
// fails (E3, E4) — never hangs the caller past ~2×perCheckTimeout.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	dbCheck := h.checkDatabase(ctx)
	inferenceCheck := h.checkInference(ctx)

	overallStatus := "ok"
	httpStatus := http.StatusOK
	if dbCheck["status"] != "ok" || inferenceCheck["status"] != "ok" {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	httpx.Status(w, httpStatus, map[string]any{
		"status": overallStatus,
		"checks": map[string]any{
			"database":  dbCheck,
			"inference": inferenceCheck,
		},
	})
}

func (h *Handler) checkDatabase(ctx context.Context) map[string]any {
	checkCtx, cancel := context.WithTimeout(ctx, perCheckTimeout)
	defer cancel()

	latency, err := postgres.Ping(checkCtx, h.DB)
	if err != nil {
		return map[string]any{"status": "fail", "error": err.Error()}
	}
	return map[string]any{"status": "ok", "latency_ms": latency.Milliseconds()}
}

func (h *Handler) checkInference(ctx context.Context) map[string]any {
	checkCtx, cancel := context.WithTimeout(ctx, perCheckTimeout)
	defer cancel()

	start := time.Now()
	_, err := h.Inference.Health(checkCtx)
	if err != nil {
		return map[string]any{"status": "fail", "error": err.Error()}
	}
	return map[string]any{"status": "ok", "latency_ms": time.Since(start).Milliseconds()}
}
