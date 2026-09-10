package face

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// DefaultCosineThreshold is the default similarity cutoff for positive identity match (>= 0.75).
	DefaultCosineThreshold float32 = 0.75

	// DefaultEuclideanThreshold is the distance cutoff for normalized embeddings (<= 0.60).
	DefaultEuclideanThreshold float32 = 0.60

	// DefaultEmbeddingDimension is the standard dimension for ArcFace / InsightFace buffalo_l models.
	DefaultEmbeddingDimension = 512
)

// CosineSimilarity computes the cosine similarity between two float vectors.
// If vectors are already L2 normalized, it equals their dot product.
// Returns a value in [-1.0, 1.0].
func CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, ErrEmptyVector
	}
	if len(a) != len(b) {
		return 0, ErrDimensionMismatch
	}

	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0, nil
	}

	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	// Clamp rounding errors
	if sim > 1.0 {
		sim = 1.0
	} else if sim < -1.0 {
		sim = -1.0
	}

	return float32(sim), nil
}

// EuclideanDistance computes the L2 euclidean distance between two float vectors.
// Returns a non-negative float value.
func EuclideanDistance(a, b []float32) (float32, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, ErrEmptyVector
	}
	if len(a) != len(b) {
		return 0, ErrDimensionMismatch
	}

	var sumSq float64
	for i := 0; i < len(a); i++ {
		diff := float64(a[i]) - float64(b[i])
		sumSq += diff * diff
	}

	return float32(math.Sqrt(sumSq)), nil
}

// NormalizeL2 normalizes a vector in-place or returns a new L2-normalized vector.
func NormalizeL2(v []float32) []float32 {
	if len(v) == 0 {
		return v
	}

	var sumSq float64
	for _, val := range v {
		sumSq += float64(val) * float64(val)
	}

	norm := math.Sqrt(sumSq)
	if norm == 0 {
		return v
	}

	res := make([]float32, len(v))
	for i, val := range v {
		res[i] = float32(float64(val) / norm)
	}
	return res
}

// VectorToPGVector formats a float32 slice as a Postgres pgvector string: "[v0,v1,...,vn]".
func VectorToPGVector(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, val := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(val), 'f', 6, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}

// PGVectorToVector parses a Postgres pgvector string "[v0,v1,...,vn]" back to []float32.
func PGVectorToVector(s string) ([]float32, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil, nil
	}
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil, fmt.Errorf("invalid pgvector literal format: %s", s)
	}

	content := s[1 : len(s)-1]
	parts := strings.Split(content, ",")
	res := make([]float32, len(parts))

	for i, part := range parts {
		val, err := strconv.ParseFloat(strings.TrimSpace(part), 32)
		if err != nil {
			return nil, fmt.Errorf("invalid float at index %d (%q): %w", i, part, err)
		}
		res[i] = float32(val)
	}

	return res, nil
}
