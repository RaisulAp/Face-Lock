// Package version holds build-time metadata injected via -ldflags in the
// Dockerfile build stage (Fase 0 § 4: "commit dan build_time diisi lewat
// -ldflags saat build, bukan hardcode"). Defaults here are what a plain
// `go run`/`go test` sees locally, never what ships in the image.
package version

import (
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
)

var (
	Version   = "0.1.0-dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// MinSupportedClient answers REV-EP-10 (Plan/09-Revisions-Log.md § 2 B):
	// GET /api/v1/version exposes the minimum client version per platform
	// so an old mobile/web build can show "Perbarui aplikasi" instead of
	// failing on an incompatible contract. Fase 0 has no client versioning
	// scheme yet (Fase 6/7 don't exist), so this starts as an explicit
	// "no constraint yet" value rather than a guessed number.
	MinSupportedClient = map[string]string{
		"web":    "0.0.0",
		"mobile": "0.0.0",
	}
)

// Handler serves GET /api/v1/version (Fase 0 § 4).
func Handler(w http.ResponseWriter, r *http.Request) {
	httpx.OK(w, map[string]any{
		"version":              Version,
		"commit":               Commit,
		"build_time":           BuildTime,
		"min_supported_client": MinSupportedClient,
	})
}
