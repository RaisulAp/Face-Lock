package attendance

import (
	"testing"
	"time"
)

func TestComputeWorkDate(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	// Normal time at 09:00 -> same day
	t1 := time.Date(2026, 9, 15, 9, 0, 0, 0, loc)
	w1 := ComputeWorkDate(t1, loc, 4)
	if w1.Format("2006-01-02") != "2026-09-15" {
		t.Errorf("Expected 2026-09-15, got %s", w1.Format("2006-01-02"))
	}

	// Early morning before cutoff at 03:30 -> previous day
	t2 := time.Date(2026, 9, 15, 3, 30, 0, 0, loc)
	w2 := ComputeWorkDate(t2, loc, 4)
	if w2.Format("2006-01-02") != "2026-09-14" {
		t.Errorf("Expected 2026-09-14, got %s", w2.Format("2006-01-02"))
	}

	// Exactly at cutoff at 04:00 -> current day
	t3 := time.Date(2026, 9, 15, 4, 0, 0, 0, loc)
	w3 := ComputeWorkDate(t3, loc, 4)
	if w3.Format("2006-01-02") != "2026-09-15" {
		t.Errorf("Expected 2026-09-15, got %s", w3.Format("2006-01-02"))
	}
}

func TestEvaluateLate(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)

	// Check-in at 08:10 with 15m tolerance -> not late
	t1 := time.Date(2026, 9, 15, 8, 10, 0, 0, loc)
	isLate, lateMins := EvaluateLate(t1, loc, "08:00", 15)
	if isLate || lateMins != 0 {
		t.Errorf("Expected not late, got isLate=%v lateMins=%d", isLate, lateMins)
	}

	// Check-in at 08:20 with 15m tolerance -> late by 20m from start
	t2 := time.Date(2026, 9, 15, 8, 20, 0, 0, loc)
	isLate, lateMins = EvaluateLate(t2, loc, "08:00", 15)
	if !isLate || lateMins != 20 {
		t.Errorf("Expected late by 20m, got isLate=%v lateMins=%d", isLate, lateMins)
	}

	// Check-in at 07:55 -> not late
	t3 := time.Date(2026, 9, 15, 7, 55, 0, 0, loc)
	isLate, lateMins = EvaluateLate(t3, loc, "08:00", 15)
	if isLate || lateMins != 0 {
		t.Errorf("Expected not late, got isLate=%v lateMins=%d", isLate, lateMins)
	}
}

func TestEvaluateEarlyLeave(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)

	// Checkout at 17:05 with 15m tolerance -> not early
	t1 := time.Date(2026, 9, 15, 17, 5, 0, 0, loc)
	isEarly, earlyMins := EvaluateEarlyLeave(t1, loc, "17:00", 15)
	if isEarly || earlyMins != 0 {
		t.Errorf("Expected not early, got isEarly=%v earlyMins=%d", isEarly, earlyMins)
	}

	// Checkout at 16:50 with 15m tolerance -> not early (within tolerance)
	t2 := time.Date(2026, 9, 15, 16, 50, 0, 0, loc)
	isEarly, earlyMins = EvaluateEarlyLeave(t2, loc, "17:00", 15)
	if isEarly || earlyMins != 0 {
		t.Errorf("Expected not early, got isEarly=%v earlyMins=%d", isEarly, earlyMins)
	}

	// Checkout at 16:30 with 15m tolerance -> early leave by 30 mins
	t3 := time.Date(2026, 9, 15, 16, 30, 0, 0, loc)
	isEarly, earlyMins = EvaluateEarlyLeave(t3, loc, "17:00", 15)
	if !isEarly || earlyMins != 30 {
		t.Errorf("Expected early by 30m, got isEarly=%v earlyMins=%d", isEarly, earlyMins)
	}
}

func TestEvaluateClockSkew(t *testing.T) {
	now := time.Now()

	// 10s skew within 300s
	c1 := now.Add(-10 * time.Second)
	skew, isSkewed := EvaluateClockSkew(c1, now, 300)
	if isSkewed || skew != 10 {
		t.Errorf("Expected skew=10, isSkewed=false, got skew=%d, isSkewed=%v", skew, isSkewed)
	}

	// 400s skew exceeds 300s
	c2 := now.Add(-400 * time.Second)
	skew, isSkewed = EvaluateClockSkew(c2, now, 300)
	if !isSkewed || skew != 400 {
		t.Errorf("Expected skew=400, isSkewed=true, got skew=%d, isSkewed=%v", skew, isSkewed)
	}
}

func TestComputePhotoSHA256(t *testing.T) {
	data := []byte("faceclock-attendance-test-photo")
	hash := ComputePhotoSHA256(data)
	if len(hash) != 64 {
		t.Errorf("Expected SHA-256 hash length 64, got %d", len(hash))
	}
}
