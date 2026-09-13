package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFProtection(t *testing.T) {
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	allowedOrigins := []string{"https://app.faceclock.io", "http://localhost:5173"}
	csrfMiddleware := CSRFProtection(allowedOrigins)
	handler := csrfMiddleware(dummyHandler)

	tests := []struct {
		name           string
		method         string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "safe method GET without headers passes",
			method:         http.MethodGet,
			headers:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "safe method HEAD without headers passes",
			method:         http.MethodHead,
			headers:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "safe method OPTIONS without headers passes",
			method:         http.MethodOptions,
			headers:        nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "mutating POST without headers returns 403",
			method:         http.MethodPost,
			headers:        nil,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "mutating POST with X-Requested-With: XMLHttpRequest passes",
			method: http.MethodPost,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating POST with lowercase xmlhttprequest passes",
			method: http.MethodPost,
			headers: map[string]string{
				"X-Requested-With": "xmlhttprequest",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating POST with X-CSRF-Token passes",
			method: http.MethodPost,
			headers: map[string]string{
				"X-CSRF-Token": "some-random-token",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating PUT with X-Requested-With passes",
			method: http.MethodPut,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating DELETE with X-Requested-With passes",
			method: http.MethodDelete,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating PATCH with X-Requested-With passes",
			method: http.MethodPatch,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "mutating POST with valid header but untrusted Origin returns 403",
			method: http.MethodPost,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
				"Origin":           "https://evil.com",
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "mutating POST with valid header and trusted Origin passes",
			method: http.MethodPost,
			headers: map[string]string{
				"X-Requested-With": "XMLHttpRequest",
				"Origin":           "https://app.faceclock.io",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected status %d, got %d. body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}

	// Test with Wildcard Origin (development mode)
	t.Run("wildcard origin allows any origin", func(t *testing.T) {
		devMiddleware := CSRFProtection([]string{"*"})
		devHandler := devMiddleware(dummyHandler)

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		req.Header.Set("Origin", "http://localhost:3000")
		rec := httptest.NewRecorder()
		devHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}
