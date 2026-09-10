package storage

import (
	"testing"

	"github.com/google/uuid"
)

func TestStorageKeys(t *testing.T) {
	empID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	refID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	sessID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	photoID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	expectedRef := "face/11111111-1111-1111-1111-111111111111/22222222-2222-2222-2222-222222222222.jpg"
	if got := ReferencePhotoKey(empID, refID); got != expectedRef {
		t.Errorf("ReferencePhotoKey got %q, want %q", got, expectedRef)
	}

	expectedStaging := "face-staging/33333333-3333-3333-3333-333333333333/44444444-4444-4444-4444-444444444444.jpg"
	if got := StagingPhotoKey(sessID, photoID); got != expectedStaging {
		t.Errorf("StagingPhotoKey got %q, want %q", got, expectedStaging)
	}

	expectedPrefix := "face-staging/33333333-3333-3333-3333-333333333333/"
	if got := StagingSessionPrefix(sessID); got != expectedPrefix {
		t.Errorf("StagingSessionPrefix got %q, want %q", got, expectedPrefix)
	}
}
