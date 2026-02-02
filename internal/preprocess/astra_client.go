package preprocess

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/RishiKendai/aegis/internal/models"
	"github.com/rs/zerolog/log"
)

// AstraClient handles communication with Astra preprocessing API
type AstraClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAstraClient creates a new Astra API client
func NewAstraClient(baseURL, apiKey string) *AstraClient {
	return &AstraClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{
			// No timeout - will wait indefinitely for response
		},
	}
}

// PreprocessRequest represents the request to Astra API
type PreprocessRequest struct {
	EmailID   string `json:"email"`
	AttemptID string `json:"attemptId"`
	DriveID   string `json:"driveId"`
	TestID    string `json:"testId"`
	Code      string `json:"sourceCode"`
	Language  string `json:"language"`
}

func (c *AstraClient) Preprocess(ctx context.Context, req *PreprocessRequest) (*models.PreprocessingResponse, error) {
	url := fmt.Sprintf("%s/api/v1/preprocess", c.baseURL)

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Trace().
		Msg("----------------------✌️--------------------------------")
	log.Trace().
		Msgf("reqBody: %s", string(reqBody))
	log.Trace().
		Msg("----------------------✌️--------------------------------")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error status codes
	if resp.StatusCode == http.StatusBadRequest ||
		resp.StatusCode == http.StatusUnsupportedMediaType ||
		resp.StatusCode == http.StatusUnprocessableEntity {
		var errResp models.PreprocessingError
		if err := json.Unmarshal(body, &errResp); err != nil {
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}
		return nil, fmt.Errorf("API error: %s - %s", errResp.Error, errResp.Message)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var preprocessingResp models.PreprocessingResponse
	if err := json.Unmarshal(body, &preprocessingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &preprocessingResp, nil
}
