package face

import (
	"context"
)

// FaceEngine defines the modular interface for face detection, feature extraction,
// and metric distance computation across Fase 2 and later phases.
type FaceEngine interface {
	// DetectAndEmbed processes an image in memory, extracting bounding box, quality,
	// and a normalized 512-d (or 128-d) vector embedding.
	DetectAndEmbed(ctx context.Context, imageBytes []byte) (*DetectionResult, error)

	// Compare evaluates two vector embeddings, returning both cosine similarity [ -1..1 ]
	// and euclidean distance [ 0..inf ).
	Compare(v1, v2 []float32) (similarity float32, distance float32)

	// IsMatch determines whether a similarity score meets or exceeds the engine's threshold.
	IsMatch(similarity float32) bool

	// Threshold returns the operational similarity threshold for face match decisions.
	Threshold() float32

	// ModelVersion returns the model identifier string (e.g. "buffalo_l" or "arcface_r50").
	ModelVersion() string
}

// BoundingBox holds the coordinates of the detected face in pixel space.
type BoundingBox struct {
	X1 float32 `json:"x1"`
	Y1 float32 `json:"y1"`
	X2 float32 `json:"x2"`
	Y2 float32 `json:"y2"`
}

// QualityMetrics contains granular image and face quality parameters evaluated by the engine.
type QualityMetrics struct {
	DetScore       float32 `json:"det_score"`
	Blur           float32 `json:"blur"`
	Brightness     float32 `json:"brightness"`
	Yaw            float32 `json:"yaw"`
	Pitch          float32 `json:"pitch"`
	FaceRatio      float32 `json:"face_ratio"`
	FaceCount      int     `json:"face_count"`
	OcclusionRatio float32 `json:"occlusion_ratio"`
}

// DetectionResult is the standard output produced after facial analysis.
type DetectionResult struct {
	BoundingBox   *BoundingBox    `json:"bbox,omitempty"`
	Embedding     []float32       `json:"embedding"`
	DetScore      float32         `json:"det_score"`
	Quality       *QualityMetrics `json:"quality,omitempty"`
	QualityScore  float32         `json:"quality_score"`
	LivenessScore *float32        `json:"liveness_score,omitempty"`
	IsUsable      bool            `json:"is_usable"`
	ModelVersion  string          `json:"model_version"`
	Hints         []string        `json:"hints,omitempty"`
}
