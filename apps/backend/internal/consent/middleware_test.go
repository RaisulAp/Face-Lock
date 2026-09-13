package consent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type mockRepo struct {
	hasConsent bool
	activeDoc  *ConsentDocument
	err        error
}

func (m *mockRepo) GetActiveDocument(ctx context.Context) (*ConsentDocument, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.activeDoc, nil
}
func (m *mockRepo) GetDocumentByVersion(ctx context.Context, version string) (*ConsentDocument, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.activeDoc, nil
}
func (m *mockRepo) GetActiveConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error) {
	return nil, nil
}
func (m *mockRepo) GetLatestConsentByEmployeeID(ctx context.Context, employeeID uuid.UUID) (*BiometricConsent, error) {
	return nil, nil
}
func (m *mockRepo) GrantConsent(ctx context.Context, c *BiometricConsent) error {
	return nil
}
func (m *mockRepo) WithdrawConsent(ctx context.Context, employeeID uuid.UUID, reason string) (int, error) {
	return 0, nil
}
func (m *mockRepo) HasActiveConsent(ctx context.Context, employeeID uuid.UUID) (bool, error) {
	return m.hasConsent, m.err
}

func TestRequireConsent_Middleware(t *testing.T) {
	empID := uuid.New()
	mock := &mockRepo{hasConsent: false}
	cache := NewCache(0)
	svc := NewService(mock, cache, nil, nil)

	handler := RequireConsent(svc, func(r *http.Request) (uuid.UUID, error) {
		return empID, nil
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// 1. When consent is missing -> 403 Forbidden with CONSENT_REQUIRED
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	// 2. When consent is granted -> 200 OK
	mock.hasConsent = true
	cache.Invalidate(empID)

	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)

	if rr2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body: %s)", rr2.Code, rr2.Body.String())
	}
}
