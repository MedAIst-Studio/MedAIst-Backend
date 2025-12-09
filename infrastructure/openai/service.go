package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	openai "github.com/sashabaranov/go-openai"
)

var (
	quizQuestionSchema = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"questionText": map[string]interface{}{
				"type":        "string",
				"description": "The quiz question text (20-40 words)",
			},
			"explanation": map[string]interface{}{
				"type":        "string",
				"description": "Explanation for the correct answer",
			},
			"options": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"no": map[string]interface{}{
							"type":        "integer",
							"description": "Option number (1-5)",
							"minimum":     1,
							"maximum":     5,
						},
						"label": map[string]interface{}{
							"type":        "string",
							"description": "Option label (A-E)",
							"enum":        []string{"A", "B", "C", "D", "E"},
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
					"required":             []string{"no", "label", "text", "isCorrect"},
					"additionalProperties": false,
				},
				"description": "List of exactly 5 options",
				"minItems":    5,
				"maxItems":    5,
			},
		},
		"required":             []string{"questionText", "explanation", "options"},
		"additionalProperties": false,
	}
)

type OpenAIConfig struct {
	ApiKey                 string
	ChatModel              string
	GenerateQuestionModel  string
	ExpandQuestionModel    string
	CheckUserQueryModel    string
	GenerateChatTitleModel string
}

type OpenAIService struct {
	client *openai.Client
	OpenAIConfig
}

func NewOpenAIService(config OpenAIConfig) *OpenAIService {
	client := openai.NewClient(config.ApiKey)

	return &OpenAIService{
		client:       client,
		OpenAIConfig: config,
	}
}

///////////////////////////////////////////////////////////////////////////////
// 1) NORMAL CHAT
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	fmt.Println("requestimiz", req)
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	request := openai.ChatCompletionRequest{
		Model:    s.ChatModel,
		Messages: messages,
		Stream:   req.Stream,
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return &domain.ChatResponse{
		Text: resp.Choices[0].Message.Content,
	}, nil
}

///////////////////////////////////////////////////////////////////////////////
// 2) CHAT TITLE GENERATION
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	request := openai.ChatCompletionRequest{
		Model:    s.GenerateChatTitleModel,
		Messages: messages,
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate title: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	return &domain.ChatResponse{
		Text: resp.Choices[0].Message.Content,
	}, nil
}

///////////////////////////////////////////////////////////////////////////////
// 3) USER QUERY CLASSIFICATION
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) CheckUserQuery(ctx context.Context, userContent string) (domain.QueryType, error) {
	request := openai.ChatCompletionRequest{
		Model: s.CheckUserQueryModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: domain.QueryClassificationSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userContent,
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return domain.GeneralQuery, fmt.Errorf("classification failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return domain.GeneralQuery, fmt.Errorf("empty response")
	}

	text := strings.TrimSpace(resp.Choices[0].Message.Content)

	switch text {
	case "General":
		return domain.GeneralQuery, nil
	case "Clarification":
		return domain.ClarificationQuery, nil
	case "PDFSearch":
		return domain.PdfSearchQuery, nil
	default:
		return domain.GeneralQuery, fmt.Errorf("unexpected classification: %s", text)
	}
}

///////////////////////////////////////////////////////////////////////////////
// 4) QUERY EXPANSION (JSON PARSE)
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) ExpandQuery(ctx context.Context, content string) ([]string, error) {
	request := openai.ChatCompletionRequest{
		Model: s.ExpandQuestionModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: domain.QueryExpansionSystemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: content,
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("expand query failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	var parsed domain.QueryExpansionResponse
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return parsed.Queries, nil
}

///////////////////////////////////////////////////////////////////////////////
// 5) QUIZ QUESTION GENERATOR (JSON MODE)
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error) {

	prompt := fmt.Sprintf(`
Based on the following medical text, generate a high-quality, board-style (USMLE/TUS level) multiple-choice question.

Text:
%s

Rules:
- Role: Act as a Senior Medical Professor creating exam questions.
- Question Style: Create a "Clinical Vignette" (Case-based) scenario if possible. Describe a patient's symptoms, history, or lab results, and ask for the diagnosis, mechanism, or treatment.
- Avoid simple recall questions (e.g., "What is X?"). Instead, test "Second-order" or "Third-order" thinking (application of knowledge).
- Length: Question body should be concise but detailed enough for a clinical scenario (40-100 words).
- Options: Exactly 5 options (A-E).
- Distractors: Wrong answers must be clinically plausible (e.g., differential diagnoses that present similarly). Avoid obviously wrong answers.
- One Correct Answer: Only ONE option isCorrect: true.
- Output: Respond ONLY with JSON.

Constraints:
- Do NOT use negative questions like "Which of the following is NOT...".
- Focus on the most high-yield concept in the text.
- Use turkish language.
`, chunk)

	fmt.Println(s.GenerateQuestionModel)
	schemaJSON, err := json.Marshal(quizQuestionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	request := openai.ChatCompletionRequest{
		Model: s.GenerateQuestionModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a quiz question generator. Generate questions that test understanding of medical and educational content. Always respond with valid JSON matching the provided schema.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "quiz_question",
				Schema: json.RawMessage(schemaJSON),
				Strict: true,
			},
		},
		Temperature: 0,
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("quiz generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("empty quiz response")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)

	var question domain.QuizQuestion
	if err := json.Unmarshal([]byte(content), &question); err != nil {
		return nil, fmt.Errorf("invalid quiz JSON: %w (content: %s)", err, content)
	}

	// Validate
	if question.QuestionText == "" {
		return nil, fmt.Errorf("question text is empty")
	}

	if len(question.Options) != 5 {
		return nil, fmt.Errorf("options count incorrect: %d", len(question.Options))
	}

	correctCount := 0
	for _, o := range question.Options {
		if o.IsCorrect {
			question.CorrectOptionLabel = o.Label
			correctCount++
		}
	}

	if correctCount != 1 {
		return nil, fmt.Errorf("expected exactly 1 correct option, got %d", correctCount)
	}

	return &question, nil
}
