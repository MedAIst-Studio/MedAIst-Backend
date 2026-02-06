package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

// Consts
const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"
)

// ----------------------------------------------------------------------
// STRUCTS (Native implementation to replace go-openai)
// ----------------------------------------------------------------------

type ChatMessagePartType string

const (
	ChatMessagePartTypeText     ChatMessagePartType = "text"
	ChatMessagePartTypeImageURL ChatMessagePartType = "image_url"
)

type ChatMessageImageURL struct {
	URL string `json:"url,omitempty"`
}

type ChatMessagePart struct {
	Type     ChatMessagePartType  `json:"type,omitempty"`
	Text     string               `json:"text,omitempty"`
	ImageURL *ChatMessageImageURL `json:"image_url,omitempty"`
}

type ChatCompletionMessage struct {
	Role         string            `json:"role"`
	Content      interface{}       `json:"content"` // string or []ChatMessagePart
	MultiContent []ChatMessagePart `json:"-"`       // Helper to build content (not used in JSON marshaling directly, handled by logic)
}

type ResponseFormat struct {
	Type       string                `json:"type,omitempty"` // "json_object" or "json_schema"
	JSONSchema *ResponseFormatSchema `json:"json_schema,omitempty"`
}

type ResponseFormatSchema struct {
	Name   string          `json:"name"`
	Schema json.RawMessage `json:"schema"`
	Strict bool            `json:"strict,omitempty"`
}

type ChatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []ChatCompletionMessage `json:"messages"`
	Temperature    float32                 `json:"temperature,omitempty"`
	Stream         bool                    `json:"stream,omitempty"`
	ResponseFormat *ResponseFormat         `json:"response_format,omitempty"`
}

type ChatCompletionChoice struct {
	Index        int                   `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
}

type EmbeddingRequest struct {
	Input interface{} `json:"input"` // string or []string
	Model string      `json:"model"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"` // OpenRouter usually returns float64
	Index     int       `json:"index"`
}

type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  interface{}     `json:"usage"`
}

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

type Config struct {
	ApiKey                 string
	ChatModel              string
	GenerateQuestionModel  string
	ExpandQuestionModel    string
	CheckUserQueryModel    string
	GenerateChatTitleModel string
	ReasoningModel         string
	VisionModel            string
	EmbeddingModel         string
}

type OpenRouterService struct {
	httpClient *http.Client
	Config
}

func NewOpenRouterService(config Config) *OpenRouterService {
	return &OpenRouterService{
		httpClient: &http.Client{},
		Config:     config,
	}
}

// ----------------------------------------------------------------------
// HELPER METHODS
// ----------------------------------------------------------------------

func (s *OpenRouterService) makeRequest(ctx context.Context, method, endpoint string, body interface{}, respDest interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", openRouterBaseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.ApiKey)
	// OpenRouter specific headers (optional but good practice)
	req.Header.Set("HTTP-Referer", "https://medaiist.com") // Replace with actual site if strictly needed
	req.Header.Set("X-Title", "MedAIist Backend")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if respDest != nil {
		if err := json.NewDecoder(resp.Body).Decode(respDest); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// ----------------------------------------------------------------------
// CHAT
// ----------------------------------------------------------------------

func (s *OpenRouterService) SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	apiMessages := make([]ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if len(msg.ImageUrl) > 0 {
			parts := []ChatMessagePart{
				{Type: ChatMessagePartTypeText, Text: msg.Content},
			}
			for _, url := range msg.ImageUrl {
				parts = append(parts, ChatMessagePart{
					Type:     ChatMessagePartTypeImageURL,
					ImageURL: &ChatMessageImageURL{URL: url},
				})
			}
			apiMessages = append(apiMessages, ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: parts,
			})
		} else {
			apiMessages = append(apiMessages, ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
	}

	apiReq := ChatCompletionRequest{
		Model:    s.ChatModel,
		Messages: apiMessages,
		Stream:   req.Stream,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	content := ""
	if str, ok := apiResp.Choices[0].Message.Content.(string); ok {
		content = str
	} else if parts, ok := apiResp.Choices[0].Message.Content.([]interface{}); ok {
		for _, p := range parts {
			if m, ok := p.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					content += t
				}
			}
		}
	}

	return &domain.ChatResponse{
		Text: content,
	}, nil
}

func (s *OpenRouterService) GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	apiMessages := make([]ChatCompletionMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		apiMessages = append(apiMessages, ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	apiReq := ChatCompletionRequest{
		Model:    s.GenerateChatTitleModel,
		Messages: apiMessages,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to generate title: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return &domain.ChatResponse{
		Text: content,
	}, nil
}

// ----------------------------------------------------------------------
// CLASSIFICATION & EXPANSION
// ----------------------------------------------------------------------

func (s *OpenRouterService) CheckUserQuery(ctx context.Context, userContent string) (domain.QueryType, error) {
	apiReq := ChatCompletionRequest{
		Model: s.CheckUserQueryModel,
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: domain.QueryClassificationSystemPrompt},
			{Role: "user", Content: userContent},
		},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return domain.GeneralQuery, fmt.Errorf("classification failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return domain.GeneralQuery, fmt.Errorf("empty response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	text := strings.TrimSpace(content)

	switch text {
	case "General":
		return domain.GeneralQuery, nil
	case "Clarification":
		return domain.ClarificationQuery, nil
	case "PDFSearch":
		return domain.PdfSearchQuery, nil
	default:
		return domain.GeneralQuery, nil
	}
}

func (s *OpenRouterService) ExpandQuery(ctx context.Context, content string) ([]string, error) {
	apiReq := ChatCompletionRequest{
		Model: s.ExpandQuestionModel,
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: domain.QueryExpansionSystemPrompt},
			{Role: "user", Content: content},
		},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("expand query failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response")
	}

	respContent, _ := apiResp.Choices[0].Message.Content.(string)

	var parsed domain.QueryExpansionResponse
	if err := json.Unmarshal([]byte(respContent), &parsed); err != nil {
		return []string{content}, nil
	}

	return parsed.Queries, nil
}

// ----------------------------------------------------------------------
// QUIZ
// ----------------------------------------------------------------------

func (s *OpenRouterService) GenerateQuizQuestion(ctx context.Context, chunk string) (*domain.QuizQuestion, error) {
	prompt := fmt.Sprintf(`
Verilen metne dayanarak YKS (TYT/AYT) formatında, kaliteli bir çoktan seçmeli soru oluştur.

Metin:
%s

Kurallar:
- Rol: Kıdemli YKS soru hazırlayıcısı.
- Soru Tarzı: ÖSYM standartlarına uygun, muhakeme ve bilgi ölçen nitelikli bir soru olsun.
- Zorluk: Orta veya zor seviye (Öğrenciyi düşündüren).
- Uzunluk: Gereksiz uzun olmasın, net ve anlaşılır olsun.
- Şıklar: Tam 5 şık (A-E).
- Çeldiriciler: Mantıklı ve güçlü çeldiriciler kullan.
- Tek Doğru: Sadece bir şık isCorrect: true olsun.
- Çıktı: SADECE JSON formatında yanıt ver.

Kısıtlamalar:
- "Hangisi değildir" gibi olumsuz köklü sorulardan kaçınmaya çalış (zorunluysa kullanabilirsin ama pozitif köklü sorular öncelikli).
- Metindeki en önemli kavramı odak noktası yap.
- Türkçe kullan.
`, chunk)

	schemaJSON, err := json.Marshal(quizQuestionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	apiReq := ChatCompletionRequest{
		Model: s.GenerateQuestionModel,
		Messages: []ChatCompletionMessage{
			{
				Role:    "system",
				Content: "Sen uzman bir YKS soru hazırlayıcısısın. TYT ve AYT müfredatına uygun, ölçücü ve öğretici sorular hazırlarsın. Sadece verilen şemaya uygun JSON formatında çıktı verirsin.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		ResponseFormat: &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &ResponseFormatSchema{
				Name:   "quiz_question",
				Schema: json.RawMessage(schemaJSON),
				Strict: true,
			},
		},
		Temperature: 0,
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("quiz generation failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("empty quiz response")
	}

	respContent, _ := apiResp.Choices[0].Message.Content.(string)
	content := strings.TrimSpace(respContent)

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

// ----------------------------------------------------------------------
// VISION & REASONING
// ----------------------------------------------------------------------

func (s *OpenRouterService) ExtractTextFromImage(ctx context.Context, imageBase64 string) (string, error) {
	if !strings.HasPrefix(imageBase64, "data:image") {
		imageBase64 = "data:image/jpeg;base64," + imageBase64
	}

	parts := []ChatMessagePart{
		{Type: ChatMessagePartTypeText, Text: "Sen bir OCR motorusun. Resimdeki sorudaki metinleri ve çizimleri çıkar. Sakın soruyu çözme. Sadece text çıktısı ver."},
		{Type: ChatMessagePartTypeImageURL, ImageURL: &ChatMessageImageURL{URL: imageBase64}},
	}

	apiReq := ChatCompletionRequest{
		Model: s.VisionModel,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: parts},
		},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return "", fmt.Errorf("ocr/vision request failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty ocr response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return content, nil
}

func (s *OpenRouterService) SolveQuestionWithReasoning(ctx context.Context, questionText string) (string, error) {
	prompt := fmt.Sprintf(`Sen uzman bir YKS öğretmenisin (Matematik, Fen, Türkçe, Sosyal). Öğrencinin sorduğu soruyu adım adım, anlaşılır ve açıklayıcı bir şekilde çöz.

Soru:
%s

Yönergeler:
1. Soruyu dikkatlice analiz et ve hangi konudan olduğunu belirle.
2. Çözüm adımlarını mantıksal bir sırayla yaz (işlem önceliği, formüller, mantık yürütme).
3. Varsa şıkları (A-E) değerlendir, neden doğru veya yanlış olduklarını açıkla.
4. Sonucu net bir şekilde belirt.
5. Samimi, eğitici ve teşvik edici bir dil kullan (Öğrenciye hitap et).
6. Türkçe yanıtla.
`, questionText)

	apiReq := ChatCompletionRequest{
		Model: s.ReasoningModel,
		Messages: []ChatCompletionMessage{
			{Role: "user", Content: prompt},
		},
	}

	var apiResp ChatCompletionResponse
	if err := s.makeRequest(ctx, "POST", "/chat/completions", apiReq, &apiResp); err != nil {
		return "", fmt.Errorf("reasoning/solving failed: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("empty reasoning response")
	}

	content, _ := apiResp.Choices[0].Message.Content.(string)
	return content, nil
}

// ----------------------------------------------------------------------
// EMBEDDING
// ----------------------------------------------------------------------

func (s *OpenRouterService) GenerateEmbedding(ctx context.Context, req *message.EmbeddingRequest) (*message.EmbeddingResponse, error) {
	apiReq := EmbeddingRequest{
		Input: req.Text,
		Model: s.EmbeddingModel,
	}

	var apiResp EmbeddingResponse
	if err := s.makeRequest(ctx, "POST", "/embeddings", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	// Assuming float64 -> float32 conversion needed
	src := apiResp.Data[0].Embedding
	vector := make([]float32, len(src))
	for i, v := range src {
		vector[i] = float32(v)
	}

	return &message.EmbeddingResponse{
		Vector: vector,
	}, nil
}

func (s *OpenRouterService) GenerateEmbeddingBatch(ctx context.Context, req *file.EmbeddingBatchRequest) (*file.EmbeddingResponse, error) {
	// []Text -> []string
	inputs := make([]string, len(req.Texts))
	for i, t := range req.Texts {
		inputs[i] = t.Text
	}

	apiReq := EmbeddingRequest{
		Input: inputs,
		Model: s.EmbeddingModel,
	}

	var apiResp EmbeddingResponse
	if err := s.makeRequest(ctx, "POST", "/embeddings", apiReq, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	results := make([]file.Embedding, len(apiResp.Data))
	for i, data := range apiResp.Data {
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}

		results[i] = file.Embedding{
			Id:          req.Texts[i].Id,
			Text:        req.Texts[i].Text,
			Vector:      vector,
			PageNumbers: []int{},
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("empty embedding results")
	}

	return &file.EmbeddingResponse{
		Results:   results,
		Dimension: len(results[0].Vector),
		Count:     len(results),
	}, nil
}
