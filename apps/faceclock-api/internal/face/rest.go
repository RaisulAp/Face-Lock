package face

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// RESTEngine implements FaceEngine by delegating facial detection and embedding
// generation to the Python faceclock-inference microservice.
type RESTEngine struct {
	baseURL      string
	token        string
	threshold    float32
	modelVersion string
	httpClient   *http.Client
}

// NewRESTEngine creates a RESTEngine pointing to the inference microservice.
func NewRESTEngine(baseURL, token, modelVersion string, threshold float32, timeout time.Duration) *RESTEngine {
	if threshold <= 0 {
		threshold = DefaultCosineThreshold
	}
	if modelVersion == "" {
		modelVersion = "buffalo_l"
	}
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	return &RESTEngine{
		baseURL:      baseURL,
		token:        token,
		threshold:    threshold,
		modelVersion: modelVersion,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (r *RESTEngine) Threshold() float32 {
	return r.threshold
}

func (r *RESTEngine) ModelVersion() string {
	return r.modelVersion
}

func (r *RESTEngine) IsMatch(similarity float32) bool {
	return similarity >= r.threshold
}

func (r *RESTEngine) Compare(v1, v2 []float32) (float32, float32) {
	sim, err := CosineSimilarity(v1, v2)
	if err != nil {
		sim = 0
	}
	dist, err := EuclideanDistance(v1, v2)
	if err != nil {
		dist = 999.0
	}
	return sim, dist
}

// embedAPIResponse handles both `{ "data": { ... } }` and flat `{ ... }` payloads.
type embedAPIResponse struct {
	Data struct {
		Embedding    []float32       `json:"embedding"`
		BBox         []float32       `json:"bbox"`
		DetScore     *float32        `json:"det_score"`
		Quality      *QualityMetrics `json:"quality"`
		QualityScore float32         `json:"quality_score"`
		Hints        []string        `json:"hints"`
		Usable       bool            `json:"usable"`
		ModelVersion string          `json:"model_version"`
		EmbeddingDim int             `json:"embedding_dim"`
		TookMs       int             `json:"took_ms"`
	} `json:"data"`
	// Flat fallback
	Embedding    []float32       `json:"embedding"`
	BBox         []float32       `json:"bbox"`
	DetScore     *float32        `json:"det_score"`
	Quality      *QualityMetrics `json:"quality"`
	QualityScore float32         `json:"quality_score"`
	Hints        []string        `json:"hints"`
	Usable       bool            `json:"usable"`
	ModelVersion string          `json:"model_version"`
	EmbeddingDim int             `json:"embedding_dim"`
}

// DetectAndEmbed sends the image bytes to the inference service via multipart form-data.
func (r *RESTEngine) DetectAndEmbed(ctx context.Context, imageBytes []byte) (*DetectionResult, error) {
	if len(imageBytes) == 0 {
		return nil, ErrEmptyImagePayload
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "frame.jpg")
	if err != nil {
		return nil, fmt.Errorf("face: create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return nil, fmt.Errorf("face: write image payload: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("face: close multipart writer: %w", err)
	}

	endpoint := r.baseURL + "/v1/embed"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("face: create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("face: inference request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, ErrFaceNotUsable
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("face: upstream status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed embedAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("face: decode inference response: %w", err)
	}

	// Unpack data wrapper if present
	emb := parsed.Data.Embedding
	bbox := parsed.Data.BBox
	detScore := parsed.Data.DetScore
	quality := parsed.Data.Quality
	qualityScore := parsed.Data.QualityScore
	hints := parsed.Data.Hints
	usable := parsed.Data.Usable
	modVer := parsed.Data.ModelVersion

	if len(emb) == 0 && len(parsed.Embedding) > 0 {
		emb = parsed.Embedding
		bbox = parsed.BBox
		detScore = parsed.DetScore
		quality = parsed.Quality
		qualityScore = parsed.QualityScore
		hints = parsed.Hints
		usable = parsed.Usable
		modVer = parsed.ModelVersion
	}

	if modVer == "" {
		modVer = r.modelVersion
	}

	var box *BoundingBox
	if len(bbox) >= 4 {
		box = &BoundingBox{
			X1: bbox[0],
			Y1: bbox[1],
			X2: bbox[2],
			Y2: bbox[3],
		}
	}

	var score float32
	if detScore != nil {
		score = *detScore
	}

	return &DetectionResult{
		BoundingBox:  box,
		Embedding:    emb,
		DetScore:     score,
		Quality:      quality,
		QualityScore: qualityScore,
		IsUsable:     usable,
		ModelVersion: modVer,
		Hints:        hints,
	}, nil
}
