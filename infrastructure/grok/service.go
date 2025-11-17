package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type GrokService struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	model      string
}

type grokMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type grokRequest struct {
	Messages []grokMessage `json:"messages"`
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
}

type grokChoice struct {
	Index   int         `json:"index"`
	Message grokMessage `json:"message"`
}

type grokResponse struct {
	ID      string       `json:"id"`
	Choices []grokChoice `json:"choices"`
	Model   string       `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func NewGrokService(apiKey, model string) *GrokService {
	return &GrokService{
		apiKey:  apiKey,
		baseURL: "https://api.x.ai/v1",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		model: model,
	}
}

func (s *GrokService) SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	messages := make([]grokMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, grokMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	grokReq := grokRequest{
		Messages: messages,
		Model:    s.model,
		Stream:   req.Stream,
	}

	return s.makeRequest(ctx, grokReq)
}

func (s *GrokService) GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	messages := make([]grokMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, grokMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	grokReq := grokRequest{
		Messages: messages,
		Model:    s.model,
		Stream:   false,
	}

	return s.makeRequest(ctx, grokReq)
}

func (s *GrokService) RouteMessage(ctx context.Context, req *domain.ChatRequest) (*domain.RouterDecision, error) {
	messages := make([]grokMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, grokMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	grokReq := grokRequest{
		Messages: messages,
		Model:    s.model,
		Stream:   false,
	}

	resp, err := s.makeRequest(ctx, grokReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get router decision: %w", err)
	}

	var decision domain.RouterDecision
	if err := json.Unmarshal([]byte(resp.Text), &decision); err != nil {
		return &domain.RouterDecision{
			QueryType: domain.QueryTypeComplex,
			Reasoning: "failed to parse router response, defaulting to complex",
		}, nil
	}

	return &decision, nil
}

func (s *GrokService) makeRequest(ctx context.Context, grokReq grokRequest) (*domain.ChatResponse, error) {
	jsonData, err := json.Marshal(grokReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/chat/completions", s.baseURL),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var grokResp grokResponse
	if err := json.Unmarshal(body, &grokResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(grokResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &domain.ChatResponse{
		Text: grokResp.Choices[0].Message.Content,
	}, nil
}
