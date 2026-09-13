package attendance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ComputeWorkDate determines the canonical work date for attendance.
// If the local hour is before cutoffHour (e.g. 04:00 AM), the shift belongs
// to the previous calendar day.
func ComputeWorkDate(t time.Time, loc *time.Location, cutoffHour int) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	localT := t.In(loc)
	if localT.Hour() < cutoffHour {
		localT = localT.AddDate(0, 0, -1)
	}
	return time.Date(localT.Year(), localT.Month(), localT.Day(), 0, 0, 0, 0, loc)
}

// ComputePhotoSHA256 returns the hexadecimal lowercase SHA-256 hash of the photo bytes.
func ComputePhotoSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ParseTimeString parses "HH:MM" into hour and minute.
func ParseTimeString(s string) (hour, minute int, err error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid time format: %q", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, 0, fmt.Errorf("invalid hour in %q", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid minute in %q", s)
	}
	return h, m, nil
}

// EvaluateLate evaluates whether a check-in is late compared to work_start_time + tolerance.
func EvaluateLate(checkInTime time.Time, loc *time.Location, workStartTimeStr string, toleranceMinutes int) (isLate bool, lateMinutes int) {
	if loc == nil {
		loc = time.UTC
	}
	h, m, err := ParseTimeString(workStartTimeStr)
	if err != nil {
		h, m = 8, 0 // fallback 08:00
	}

	localT := checkInTime.In(loc)
	sched := time.Date(localT.Year(), localT.Month(), localT.Day(), h, m, 0, 0, loc)
	deadline := sched.Add(time.Duration(toleranceMinutes) * time.Minute)

	if localT.After(deadline) {
		diff := localT.Sub(sched)
		minutes := int(math.Ceil(diff.Minutes()))
		if minutes < 0 {
			minutes = 0
		}
		return true, minutes
	}
	return false, 0
}

// EvaluateEarlyLeave evaluates whether a check-out is early compared to work_end_time - tolerance.
func EvaluateEarlyLeave(checkOutTime time.Time, loc *time.Location, workEndTimeStr string, toleranceMinutes int) (isEarly bool, earlyMinutes int) {
	if loc == nil {
		loc = time.UTC
	}
	h, m, err := ParseTimeString(workEndTimeStr)
	if err != nil {
		h, m = 17, 0 // fallback 17:00
	}

	localT := checkOutTime.In(loc)
	sched := time.Date(localT.Year(), localT.Month(), localT.Day(), h, m, 0, 0, loc)
	earliestAllowed := sched.Add(-time.Duration(toleranceMinutes) * time.Minute)

	if localT.Before(earliestAllowed) {
		diff := sched.Sub(localT)
		minutes := int(math.Ceil(diff.Minutes()))
		if minutes < 0 {
			minutes = 0
		}
		return true, minutes
	}
	return false, 0
}

// EvaluateClockSkew calculates skew in seconds between client reported time and server time.
// skew = serverTime - clientTime
func EvaluateClockSkew(clientTime, serverTime time.Time, maxSkewSeconds int) (skewSeconds int, isSkewed bool) {
	diff := serverTime.Sub(clientTime).Seconds()
	skewSeconds = int(math.Round(diff))
	if math.Abs(diff) > float64(maxSkewSeconds) {
		return skewSeconds, true
	}
	return skewSeconds, false
}
