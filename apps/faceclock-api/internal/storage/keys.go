package storage

import (
	"fmt"

	"github.com/google/uuid"
)

// ReferencePhotoKey returns the storage key for a permanent face reference photo:
// face/{employee_id}/{face_reference_id}.jpg
func ReferencePhotoKey(employeeID, refID uuid.UUID) string {
	return fmt.Sprintf("face/%s/%s.jpg", employeeID, refID)
}

// StagingPhotoKey returns the storage key for an enrollment session staging photo:
// face-staging/{session_id}/{photo_id}.jpg
func StagingPhotoKey(sessionID, photoID uuid.UUID) string {
	return fmt.Sprintf("face-staging/%s/%s.jpg", sessionID, photoID)
}

// StagingSessionPrefix returns the storage key prefix for an enrollment session's staging photos:
// face-staging/{session_id}/
func StagingSessionPrefix(sessionID uuid.UUID) string {
	return fmt.Sprintf("face-staging/%s/", sessionID)
}
