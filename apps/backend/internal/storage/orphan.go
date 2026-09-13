package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// OrphanCleaner sweeps storage objects that are no longer referenced in the database.
// As specified in Fase 3 § 2.8 and E20, it ensures storage does not leak orphan files.
type OrphanCleaner struct {
	db     *pgxpool.Pool
	store  Store
	logger *slog.Logger
}

// NewOrphanCleaner creates a cleaner instance.
func NewOrphanCleaner(db *pgxpool.Pool, store Store, logger *slog.Logger) *OrphanCleaner {
	return &OrphanCleaner{
		db:     db,
		store:  store,
		logger: logger,
	}
}

// CleanStaging deletes staging photos belonging to expired sessions or abandoned staging entries older than ttl.
func (c *OrphanCleaner) CleanStaging(ctx context.Context, ttl time.Duration) (int, error) {
	intervalStr := fmt.Sprintf("%d seconds", int(ttl.Seconds()))
	rows, err := c.db.Query(ctx, `
		SELECT photo_key FROM face_enrollment_photos
		WHERE created_at < NOW() - $1::interval
	`, intervalStr)
	if err != nil {
		return 0, fmt.Errorf("clean staging: query photos: %w", err)
	}
	defer rows.Close()

	deleted := 0
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			continue
		}
		if err := c.store.Delete(ctx, key); err == nil {
			deleted++
		}
	}
	return deleted, nil
}

// CleanOrphans lists storage objects under face/ and removes any not present in face_references.
func (c *OrphanCleaner) CleanOrphans(ctx context.Context) (int, error) {
	keys, err := c.store.List(ctx, "face/")
	if err != nil {
		return 0, fmt.Errorf("clean orphans: list keys: %w", err)
	}

	deleted := 0
	for _, key := range keys {
		if !strings.HasPrefix(key, "face/") || !strings.HasSuffix(key, ".jpg") {
			continue
		}
		var exists bool
		err := c.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM face_references WHERE photo_key = $1)`, key).Scan(&exists)
		if err != nil {
			continue
		}
		if !exists {
			if err := c.store.Delete(ctx, key); err == nil {
				deleted++
				if c.logger != nil {
					c.logger.Info("storage: cleaned orphan file", slog.String("key", key))
				}
			}
		}
	}
	return deleted, nil
}
