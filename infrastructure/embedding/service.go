package embeddingapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
)

type EmbeddingService struct {
	APIKey     string
	APIDomain  string
	HTTPClient *http.Client
}

func NewEmbeddingService(apiKey, apiDomain string) *EmbeddingService {
	return &EmbeddingService{
		APIKey:     apiKey,
		APIDomain:  apiDomain,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *EmbeddingService) GenerateEmbeddingBatch(ctx context.Context, req *file.EmbeddingBatchRequest) (*file.EmbeddingResponse, error) {
	fmt.Printf("Sending embedding request with %d texts\n", len(req.Texts))
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	url := c.APIDomain + "/embed/batch"
	fmt.Println("Sending embedding request to:", url)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var embeddingResp file.EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	return &embeddingResp, nil
}
