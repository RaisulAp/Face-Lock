package face_test

import (
	"context"
	"math"
	"testing"

	"github.com/faceclock/faceclock/apps/faceclock-api/internal/face"
)

func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors have similarity 1.0", func(t *testing.T) {
		v1 := []float32{1.0, 2.0, 3.0}
		v2 := []float32{1.0, 2.0, 3.0}
		sim, err := face.CosineSimilarity(v1, v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(float64(sim-1.0)) > 1e-5 {
			t.Fatalf("expected similarity ~1.0, got %f", sim)
		}
	})

	t.Run("orthogonal vectors have similarity 0.0", func(t *testing.T) {
		v1 := []float32{1.0, 0.0, 0.0}
		v2 := []float32{0.0, 1.0, 0.0}
		sim, err := face.CosineSimilarity(v1, v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(float64(sim)) > 1e-5 {
			t.Fatalf("expected similarity 0.0, got %f", sim)
		}
	})

	t.Run("opposite vectors have similarity -1.0", func(t *testing.T) {
		v1 := []float32{1.0, 2.0, 3.0}
		v2 := []float32{-1.0, -2.0, -3.0}
		sim, err := face.CosineSimilarity(v1, v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(float64(sim-(-1.0))) > 1e-5 {
			t.Fatalf("expected similarity -1.0, got %f", sim)
		}
	})

	t.Run("dimension mismatch returns error", func(t *testing.T) {
		v1 := []float32{1.0, 2.0}
		v2 := []float32{1.0, 2.0, 3.0}
		_, err := face.CosineSimilarity(v1, v2)
		if err != face.ErrDimensionMismatch {
			t.Fatalf("expected ErrDimensionMismatch, got %v", err)
		}
	})

	t.Run("empty vector returns error", func(t *testing.T) {
		_, err := face.CosineSimilarity([]float32{}, []float32{})
		if err != face.ErrEmptyVector {
			t.Fatalf("expected ErrEmptyVector, got %v", err)
		}
	})
}

func TestEuclideanDistance(t *testing.T) {
	t.Run("identical vectors have distance 0.0", func(t *testing.T) {
		v1 := []float32{3.0, 4.0}
		v2 := []float32{3.0, 4.0}
		dist, err := face.EuclideanDistance(v1, v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dist != 0.0 {
			t.Fatalf("expected distance 0.0, got %f", dist)
		}
	})

	t.Run("known 3-4-5 triangle distance", func(t *testing.T) {
		v1 := []float32{0.0, 0.0}
		v2 := []float32{3.0, 4.0}
		dist, err := face.EuclideanDistance(v1, v2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if math.Abs(float64(dist-5.0)) > 1e-5 {
			t.Fatalf("expected distance 5.0, got %f", dist)
		}
	})

	t.Run("dimension mismatch returns error", func(t *testing.T) {
		_, err := face.EuclideanDistance([]float32{1.0}, []float32{1.0, 2.0})
		if err != face.ErrDimensionMismatch {
			t.Fatalf("expected ErrDimensionMismatch, got %v", err)
		}
	})
}

func TestNormalizeL2(t *testing.T) {
	v := []float32{3.0, 4.0}
	norm := face.NormalizeL2(v)
	if len(norm) != 2 {
		t.Fatalf("expected length 2, got %d", len(norm))
	}
	if math.Abs(float64(norm[0]-0.6)) > 1e-5 || math.Abs(float64(norm[1]-0.8)) > 1e-5 {
		t.Fatalf("expected [0.6, 0.8], got %v", norm)
	}

	// Length of normalized vector is 1.0
	length := math.Sqrt(float64(norm[0]*norm[0] + norm[1]*norm[1]))
	if math.Abs(length-1.0) > 1e-5 {
		t.Fatalf("expected unit length, got %f", length)
	}
}

func TestVectorPGVectorRoundtrip(t *testing.T) {
	orig := []float32{0.123456, -0.654321, 0.999999}
	s := face.VectorToPGVector(orig)
	parsed, err := face.PGVectorToVector(s)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if len(parsed) != len(orig) {
		t.Fatalf("expected length %d, got %d", len(orig), len(parsed))
	}
	for i := range orig {
		if math.Abs(float64(parsed[i]-orig[i])) > 1e-5 {
			t.Errorf("mismatch at index %d: orig %f, parsed %f", i, orig[i], parsed[i])
		}
	}
}

func TestMockEngine(t *testing.T) {
	engine := face.NewMockEngine()

	t.Run("deterministic embeddings from identical images", func(t *testing.T) {
		img1 := []byte("employee-1-face-photo-data")
		img2 := []byte("employee-1-face-photo-data")
		img3 := []byte("employee-2-face-photo-data")

		res1, err := engine.DetectAndEmbed(context.Background(), img1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res2, err := engine.DetectAndEmbed(context.Background(), img2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		res3, err := engine.DetectAndEmbed(context.Background(), img3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(res1.Embedding) != 512 {
			t.Fatalf("expected 512-dim embedding, got %d", len(res1.Embedding))
		}

		simSame, distSame := engine.Compare(res1.Embedding, res2.Embedding)
		if simSame < 0.9999 || distSame > 0.0001 {
			t.Fatalf("identical images must have similarity ~1.0 and distance ~0, got sim=%f, dist=%f", simSame, distSame)
		}
		if !engine.IsMatch(simSame) {
			t.Fatalf("expected match for identical images")
		}

		simDiff, distDiff := engine.Compare(res1.Embedding, res3.Embedding)
		if simDiff >= 0.75 {
			t.Fatalf("different images should have distinct embeddings, got sim=%f", simDiff)
		}
		_ = distDiff
	})

	t.Run("empty image returns ErrEmptyImagePayload", func(t *testing.T) {
		_, err := engine.DetectAndEmbed(context.Background(), nil)
		if err != face.ErrEmptyImagePayload {
			t.Fatalf("expected ErrEmptyImagePayload, got %v", err)
		}
	})

	t.Run("forced error returns immediately", func(t *testing.T) {
		engine.SetForceError(face.ErrNoFaceDetected)
		_, err := engine.DetectAndEmbed(context.Background(), []byte("some-face"))
		if err != face.ErrNoFaceDetected {
			t.Fatalf("expected ErrNoFaceDetected, got %v", err)
		}
		// Reset
		engine.SetForceError(nil)
	})
}
