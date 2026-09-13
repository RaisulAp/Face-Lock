// Package inference is the HTTP client faceclock-api uses to talk to
// faceclock-inference.
package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

type HealthStatus struct {
	Status       string `json:"status"`
	ModelVersion string `json:"model_version"`
	Stub         bool   `json:"stub"`
}

type QualityInfo struct {
	DetScore       float32 `json:"det_score"`
	Blur           float32 `json:"blur"`
	Brightness     float32 `json:"brightness"`
	Yaw            float32 `json:"yaw"`
	Pitch          float32 `json:"pitch"`
	FaceRatio      float32 `json:"face_ratio"`
	FaceCount      int     `json:"face_count"`
	OcclusionRatio float32 `json:"occlusion_ratio"`
}

type EmbedResult struct {
	Embedding    []float32    `json:"embedding"`
	BBox         []float32    `json:"bbox"`
	DetScore     float32      `json:"det_score"`
	Quality      *QualityInfo `json:"quality"`
	QualityScore float32      `json:"quality_score"`
	Hints        []string     `json:"hints"`
	Usable       bool         `json:"usable"`
	ModelVersion string       `json:"model_version"`
	TookMs       int          `json:"took_ms"`
}

type BatchEmbedItem struct {
	Index  int          `json:"index"`
	Result *EmbedResult `json:"result,omitempty"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type BatchEmbedResult struct {
	Items        []BatchEmbedItem `json:"items"`
	ModelVersion string           `json:"model_version"`
	TookMs       int              `json:"took_ms"`
}

type QualityThresholds struct {
	MinDetScore   float64 `json:"min_det_score"`
	MinBlurVar    float64 `json:"min_blur_var"`
	MinBrightness float64 `json:"min_brightness"`
	MaxBrightness float64 `json:"max_brightness"`
	MinFaceRatio  float64 `json:"min_face_ratio"`
	MaxAbsYaw     float64 `json:"max_abs_yaw"`
	MaxAbsPitch   float64 `json:"max_abs_pitch"`
}

type ReadyData struct {
	Status            string            `json:"status"`
	ModelName         string            `json:"model_name"`
	ModelVersion      string            `json:"model_version"`
	EmbeddingDim      int               `json:"embedding_dim"`
	QualityThresholds QualityThresholds `json:"quality_thresholds"`
}

type FaceEngine interface {
	DetectAndEmbed(ctx context.Context, imageBytes []byte) (*EmbedResult, error)
	EmbedBatch(ctx context.Context, images [][]byte) (*BatchEmbedResult, error)
	Health(ctx context.Context) (*HealthStatus, error)
	Ready(ctx context.Context) (*ReadyData, error)
}

// Client talks to faceclock-inference over the internal compose network.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New builds a Client with the given request timeout (INFERENCE_TIMEOUT_MS).
func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

var _ FaceEngine = (*Client)(nil)

func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("inference: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inference: unexpected status %d", resp.StatusCode)
	}

	var status HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("inference: decode response: %w", err)
	}
	return &status, nil
}

func (c *Client) Ready(ctx context.Context) (*ReadyData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/ready", nil)
	if err != nil {
		return nil, fmt.Errorf("inference: build ready request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference: ready request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inference: unexpected ready status %d", resp.StatusCode)
	}

	var envelope struct {
		Data ReadyData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("inference: decode ready response: %w", err)
	}
	return &envelope.Data, nil
}

func (c *Client) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*EmbedResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", "face.jpg")
	if err != nil {
		return nil, fmt.Errorf("create multipart form file: %w", err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		return nil, fmt.Errorf("write image bytes: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/embed", &body)
	if err != nil {
		return nil, fmt.Errorf("create embed request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embed endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed endpoint returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var envelope struct {
		Data EmbedResult `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal embed response: %w", err)
	}

	return &envelope.Data, nil
}

func (c *Client) EmbedBatch(ctx context.Context, images [][]byte) (*BatchEmbedResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for i, img := range images {
		part, err := writer.CreateFormFile("images", fmt.Sprintf("face_%d.jpg", i))
		if err != nil {
			return nil, fmt.Errorf("create multipart file: %w", err)
		}
		if _, err := part.Write(img); err != nil {
			return nil, fmt.Errorf("write image bytes: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/embed-batch", &body)
	if err != nil {
		return nil, fmt.Errorf("create embed-batch request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embed-batch endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed-batch returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var envelope struct {
		Data BatchEmbedResult `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal batch response: %w", err)
	}

	return &envelope.Data, nil
}

// FakeClient is useful for testing without needing the real Python inference service.
type FakeClient struct {
	CustomEmbedFn func(ctx context.Context, imageBytes []byte) (*EmbedResult, error)
	CustomBatchFn func(ctx context.Context, images [][]byte) (*BatchEmbedResult, error)
	HealthFn      func(ctx context.Context) (*HealthStatus, error)
	ReadyFn       func(ctx context.Context) (*ReadyData, error)
}

func NewFakeClient() *FakeClient {
	return &FakeClient{}
}

func (f *FakeClient) SetOverride(fn func(imageBytes []byte) (*EmbedResult, error)) {
	if fn == nil {
		f.CustomEmbedFn = nil
		return
	}
	f.CustomEmbedFn = func(ctx context.Context, imageBytes []byte) (*EmbedResult, error) {
		return fn(imageBytes)
	}
}

var _ FaceEngine = (*FakeClient)(nil)

func (f *FakeClient) Health(ctx context.Context) (*HealthStatus, error) {
	if f.HealthFn != nil {
		return f.HealthFn(ctx)
	}
	return &HealthStatus{Status: "ok", ModelVersion: "buffalo_l", Stub: false}, nil
}

func (f *FakeClient) Ready(ctx context.Context) (*ReadyData, error) {
	if f.ReadyFn != nil {
		return f.ReadyFn(ctx)
	}
	return &ReadyData{
		Status:       "ok",
		ModelName:    "buffalo_l",
		ModelVersion: "buffalo_l@v1",
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

func (f *FakeClient) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*EmbedResult, error) {
	if f.CustomEmbedFn != nil {
		return f.CustomEmbedFn(ctx, imageBytes)
	}
	if len(imageBytes) == 0 {
		return nil, errors.New("empty image bytes")
	}

	emb := make([]float32, 512)
	emb[0] = 1.0

	return &EmbedResult{
		Embedding:    emb,
		BBox:         []float32{100, 100, 200, 200},
		DetScore:     0.95,
		QualityScore: 0.85,
		Hints:        []string{},
		Usable:       true,
		ModelVersion: "buffalo_l",
		TookMs:       50,
	}, nil
}

func (f *FakeClient) EmbedBatch(ctx context.Context, images [][]byte) (*BatchEmbedResult, error) {
	if f.CustomBatchFn != nil {
		return f.CustomBatchFn(ctx, images)
	}
	items := make([]BatchEmbedItem, len(images))
	for i, img := range images {
		res, err := f.DetectAndEmbed(ctx, img)
		if err != nil {
			items[i] = BatchEmbedItem{
				Index: i,
				Error: &struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{
					Code:    "ERROR",
					Message: err.Error(),
				},
			}
		} else {
			items[i] = BatchEmbedItem{
				Index:  i,
				Result: res,
			}
		}
	}
	return &BatchEmbedResult{
		Items:        items,
		ModelVersion: "buffalo_l",
		TookMs:       100,
	}, nil
}
