package rbac

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type cacheEntry struct {
	roles       []string
	permissions map[string]struct{}
	expiresAt   time.Time
}

// Cache holds an in-memory cache of user roles and permissions with TTL.
type Cache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*cacheEntry
	ttl     time.Duration
}

// NewCache constructs an RBAC cache with the specified TTL (default 30s).
func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Cache{
		entries: make(map[uuid.UUID]*cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves cached roles and permissions for a user if present and not expired.
func (c *Cache) Get(userID uuid.UUID) ([]string, map[string]struct{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[userID]
	if !ok || time.Now().UTC().After(entry.expiresAt) {
		return nil, nil, false
	}

	// Return copies or references
	return entry.roles, entry.permissions, true
}

// Set stores roles and permissions in cache for a user.
func (c *Cache) Set(userID uuid.UUID, roles []string, permissions map[string]struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[userID] = &cacheEntry{
		roles:       roles,
		permissions: permissions,
		expiresAt:   time.Now().UTC().Add(c.ttl),
	}
}

// InvalidateUser purges cache for a specific user.
func (c *Cache) InvalidateUser(userID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, userID)
}

// InvalidateAll clears the entire permission cache.
func (c *Cache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[uuid.UUID]*cacheEntry)
}
