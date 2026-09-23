package httpx

import (
	"net"
	"net/http/httptest"
	"testing"
)

// TestClientIP covers the two silent failures this helper was introduced to
// fix:
//
//   - audit_logs.ip is an `inet` column. Postgres rejects "host:port" with
//     "invalid input syntax for type inet", and req.RemoteAddr always includes
//     a port, so every audit insert failed. The errors were discarded by the
//     callers, leaving the audit log permanently empty.
//   - refresh_tokens.ip_address is only written when net.ParseIP succeeds, so
//     the port-bearing value silently produced NULL for every session.
//
// The contract: the result is either a bare IP that Postgres accepts for
// `inet` and that net.ParseIP understands, or "" (stored as NULL). Never a
// port, never garbage.
func TestClientIP(t *testing.T) {
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
			name:       "keeps a bare IPv4 address with no port",
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
			name:       "keeps a bracketed IPv6 literal with no port",
			remoteAddr: "[::1]",
			want:       "::1",
		},
		{
			name:       "takes only the first hop of X-Forwarded-For",
			remoteAddr: "127.0.0.1:1234",
			forwarded:  "203.0.113.9, 70.41.3.18, 150.172.238.178",
			want:       "203.0.113.9",
		},
		{
			name:       "trims whitespace around the first forwarded hop",
			remoteAddr: "127.0.0.1:1234",
			forwarded:  "   203.0.113.9  , 70.41.3.18",
			want:       "203.0.113.9",
		},
		{
			name:       "X-Forwarded-For carrying a port is normalised",
			remoteAddr: "127.0.0.1:1234",
			forwarded:  "203.0.113.9:8080",
			want:       "203.0.113.9",
		},
		{
			name:       "X-Forwarded-For wins over RemoteAddr",
			remoteAddr: "10.1.1.1:9999",
			forwarded:  "198.51.100.7",
			want:       "198.51.100.7",
		},
		{
			name:       "empty remote address yields empty, not garbage",
			remoteAddr: "",
			want:       "",
		},
		{
			name:       "unparseable address is dropped rather than aborting the insert",
			remoteAddr: "not-an-ip",
			want:       "",
		},
		{
			name:       "hostname is dropped because inet cannot store it",
			remoteAddr: "some-host.internal:9999",
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

			got := ClientIP(req)
			if got != tc.want {
				t.Fatalf("ClientIP() = %q, want %q", got, tc.want)
			}

			// A non-empty result must be something both Postgres `inet` and
			// net.ParseIP accept — that is the whole point of this helper.
			if got != "" && net.ParseIP(got) == nil {
				t.Errorf("ClientIP() returned %q, which net.ParseIP rejects", got)
			}
		})
	}
}

// TestClientIPNilRequest guards the nil case so a caller that forgets to check
// cannot panic the request handler.
func TestClientIPNilRequest(t *testing.T) {
	if got := ClientIP(nil); got != "" {
		t.Errorf("ClientIP(nil) = %q, want %q", got, "")
	}
}
