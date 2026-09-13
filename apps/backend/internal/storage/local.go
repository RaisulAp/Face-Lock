package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStore implements Store on the local filesystem, rooted at baseDir.
// This is the dev-only STORAGE_DRIVER=local backend (Fase 0 § 2.0 D9); an
// S3-backed implementation is added when D13 (Fase 3) is confirmed.
type LocalStore struct {
	baseDir string
}

// NewLocalStore creates baseDir if it does not exist yet.
func NewLocalStore(baseDir string) (*LocalStore, error) {
	if err := os.MkdirAll(baseDir, 0o750); err != nil {
		return nil, fmt.Errorf("storage: create base dir: %w", err)
	}
	return &LocalStore{baseDir: baseDir}, nil
}

func (s *LocalStore) resolve(key string) (string, error) {
	// Reject path traversal outright — a key is an opaque identifier, never
	// a filesystem path chosen by a caller.
	if strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return filepath.Join(s.baseDir, filepath.FromSlash(key)), nil
}

func (s *LocalStore) Put(_ context.Context, key string, r io.Reader, _ string) (string, error) {
	path, err := s.resolve(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", fmt.Errorf("storage: create parent dir: %w", err)
	}
	f, err := os.Create(path) //nolint:gosec // path is derived from resolve(), which rejects traversal
	if err != nil {
		return "", fmt.Errorf("storage: create file: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close after explicit error handling below

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("storage: write file: %w", err)
	}
	return key, nil
}

func (s *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path) //nolint:gosec // path is derived from resolve(), which rejects traversal
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("storage: open file: %w", err)
	}
	return f, nil
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("storage: delete file: %w", err)
	}
	return nil
}

func (s *LocalStore) Copy(_ context.Context, srcKey, dstKey string) error {
	srcPath, err := s.resolve(srcKey)
	if err != nil {
		return err
	}
	dstPath, err := s.resolve(dstKey)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
		return fmt.Errorf("storage: create parent dir: %w", err)
	}
	srcFile, err := os.Open(srcPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("storage: open src file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("storage: create dst file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("storage: copy file: %w", err)
	}
	return nil
}

func (s *LocalStore) List(_ context.Context, prefix string) ([]string, error) {
	var keys []string
	if _, err := os.Stat(s.baseDir); os.IsNotExist(err) {
		return keys, nil
	}
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(s.baseDir, path)
		if err != nil {
			return err
		}
		slashKey := filepath.ToSlash(rel)
		if prefix == "" || strings.HasPrefix(slashKey, prefix) {
			keys = append(keys, slashKey)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage: list: %w", err)
	}
	return keys, nil
}

func (s *LocalStore) SignedURL(_ context.Context, _ string, _ int) (string, error) {
	return "", errors.New("storage: SignedURL is not used by LocalStore — see REV-INF-04 (photos are served via a streaming API, not signed URLs)")
}
