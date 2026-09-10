package face

import "errors"

var (
	// ErrNoFaceDetected is returned when the engine detects zero faces in the input image.
	ErrNoFaceDetected = errors.New("face: no face detected in image")

	// ErrMultipleFaces is returned when more than one face is detected in the input image.
	ErrMultipleFaces = errors.New("face: multiple faces detected in image")

	// ErrFaceNotUsable is returned when detected face fails quality checks (blur, pose angle, brightness, etc.).
	ErrFaceNotUsable = errors.New("face: face image quality below acceptable threshold")

	// ErrDimensionMismatch is returned when comparing vectors of unequal length.
	ErrDimensionMismatch = errors.New("face: embedding dimension mismatch")

	// ErrEmptyVector is returned when an embedding slice has length 0.
	ErrEmptyVector = errors.New("face: empty vector provided")

	// ErrEmptyImagePayload is returned when image byte slice is empty.
	ErrEmptyImagePayload = errors.New("face: empty image payload")
)
