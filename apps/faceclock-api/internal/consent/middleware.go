package consent

import (
	"net/http"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
	"github.com/google/uuid"
)

// RequireConsent ensures that the target employee has active biometric consent.
// resolveEmployeeID extracts the employee UUID from the request (e.g. from session, auth context, or URL path).
func RequireConsent(consentSvc *Service, resolveEmployeeID func(*http.Request) (uuid.UUID, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			employeeID, err := resolveEmployeeID(r)
			if err != nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeBadRequest, "invalid employee identification"))
				return
			}

			hasConsent, err := consentSvc.HasActiveConsent(r.Context(), employeeID)
			if err != nil {
				httpx.Fail(r.Context(), w, httpx.NewAppError(httpx.CodeInternalError, "failed to verify biometric consent status"))
				return
			}

			if !hasConsent {
				appErr := &httpx.AppError{
					Code:    httpx.CodeConsentRequired,
					Message: "Biometric consent is required before performing this operation",
					Extra: map[string]any{
						"action_url": "/api/v1/consents",
					},
				}
				httpx.Fail(r.Context(), w, appErr)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
