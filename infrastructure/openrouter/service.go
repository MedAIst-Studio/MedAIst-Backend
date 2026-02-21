package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

// Consts
const (
	openRouterBaseURL = "https://openrouter.ai/api/v1"

	queryExpansionSystemPrompt = `Sen bir arama sorgusu genişletme uzmanısın. Görevin, kullanıcının asıl niyetini koruyarak farklı açılardan 5 adet Türkçe arama sorgusu üretmektir.

	Kurallar:
	1. Asıl niyeti koru - kullanıcının ne sorduğunu değiştirme.
	2. Farklı ifadeler, eş anlamlılar ve bakış açıları kullan.
	3. YKS (TYT/AYT) müfredatına uygun terimler kullanmaya çalış.
	4. Asıl sorguda olmayan tamamen yeni bilgi ekleme.

	SADECE şu formatta geçerli bir JSON döndür:
	{"queries": ["...", "...", "...", "...", "..."]}
`
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

func (s *OpenRouterService) ExpandQuery(ctx context.Context, content string) ([]string, error) {
	apiReq := ChatCompletionRequest{
		Model: s.ExpandQuestionModel,
		Messages: []ChatCompletionMessage{
			{Role: "system", Content: queryExpansionSystemPrompt},
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
	prompt := fmt.Sprintf(`Aşağıdaki metin parçasını dikkatlice oku ve bu metne dayalı bir çoktan seçmeli soru üret.

--- METİN BAŞLANGIÇ ---
%s
--- METİN BİTİŞ ---

SORU ÜRETME ADIMLARI (Sırasıyla uygula):

1. METİN ANALİZİ: Metindeki en önemli kavramı, ilişkiyi veya bilgiyi belirle.
2. SORU KÖKÜ YAZIMI: Bu kavramı ölçen, net ve anlaşılır bir soru kökü yaz (20-40 kelime). Soru kökü tek başına okunduğunda ne sorulduğu anlaşılmalı.
3. DOĞRU ŞIK: Metne dayalı olarak doğru cevabı yaz.
4. ÇELDİRİCİ ŞIKLAR: Kalan 4 şıkkı oluştururken şu kriterlere MUTLAKA uy:
   a) Her çeldirici, soru kökünü doğal şekilde tamamlamalı veya cevaplamalıdır.
   b) Çeldiriciler, sorunun konusuyla AYNI konu alanından olmalıdır.
   c) Çeldiriciler mantıklı ama yanlış olmalıdır — rastgele veya alakasız bilgi YAZMA.
   d) Tüm şıklar benzer uzunlukta ve dil yapısında olmalıdır.
   e) Bir öğrenci şıklara baktığında hepsinin makul bir cevap olabileceğini düşünmeli.
5. AÇIKLAMA: Doğru cevabın neden doğru olduğunu kısaca açıkla.

FORMAT KURALLARI:
- ÖSYM/YKS standardında ol.
- Zorluk: Orta-zor (öğrenciyi düşündüren).
- Tam 5 şık (A-E), sadece 1 tanesi doğru.
- Türkçe yaz.
- SADECE JSON çıktısı ver, başka metin yazma.`, chunk)

	schemaJSON, err := json.Marshal(quizQuestionSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	apiReq := ChatCompletionRequest{
		Model: s.GenerateQuestionModel,
		Messages: []ChatCompletionMessage{
			{
				Role: "system",
				Content: `Sen 20 yıllık deneyime sahip, kıdemli bir ÖSYM soru hazırlayıcısısın.

TEMEL GÖREVİN: Verilen metne dayalı, yüksek kaliteli çoktan seçmeli sorular üretmek.

KRİTİK KURALLAR:
1. TUTARLILIK: Ürettiğin her şık (A-E), soru köküyle dilbilgisel ve anlamsal olarak uyumlu olmalıdır. Soru kökünü oku, ardından her şıkkı soru köküne ekleyerek oku — cümle doğal ve anlamlı olmalı.
2. KONU BİRLİĞİ: Tüm şıklar aynı konu alanından olmalıdır. Örneğin soru hücre bölünmesi hakkındaysa, şıkların hepsi biyoloji/hücre biyolojisi alanından olmalı.
3. ÇELDİRİCİ KALİTESİ: Yanlış şıklar, "neredeyse doğru" olmalı — tamamen alakasız veya saçma olmamalı. İyi bir çeldirici, konuyu yüzeysel bilen öğrenciyi yanıltabilmelidir.
4. Sadece verilen JSON şemasına uygun çıktı üret.`,
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
		Temperature: 0.3,
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
	apiReq := ChatCompletionRequest{
		Model: s.ReasoningModel,
		Messages: []ChatCompletionMessage{
			{
				Role: "system",
				Content: `Soru çözücüsün. Giriş cümlesi olmadan direkt çözüme başla.

FORMAT (Markdown kullan):
### Konu: [Konuyu kısa belirt]

###  Çözüm
Adımları numaralı liste halinde yaz. Her adımda:
- Önemli terimleri **kalın** yap
- Formülleri LaTeX formatında yaz: $formül$
- Gerekirse kısa açıklama ekle

### Cevap
Sonucu net ve belirgin şekilde yaz.

KURALLAR:
- Kendini tanıtma, selamlama, "Merhaba", "Hadi bakalım" gibi dolgu cümleler YAZMA
- Soruyu tekrar alıntılama
- Her adım öz ve anlaşılır olsun, ama gerekli açıklamayı atla demek değil
- Türkçe yanıtla`,
			},
			{
				Role:    "user",
				Content: questionText,
			},
		},
		Temperature: 0.4,
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

func (s *OpenRouterService) GenerateEmbedding(ctx context.Context, req *domain.SingleEmbeddingRequest) (*domain.SingleEmbeddingResponse, error) {
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

	src := apiResp.Data[0].Embedding
	vector := make([]float32, len(src))
	for i, v := range src {
		vector[i] = float32(v)
	}

	return &domain.SingleEmbeddingResponse{
		Vector: vector,
	}, nil
}

func (s *OpenRouterService) GenerateEmbeddingBatch(ctx context.Context, req *domain.EmbeddingBatchRequest) (*domain.EmbeddingResponse, error) {
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

	results := make([]domain.Embedding, len(apiResp.Data))
	for i, data := range apiResp.Data {
		vector := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			vector[j] = float32(v)
		}

		results[i] = domain.Embedding{
			Id:          req.Texts[i].Id,
			Text:        req.Texts[i].Text,
			Vector:      vector,
			PageNumbers: []int{},
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("empty embedding results")
	}

	return &domain.EmbeddingResponse{
		Results:   results,
		Dimension: len(results[0].Vector),
		Count:     len(results),
	}, nil
}
