package httpx

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP extracts the caller's IP address from an HTTP request, returning a
// bare address with no port (e.g. "10.0.0.5", "::1") or "" if none can be
// determined.
//
// Why this exists: req.RemoteAddr is always "host:port" in a real server, e.g.
// "10.0.0.5:54321". Several call sites used it directly and then fed it to
// net.ParseIP or a Postgres `inet` column, both of which reject a value that
// carries a port:
//
//   - audit_logs.ip (inet) raised "invalid input syntax for type inet" on every
//     insert. Callers ignored the error, so the audit log was always empty.
//   - refresh_tokens.ip_address is written only when net.ParseIP succeeds, so
//     every session recorded a NULL IP instead of the real client.
//
// Both failures were silent. Centralising the parsing here means there is one
// place to get it right, and one place to test.
//
// X-Forwarded-For takes precedence because the API normally runs behind a
// proxy, where RemoteAddr is the proxy's address. It is a comma-separated
// chain and the first entry is the original client, so only that is used. The
// header is client-controlled and therefore only trusted to the extent the
// deployment controls it — same caveat as any X-Forwarded-* use.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// "client, proxy1, proxy2" → "client"
		ip = strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	return normalizeIP(ip)
}

// normalizeIP strips a port and surrounding IPv6 brackets, returning "" when
// the input is not a usable IP address.
func normalizeIP(ip string) string {
	if ip == "" {
		return ""
	}

	// SplitHostPort handles "host:port" and "[v6]:port" but errors on a bare
	// host or an unbracketed IPv6 literal; in those cases keep the input as-is
	// and let ParseIP decide.
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// "[::1]" → "::1"
	ip = strings.Trim(ip, "[]")

	// Values that do not parse must become "" rather than be passed on: an
	// empty string becomes SQL NULL, which is correct, whereas an unparseable
	// value would abort an insert or be discarded later anyway.
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}
