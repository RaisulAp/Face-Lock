// Package inference is the HTTP client faceclock-api uses to talk to
// faceclock-inference. Fase 0 only needs Health() to power /readyz; the
// embedding endpoints are added in Fase 2 (Plan/03-Fase2.md § 4) without
// changing this file's shape.
package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client talks to faceclock-inference over the internal compose network.
// baseURL is never reachable from outside the compose network (Fase 0 §
// 2.9: faceclock-inference does not publish a port to the host).
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

// HealthStatus is the stub's /health response shape (Fase 0 § 2.8); Fase 2
// replaces the stub but keeps this endpoint's contract stable.
type HealthStatus struct {
	Status       string `json:"status"`
	ModelVersion string `json:"model_version"`
	Stub         bool   `json:"stub"`
}

// Health calls GET /health on faceclock-inference. The caller (health
// handler's /readyz check) is responsible for enforcing its own 2-second
// budget on top of this; Health itself just uses the client's configured
// INFERENCE_TIMEOUT_MS.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return nil, fmt.Errorf("inference: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("inference: request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response body, close error is not actionable

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("inference: unexpected status %d", resp.StatusCode)
	}

	var status HealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("inference: decode response: %w", err)
	}
	return &status, nil
}
