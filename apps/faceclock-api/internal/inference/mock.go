package inference

import (
	"context"
	"crypto/sha256"
	"math"
	"sync"
)

// DeterministicUnitVector generates an L2-normalized 512-dim vector from bytes.
func DeterministicUnitVector(seed []byte) []float32 {
	h := sha256.Sum256(seed)
	emb := make([]float32, 512)
	var sumSq float64
	for i := 0; i < 512; i++ {
		v := float32(int8(h[i%32])) + float32((i*7)%31) - 15.0
		if v == 0 {
			v = 1.0
		}
		emb[i] = v
		sumSq += float64(v * v)
	}
	norm := float32(math.Sqrt(sumSq))
	for i := range emb {
		emb[i] /= norm
	}
	return emb
}

// MockFaceEngine provides a mock implementation of FaceEngine for testing.
type MockFaceEngine struct {
	mu              sync.RWMutex
	ModelVer        string
	HealthStat      string
	DefaultEmbed    []float32
	DeriveFromImage bool
	EmbedFn         func(ctx context.Context, imageBytes []byte) (*EmbedResult, error)
	DetectErr       error
	EmbedErr        error
	HealthErr       error
	ReadyErr        error
	CustomReadyData *ReadyData
}

// NewMockFaceEngine creates a MockFaceEngine with buffalo_l default 512-dim embedding.
func NewMockFaceEngine() *MockFaceEngine {
	return &MockFaceEngine{
		ModelVer:        "buffalo_l@v1",
		HealthStat:      "ok",
		DeriveFromImage: true,
	}
}

func (m *MockFaceEngine) SetModelVersion(v string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ModelVer = v
}

func (m *MockFaceEngine) SetDetectError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DetectErr = err
}

func (m *MockFaceEngine) SetDefaultEmbedding(emb []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DefaultEmbed = emb
	m.DeriveFromImage = false
}

func (m *MockFaceEngine) SetEmbedFn(fn func(ctx context.Context, imageBytes []byte) (*EmbedResult, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.EmbedFn = fn
}

func (m *MockFaceEngine) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*EmbedResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.DetectErr != nil {
		return nil, m.DetectErr
	}
	if m.EmbedFn != nil {
		return m.EmbedFn(ctx, imageBytes)
	}

	emb := m.DefaultEmbed
	if emb == nil || m.DeriveFromImage {
		emb = DeterministicUnitVector(imageBytes)
	}

	return &EmbedResult{
		Embedding:    emb,
		DetScore:     0.95,
		QualityScore: 0.92,
		Usable:       true,
		ModelVersion: m.ModelVer,
	}, nil
}

func (m *MockFaceEngine) EmbedBatch(ctx context.Context, images [][]byte) (*BatchEmbedResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.EmbedErr != nil {
		return nil, m.EmbedErr
	}
	items := make([]BatchEmbedItem, len(images))
	for i, img := range images {
		emb := m.DefaultEmbed
		if emb == nil || m.DeriveFromImage {
			emb = DeterministicUnitVector(img)
		}
		items[i] = BatchEmbedItem{
			Index: i,
			Result: &EmbedResult{
				Embedding:    emb,
				DetScore:     0.95,
				QualityScore: 0.92,
				ModelVersion: m.ModelVer,
				Usable:       true,
			},
		}
	}
	return &BatchEmbedResult{
		Items:        items,
		ModelVersion: m.ModelVer,
		TookMs:       10,
	}, nil
}

func (m *MockFaceEngine) Health(ctx context.Context) (*HealthStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.HealthErr != nil {
		return nil, m.HealthErr
	}
	return &HealthStatus{
		Status:       m.HealthStat,
		ModelVersion: m.ModelVer,
	}, nil
}

func (m *MockFaceEngine) Ready(ctx context.Context) (*ReadyData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.ReadyErr != nil {
		return nil, m.ReadyErr
	}
	if m.CustomReadyData != nil {
		return m.CustomReadyData, nil
	}
	return &ReadyData{
		Status:       "ok",
		ModelName:    "buffalo_l",
		ModelVersion: m.ModelVer,
		EmbeddingDim: 512,
		QualityThresholds: QualityThresholds{
			MinDetScore:   0.60,
			MinBlurVar:    40.0,
			MinBrightness: 55.0,
			MaxBrightness: 215.0,
			MinFaceRatio:  0.18,
			MaxAbsYaw:     0.35,
			MaxAbsPitch:   0.30,
		},
	}, nil
}
