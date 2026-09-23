package audit

import (
	"net/http/httptest"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/httpx"
)

// TestClientIPGuard locks in the regression that made every audit insert fail.
//
// The audit_logs.ip column is `inet`. Postgres rejects "host:port" for that
// type with "invalid input syntax for type inet", and req.RemoteAddr is always
// "host:port" in a real server. Because callers discard the returned error
// (`_ = RecordFromRequest(...)`), the entire audit log silently stayed empty.
//
// The parsing itself is unit-tested in httpx (httpx.ClientIP); what matters
// here is the invariant the audit recorder depends on: whatever value reaches
// the SQL layer is either a bare, valid IP or "" (which NULLIF turns into
// NULL). This test asserts that contract against the exact helper audit uses.
func TestClientIPGuard(t *testing.T) {
	cases := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       string
	}{
		{
			name:       "strips the port from an IPv4 remote address",
			remoteAddr: "192.168.1.5:54321",
			want:       "192.168.1.5",
		},
		{
			name:       "keeps a bare IPv4 address",
			remoteAddr: "10.0.0.7",
			want:       "10.0.0.7",
		},
		{
			name:       "strips brackets and port from an IPv6 literal",
			remoteAddr: "[2001:db8::1]:443",
			want:       "2001:db8::1",
		},
		{
			name:       "keeps a bare IPv6 literal",
			remoteAddr: "::1",
			want:       "::1",
		},
		{
			name:       "takes only the first hop of X-Forwarded-For",
			remoteAddr: "127.0.0.1:1234",
			forwarded:  "203.0.113.9, 70.41.3.18",
			want:       "203.0.113.9",
		},
		{
			name:       "unparseable address becomes empty rather than breaking the insert",
			remoteAddr: "not-an-ip",
			want:       "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/employees/me", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}

			got := httpx.ClientIP(req)
			if got != tc.want {
				t.Errorf("httpx.ClientIP() = %q, want %q", got, tc.want)
			}
		})
	}
}
