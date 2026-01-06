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
	OcrModel               string
	ReasoningModel         string
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
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if len(msg.ImageUrl) > 0 {
			// Handle Multimodal Message (Text + Image)
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
			// Handle Text-only Message
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

///////////////////////////////////////////////////////////////////////////////
// 6) OCR (IMAGE TO TEXT)
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) ExtractTextFromImage(ctx context.Context, imageUrl string) (string, error) {
	request := openai.ChatCompletionRequest{
		Model: s.OcrModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleSystem,
				Content: `Sen bir OCR ve Soru Dijitalleştirme Uzmanısın. Görevin: Resimdeki metni BİREBİR, KELİMESİ KELİMESİNE dışarı aktarmaktır.
Kurallar:
1. Asla özetleme yapma. Resimdeki en ufak detayı bile atlama.
2. Matematiksel ifadeleri LaTeX formatında yaz (örn: $\frac{a}{b}$).
3. Satır sonlarını rastgele kesme, cümlenin bittiği yere kadar oku.
4. "..." gibi kısaltmalar kullanma, metinde ne yazıyorsa aynısını yaz.
5. Soru numarası, öncüller (I, II, III) ve şıkların (A, B, C, D, E) TAMAMINI eksiksiz çıkar.
6. Sadece çıkarılan metni döndür. Yorum veya giriş cümlesi yazma.
7. Resim içerisindeki resimleri de eksiksiz çıkar.`,
			},
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: imageUrl,
						},
					},
				},
			},
		},
		MaxTokens: 1100,
	}

	resp, err := s.client.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("OCR failed: %w", err)
	}

	fmt.Printf("DEBUG: OCR FinishReason: %s\n", resp.Choices[0].FinishReason)
	fmt.Printf("DEBUG: OCR Token Usage: Prompt: %d, Completion: %d, Total: %d\n", resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

///////////////////////////////////////////////////////////////////////////////
// 7) REASONING (SOLVE QUESTION)
///////////////////////////////////////////////////////////////////////////////

func (s *OpenAIService) SolveQuestionWithReasoning(ctx context.Context, questionText string) (string, error) {
	// o1 models (beta) typically do not support "system" role or have restrictions.
	// We put instructions in the user message.

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
