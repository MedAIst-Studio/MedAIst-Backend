package jina

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Service struct {
	httpClient *http.Client
	apiKey     string
	model      string
}

type RerankRequest struct {
	Model           string   `json:"model"`
	Query           string   `json:"query"`
	TopN            int      `json:"top_n"`
	Documents       []string `json:"documents"`
	ReturnDocuments bool     `json:"return_documents"`
}

type RerankResponse struct {
	Results []struct {
		Index          int     `json:"index"`
		RelevanceScore float64 `json:"relevance_score"`
	} `json:"results"`
}

func NewService(apiKey string) *Service {
	return &Service{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		model:      "jina-reranker-v3",
	}
}

func (s *Service) RerankChunks(ctx context.Context, chunks []string, userQuestion string) ([]string, error) {
	req := RerankRequest{
		Model:           s.model,
		Query:           userQuestion,
		TopN:            3,
		Documents:       chunks,
		ReturnDocuments: false,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rerank request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://api.jina.ai/v1/rerank",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rerank request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send rerank request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read rerank response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Jina API error (status %d): %s", resp.StatusCode, string(body))
	}

	var jinaResp RerankResponse
	if err := json.Unmarshal(body, &jinaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rerank response: %w", err)
	}

	rerankedChunks := make([]string, 0, len(jinaResp.Results))
	for _, result := range jinaResp.Results {
		if result.Index >= 0 && result.Index < len(chunks) {
			rerankedChunks = append(rerankedChunks, chunks[result.Index])
		}
	}

	return rerankedChunks, nil
}
