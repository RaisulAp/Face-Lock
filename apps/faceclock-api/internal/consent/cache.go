package consent

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type consentCacheEntry struct {
	hasConsent bool
	expiresAt  time.Time
}

type Cache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*consentCacheEntry
	ttl     time.Duration
}

func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &Cache{
		entries: make(map[uuid.UUID]*consentCacheEntry),
		ttl:     ttl,
	}
}

func (c *Cache) Get(employeeID uuid.UUID) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[employeeID]
	if !ok || time.Now().UTC().After(entry.expiresAt) {
		return false, false
	}
	return entry.hasConsent, true
}

func (c *Cache) Set(employeeID uuid.UUID, hasConsent bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[employeeID] = &consentCacheEntry{
		hasConsent: hasConsent,
		expiresAt:  time.Now().UTC().Add(c.ttl),
	}
}

func (c *Cache) Invalidate(employeeID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, employeeID)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[uuid.UUID]*consentCacheEntry)
}
