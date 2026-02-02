package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/file"
	"github.com/MedAIst-Studio/MedAIist-Backend/app/message"
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
	client *openai.Client
	Config
}

func NewOpenRouterService(config Config) *OpenRouterService {
	openaiConfig := openai.DefaultConfig(config.ApiKey)
	openaiConfig.BaseURL = "https://openrouter.ai/api/v1"

	client := openai.NewClientWithConfig(openaiConfig)

	return &OpenRouterService{
		client: client,
		Config: config,
	}
}

// ----------------------------------------------------------------------
// CHAT
// ----------------------------------------------------------------------

func (s *OpenRouterService) SendChatRequest(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if len(msg.ImageUrl) > 0 {
			multiContent := []openai.ChatMessagePart{
				{
					Type: openai.ChatMessagePartTypeText,
					Text: msg.Content,
				},
			}
			for _, url := range msg.ImageUrl {
				multiContent = append(multiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					ImageURL: &openai.ChatMessageImageURL{
						URL: url,
					},
				})
			}
			messages = append(messages, openai.ChatCompletionMessage{
				Role:         string(msg.Role),
				MultiContent: multiContent,
			})
		} else {
			messages = append(messages, openai.ChatCompletionMessage{
				Role:    string(msg.Role),
				Content: msg.Content,
			})
		}
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

func (s *OpenRouterService) GenerateChatTitle(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
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

// ----------------------------------------------------------------------
// CLASSIFICATION & EXPANSION
// ----------------------------------------------------------------------

func (s *OpenRouterService) CheckUserQuery(ctx context.Context, userContent string) (domain.QueryType, error) {
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
		return domain.GeneralQuery, nil // Default fallback
	}
}

func (s *OpenRouterService) ExpandQuery(ctx context.Context, content string) ([]string, error) {
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
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
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
		// Fallback if not valid JSON
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

	request := openai.ChatCompletionRequest{
		Model: s.GenerateQuestionModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "Sen uzman bir YKS soru hazırlayıcısısın. TYT ve AYT müfredatına uygun, ölçücü ve öğretici sorular hazırlarsın. Sadece verilen şemaya uygun JSON formatında çıktı verirsin.",
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

// ----------------------------------------------------------------------
// VISION & REASONING (Formerly Gemini & OpenAI reasoning)
// ----------------------------------------------------------------------

func (s *OpenRouterService) ExtractTextFromImage(ctx context.Context, imageBase64 string) (string, error) {
	if !strings.HasPrefix(imageBase64, "data:image") {
		imageBase64 = "data:image/jpeg;base64," + imageBase64
	}

	request := openai.ChatCompletionRequest{
		Model: s.VisionModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Sen bir OCR motorusun. Resimdeki sorudaki metinleri ve çizimleri çıkar. Sakın soruyu çözme. Sadece text çıktısı ver.",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: imageBase64,
						},
					},
				},
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("ocr/vision request failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty ocr response")
	}

	return resp.Choices[0].Message.Content, nil
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

	request := openai.ChatCompletionRequest{
		Model: s.ReasoningModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("reasoning/solving failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty reasoning response")
	}

	return resp.Choices[0].Message.Content, nil
}

// ----------------------------------------------------------------------
// EMBEDDING (Implementing EmbeddingService interface directly here)
// ----------------------------------------------------------------------

func (s *OpenRouterService) GenerateEmbedding(ctx context.Context, req *message.EmbeddingRequest) (*message.EmbeddingResponse, error) {
	embeddingReq := openai.EmbeddingRequest{
		Input: req.Text,
		Model: openai.EmbeddingModel(s.EmbeddingModel),
	}

	resp, err := s.client.CreateEmbeddings(ctx, embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	src := resp.Data[0].Embedding
	vector := make([]float32, len(src))
	for i, v := range src {
		vector[i] = float32(v)
	}

	return &message.EmbeddingResponse{
		Vector: vector,
	}, nil
}

// RerankChunks - This is NOT implemented by OpenRouter (Provider agnostic).
// User should still use Jina or similar for reranking.
// Since AIService interface requires it, we might need to compose this service with Jina
// or implement a dummy pass-through if we wanted to move EVERYTHING here, but strict SOLID
// suggests keeping specialized services separate.
// However, the previous MultiModelService implemented AIService by composition.
// OpenRouterService will implement `AIService` methods *except* RerankChunks
// if we use composition in `dependency_containers.go`.
// BUT, if we want OpenRouterService to be the standalone AIService, we can't unless it handles rerank.
// So, we will likely keep the Composition pattern in `dependency_containers.go`.
// For the sake of the interface satisfaction in `openrouter` package, we don't strictly need RerankChunks
// if we don't claim `OpenRouterService` implements the FULL `AIService` by itself,

func (s *OpenRouterService) GenerateEmbeddingBatch(ctx context.Context, req *file.EmbeddingBatchRequest) (*file.EmbeddingResponse, error) {

	// --- 1) []Text → []string conversion ---
	inputs := make([]string, len(req.Texts))
	for i, t := range req.Texts {
		inputs[i] = t.Text
	}

	embeddingReq := openai.EmbeddingRequestStrings{
		Input: inputs,
		Model: openai.EmbeddingModel(s.EmbeddingModel),
	}

	// --- 2) OpenAI call ---
	resp, err := s.client.CreateEmbeddings(ctx, embeddingReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// --- 3) Response mapping ---
	results := make([]file.Embedding, len(resp.Data))

	for i, data := range resp.Data {

		// float64 → float32 conversion
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}

		results[i] = file.Embedding{
			Id:          req.Texts[i].Id,
			Text:        req.Texts[i].Text,
			Vector:      vector,
			PageNumbers: []int{}, // if page exists, fill here
		}
	}

	// --- 4) Final response ---
	return &file.EmbeddingResponse{
		Results:   results,
		Dimension: len(results[0].Vector),
		Count:     len(results),
	}, nil
}
