package consent

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCache_GetSetInvalidate(t *testing.T) {
	c := NewCache(100 * time.Millisecond)
	empID := uuid.New()

	// Initial get -> false, false
	has, ok := c.Get(empID)
	if ok || has {
		t.Fatalf("expected miss, got ok=%v, has=%v", ok, has)
	}

	// Set true
	c.Set(empID, true)
	has, ok = c.Get(empID)
	if !ok || !has {
		t.Fatalf("expected hit with true, got ok=%v, has=%v", ok, has)
	}

	// Invalidate
	c.Invalidate(empID)
	has, ok = c.Get(empID)
	if ok || has {
		t.Fatalf("expected miss after invalidate, got ok=%v, has=%v", ok, has)
	}

	// Set false (withdrawn)
	c.Set(empID, false)
	has, ok = c.Get(empID)
	if !ok || has {
		t.Fatalf("expected hit with false, got ok=%v, has=%v", ok, has)
	}

	// Wait for TTL expiration
	time.Sleep(150 * time.Millisecond)
	has, ok = c.Get(empID)
	if ok {
		t.Fatalf("expected expired entry to miss, got ok=%v", ok)
	}
}
