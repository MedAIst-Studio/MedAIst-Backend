package domain

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type QueryType string

const (
	QueryTypeSimple  QueryType = "simple"  // Basit soru, doküman gerektirmiyor
	QueryTypeComplex QueryType = "complex" // PDF context gerekiyor
)

type RouterDecision struct {
	QueryType      QueryType `json:"query_type"`
	SimpleResponse string    `json:"simple_response,omitempty"` // QueryTypeSimple için hazır cevap
	Reasoning      string    `json:"reasoning,omitempty"`       // Debug için
}

type ChatRequest struct {
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Text string `json:"text"`
}

type ChatMessage struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

func NewChatRequest(stream bool, relevantTexts []string, userMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   stream,
		Messages: buildInitialMessages(relevantTexts, userMessage),
	}
}

func NewChatMessage(role Role, content string) *ChatMessage {
	return &ChatMessage{
		Role:    role,
		Content: content,
	}
}
func buildSystemMessage(context []string, message string) *ChatMessage {
	contextStr := ""
	for _, text := range context {
		contextStr += text + "\n"
	}

	systemPrompt := `[ROL VE KİŞİLİK]
Sen, tıp fakültesi öğrencilerine rehberlik eden, deneyimli ve destekleyici bir Akademik Mentorsün. Görevin, sana sunulan kaynak metinleri kullanarak karmaşık tıp konularını öğretmek. Üslubun profesyonel, akademik ve destekleyicidir.

[TEMEL GÖREV]
Sana <context>...etiketleri arasında bir PDF metni ve<user_question>...</user_question> etiketleri arasında bir öğrenci sorusu verilecek. Görevin, bu soruyu cevaplamak.

[CEVAPLAMA KURALLARI]

BAĞLAMA MUTLAK SADAKAT: Cevapların SADECE VE SADECE <context> etiketleri içindeki bilgilere dayanmalıdır. ASLA bu metnin dışına çıkma veya bilgi uydurma.

Eğer cevap <context> içinde yoksa, net bir şekilde: Bu spesifik bilgi şu an önümdeki kaynak metinde bulunmuyor. de.
DURUMSAL DETAY (Gerektiği Kadar):

Kullanıcının sorusuna her zaman net, kısa ve doğrudan bir cevap ver.
Kullanıcıyı bilgiye boğma. Soru basitse (örn: X nedir?), cevap da basit ve net olmalıdır.
OPSİYONEL KLİNİK KÖPRÜ:

Eğer <context> içindeki bilgi, çok bariz bir klinik bağlantı (hastalık, belirti, tanı) içeriyorsa VE soru buna uygunsa, cevaba kısa bir Klinik Not: veya Pratik Bilgi: başlığı altında bu bağlantıyı ekleyebilirsin.
Bu bir zorunluluk DEĞİLDİR. Basit tanım soruları için bu adımı atla.

[GİRDİLER VE CEVAP AKIŞI]

<context>
` + contextStr + `
</context>

<user_question>
` + message + `
</user_question>`

	return NewChatMessage(RoleSystem, systemPrompt)
}

func buildInitialMessages(relevantTexts []string, userMessage string) []ChatMessage {
	// System message zaten user question'ı içeriyor, tekrar eklemeye gerek yok
	messages := []ChatMessage{
		*buildSystemMessage(relevantTexts, userMessage),
	}

	return messages
}

func (r *ChatRequest) AppendUserMessage(content string) {
	r.Messages = append(r.Messages, *NewChatMessage(RoleUser, content))
}

func (r *ChatRequest) AppendAssistantMessage(content string) {
	r.Messages = append(r.Messages, *NewChatMessage(RoleAssistant, content))
}

func (r *ChatRequest) AppendMessages(messages []ChatMessage) {
	r.Messages = append(r.Messages, messages...)
}

func (cm *ChatMessage) UpdateRole(isUser bool) {
	if isUser {
		cm.Role = RoleUser
	} else {
		cm.Role = RoleAssistant
	}
}

func NewChatTitleRequest(firstUserMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   false,
		Messages: buildTitleGenerationMessage(firstUserMessage),
	}
}

func buildTitleGenerationMessage(firstUserMessage string) []ChatMessage {
	return []ChatMessage{
		*NewChatMessage(
			RoleSystem,
			"Generate a 3-7 word title for this medical document chat: "+firstUserMessage,
		),
	}
}

func NewRouterRequest(userMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   false,
		Messages: buildRouterMessage(userMessage),
	}
}

func buildRouterMessage(userMessage string) []ChatMessage {
	return []ChatMessage{
		*NewChatMessage(
			RoleSystem,
			`You are a query router for a medical education chatbot. Your job is to determine if a user's question requires searching through PDF documents or can be answered directly.

Classify the query as:
- "simple": Greetings, general questions, casual conversation, or questions that don't require specific medical document content
- "complex": Questions that require specific medical knowledge from documents (symptoms, diagnoses, treatments, anatomy, etc.)

For "simple" queries, provide a friendly, helpful response directly.
For "complex" queries, just classify them without answering.

Respond ONLY with valid JSON in this exact format:
{"query_type": "simple" or "complex", "simple_response": "your response here (only for simple)", "reasoning": "brief explanation"}

Examples:
User: "Merhaba!"
Response: {"query_type": "simple", "simple_response": "Merhaba! Size nasıl yardımcı olabilirim? Tıbbi dokümanlardaki konular hakkında sorularınızı cevaplayabilirim.", "reasoning": "greeting"}

User: "Kalp yetmezliğinin belirtileri nelerdir?"
Response: {"query_type": "complex", "reasoning": "requires medical document content"}

User: "how are you?"
Response: {"query_type": "simple", "simple_response": "I'm doing well, thank you! I'm here to help you with questions about medical documents. What would you like to know?", "reasoning": "casual conversation"}

Now classify this query:
`+userMessage,
		),
	}
}
