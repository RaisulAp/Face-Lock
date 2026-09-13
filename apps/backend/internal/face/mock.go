package face

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"sync"
)

// MockEngine provides a deterministic, in-memory implementation of FaceEngine
// for integration tests and environments without active Python inference engine.
type MockEngine struct {
	mu           sync.RWMutex
	threshold    float32
	modelVersion string
	forceError   error
	forceResult  *DetectionResult
}

// NewMockEngine creates a MockEngine initialized with standard defaults.
func NewMockEngine() *MockEngine {
	return &MockEngine{
		threshold:    DefaultCosineThreshold,
		modelVersion: "buffalo_l",
	}
}

// SetThreshold allows overriding the match threshold.
func (m *MockEngine) SetThreshold(t float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.threshold = t
}

// SetForceError configures an error to be returned by next DetectAndEmbed calls.
func (m *MockEngine) SetForceError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceError = err
}

// SetForceResult configures an exact DetectionResult to be returned by DetectAndEmbed.
func (m *MockEngine) SetForceResult(res *DetectionResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceResult = res
}

func (m *MockEngine) Threshold() float32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.threshold
}

func (m *MockEngine) ModelVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.modelVersion
}

func (m *MockEngine) IsMatch(similarity float32) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return similarity >= m.threshold
}

func (m *MockEngine) Compare(v1, v2 []float32) (float32, float32) {
	sim, err := CosineSimilarity(v1, v2)
	if err != nil {
		sim = 0
	}
	dist, err := EuclideanDistance(v1, v2)
	if err != nil {
		dist = math.MaxFloat32
	}
	return sim, dist
}

// DetectAndEmbed generates a deterministic normalized 512-d vector from image payload,
// unless configured with forceError or forceResult.
func (m *MockEngine) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*DetectionResult, error) {
	m.mu.RLock()
	err := m.forceError
	res := m.forceResult
	ver := m.modelVersion
	m.mu.RUnlock()

	if err != nil {
		return nil, err
	}
	if res != nil {
		return res, nil
	}
	if len(imageBytes) == 0 {
		return nil, ErrEmptyImagePayload
	}

	// Deterministic pseudo-embedding from SHA-256 hash of payload
	hash := sha256.Sum256(imageBytes)
	raw := make([]float32, DefaultEmbeddingDimension)

	for i := 0; i < DefaultEmbeddingDimension; i++ {
		// Use rolling slices of the 32-byte hash
		byteOffset := (i * 4) % (len(hash) - 4)
		seedVal := binary.LittleEndian.Uint32(hash[byteOffset : byteOffset+4])
		// Center in [-1, 1]
		raw[i] = (float32(seedVal%10000) / 5000.0) - 1.0
	}

	normalized := NormalizeL2(raw)
	liveness := float32(0.98)

	return &DetectionResult{
		BoundingBox: &BoundingBox{
			X1: 50.0,
			Y1: 50.0,
			X2: 250.0,
			Y2: 250.0,
		},
		Embedding: normalized,
		DetScore:  0.96,
		Quality: &QualityMetrics{
			DetScore:       0.96,
			Blur:           0.85,
			Brightness:     0.75,
			Yaw:            2.5,
			Pitch:          -1.2,
			FaceRatio:      0.35,
			FaceCount:      1,
			OcclusionRatio: 0.05,
		},
		QualityScore:  0.92,
		LivenessScore: &liveness,
		IsUsable:      true,
		ModelVersion:  ver,
		Hints:         []string{},
	}, nil
}
