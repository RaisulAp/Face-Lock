package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalStore_PutGetDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	ctx := context.Background()

	key, err := store.Put(ctx, "employees/abc/photo1.jpg", strings.NewReader("fake-jpeg-bytes"), "image/jpeg")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if key != "employees/abc/photo1.jpg" {
		t.Errorf("expected key to round-trip unchanged, got %q", key)
	}

	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	// Close explicitly (not deferred) — on Windows, Delete below cannot
	// remove a file that is still open under a live handle.
	if err := rc.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if string(body) != "fake-jpeg-bytes" {
		t.Errorf("expected round-tripped content, got %q", body)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := store.Get(ctx, key); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	// Deleting again must be idempotent, not an error.
	if err := store.Delete(ctx, key); err != nil {
		t.Errorf("expected idempotent delete, got %v", err)
	}
}

func TestLocalStore_RejectsPathTraversal(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	ctx := context.Background()

	_, err = store.Put(ctx, "../../etc/passwd", strings.NewReader("x"), "text/plain")
	if err == nil {
		t.Fatal("expected an error for a path-traversal key, got nil")
	}
}

func TestLocalStore_SignedURLIsUnimplemented(t *testing.T) {
	// REV-INF-04: LocalStore deliberately refuses SignedURL — Fase 3-6
	// access photos via a streaming endpoint, never a signed URL.
	dir := t.TempDir()
	store, err := NewLocalStore(dir)
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}
	if _, err := store.SignedURL(context.Background(), "any-key", 60); err == nil {
		t.Fatal("expected SignedURL to return an error, got nil")
	}
}
