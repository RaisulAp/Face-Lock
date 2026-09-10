// Package storage defines the object-storage abstraction every fase that
// handles photos (Fase 3 reference photos, Fase 4 attendance photos) builds
// on, behind an interface so the dev-only LocalStore can be swapped for S3
// or MinIO (D9, Fase 0 § 2.0) without touching call sites.
package storage

import (
	"context"
	"io"
)

// Store is intentionally small. Put/Get/Delete are what every fase that
// touches biometric photos actually uses.
type Store interface {
	// Put writes r under key, returning the key actually stored (drivers
	// may normalize it) and an error. contentType is stored as metadata
	// where the backend supports it (S3); LocalStore ignores it beyond
	// validation.
	Put(ctx context.Context, key string, r io.Reader, contentType string) (string, error)

	// Get opens key for reading. Caller must Close the returned ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes key. Deleting a key that does not exist is not an
	// error — callers (retention jobs, Fase 4 § 2.9) call this
	// idempotently.
	Delete(ctx context.Context, key string) error

	// Copy duplicates an object from srcKey to dstKey within the store.
	Copy(ctx context.Context, srcKey, dstKey string) error

	// List returns all keys matching the given prefix.
	List(ctx context.Context, prefix string) ([]string, error)

	// SignedURL returns a time-limited URL for key.
	//
	// Reserved for a future CDN-backed deployment — [REV-INF-04] (Fase 3 §
	// 2.2 / D14) locks the decision that Fase 3–6 access photos through a
	// streaming API endpoint with a per-request permission check, NOT a
	// signed URL, because a signed URL bypasses that per-request check
	// once issued. No caller in this codebase should invoke SignedURL
	// until that decision is revisited with a new ADR.
	SignedURL(ctx context.Context, key string, ttlSeconds int) (string, error)
}

// ErrNotFound is returned by Get when key does not exist.
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (*notFoundError) Error() string { return "storage: key not found" }
