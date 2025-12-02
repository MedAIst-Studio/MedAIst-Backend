package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

var (
	schema = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"questionText": map[string]interface{}{
				"type":        "string",
				"description": "The quiz question text",
			},
			"options": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"no": map[string]interface{}{
							"type":        "integer",
							"description": "Option number (1-4)",
						},
						"label": map[string]interface{}{
							"type":        "string",
							"description": "Option label (A-D)",
						},
						"text": map[string]interface{}{
							"type":        "string",
							"description": "Option text",
						},
						"isCorrect": map[string]interface{}{
							"type":        "boolean",
							"description": "Whether this option is correct",
						},
					},
					"required": []string{"no", "label", "text", "isCorrect"},
				},
				"description": "List of 4 options",
			},
		},
		"required": []string{"questionText", "options"},
	}
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

func (s *GrokService) CheckUserQuery(ctx context.Context, userContent string) (domain.QueryType, error) {
	grokReq := grokRequest{
		Messages: []grokMessage{
			{
				Role:    "system",
				Content: domain.QueryClassificationSystemPrompt,
			},
			{
				Role:    "user",
				Content: userContent,
			},
		},
		Model:  s.model,
		Stream: false,
	}

	resp, err := s.makeRequest(ctx, grokReq)
	if err != nil {
		return domain.GeneralQuery, err
	}

	switch resp.Text {
	case "General":
		return domain.GeneralQuery, nil
	case "Clarification":
		return domain.ClarificationQuery, nil
	case "PDFSearch":
		return domain.PdfSearchQuery, nil
	default:
		return domain.GeneralQuery, fmt.Errorf("unexpected response: %s", resp.Text)
	}
}

func (s *GrokService) ExpandQuery(ctx context.Context, content string) ([]string, error) {

	messages := []grokMessage{
		{
			Role:    "system",
			Content: domain.QueryExpansionSystemPrompt,
		},
		{
			Role:    "user",
			Content: content,
		},
	}
	grokReq := grokRequest{
		Messages: messages,
		Model:    s.model,
		Stream:   false,
	}

	resp, err := s.makeQueryExpansionRequest(ctx, grokReq)
	if err != nil {
		return nil, err
	}

	return resp.Queries, nil
}

func (s *GrokService) GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error) {

	prompt := fmt.Sprintf(`Based on the following text, generate a multiple-choice quiz question.

Text: %s

Requirements:
- Create one question that tests understanding of the text
- Question should be detailed and comprehensive, around 20-40 words long
- Provide exactly 5 options (A, B, C, D, E)
- Mark exactly one option as correct (isCorrect: true)
- Option numbers should be 1, 2, 3, 4, 5
- Option labels should be "A", "B", "C", "D", "E"
- Make the question clear and specific
- Ensure incorrect options are plausible but clearly wrong
- CRITICAL: Randomly place the correct answer in ANY position (A, B, C, D, or E), NOT always in position B
- Vary the position of the correct answer across different questions

IMPORTANT: Respond ONLY with valid JSON in this exact format and the same language as the input text, no other text:

{
  "questionText": "Your detailed question here (20-40 words)",
  "options": [
    {"no": 1, "label": "A", "text": "Option A text", "isCorrect": false},
    {"no": 2, "label": "B", "text": "Option B text", "isCorrect": false},
    {"no": 3, "label": "C", "text": "Option C text", "isCorrect": true},
    {"no": 4, "label": "D", "text": "Option D text", "isCorrect": false},
    {"no": 5, "label": "E", "text": "Option E text", "isCorrect": false}
  ]
}

Remember: The correct answer position should vary randomly - it could be A, B, C, D, or E.`, chunk)

	requestBody := map[string]interface{}{
		"model": s.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a quiz question generator. Always respond with valid JSON only, no markdown, no explanations.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"response_format": map[string]interface{}{
			"type": "json_object",
		},
		"temperature": 0.7,
	}

	jsonData, err := json.Marshal(requestBody)
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

	content := grokResp.Choices[0].Message.Content

	// Clean up potential markdown code blocks
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Parse the JSON response
	var quizQuestion domain.QuizQuestion
	if err := json.Unmarshal([]byte(content), &quizQuestion); err != nil {
		return nil, fmt.Errorf("failed to parse quiz question JSON: %w (content: %s)", err, content)
	}

	// Validate the question
	if quizQuestion.QuestionText == "" {
		return nil, fmt.Errorf("generated question text is empty")
	}

	if len(quizQuestion.Options) != 5 {
		return nil, fmt.Errorf("expected 5 options, got %d", len(quizQuestion.Options))
	}

	// Find and set the correct option label
	correctCount := 0
	for _, option := range quizQuestion.Options {
		if option.IsCorrect {
			quizQuestion.CorrectOptionLabel = option.Label
			correctCount++
		}
	}

	if correctCount != 1 {
		return nil, fmt.Errorf("expected exactly 1 correct option, got %d", correctCount)
	}

	return &quizQuestion, nil
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

func (s *GrokService) makeQueryExpansionRequest(ctx context.Context, grokReq grokRequest) (*domain.QueryExpansionResponse, error) {
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

	fmt.Printf("grokResp: %v\n", grokResp)

	if len(grokResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	var queries domain.QueryExpansionResponse
	if err := json.Unmarshal([]byte(grokResp.Choices[0].Message.Content), &queries); err != nil {
		return nil, fmt.Errorf("failed to parse queries: %w", err)
	}
	return &queries, nil
}
