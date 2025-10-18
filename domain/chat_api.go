package domain

// AI provider'a gönderilen istek
type ChatRequest struct {
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"` // Input yerine Messages daha açık
}

// AI provider'dan gelen yanıt
type ChatResponse struct {
	Text string `json:"text"`
}

// Tekil mesaj yapısı
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Constructor'lar
func NewChatRequest(stream bool, relevantText, userMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   stream,
		Messages: buildInitialMessages(relevantText, userMessage), // daha açıklayıcı
	}
}

func NewChatMessage(role, content string) *ChatMessage {
	return &ChatMessage{
		Role:    role,
		Content: content,
	}
}

// Helper methods
func buildSystemMessage() *ChatMessage {
	return NewChatMessage(
		"system",
		"You are a helpful medical assistant. Answer the user's questions based on the provided medical document excerpts. If the information is not available in the excerpts, respond with 'I don't know.' Do not provide any medical advice beyond the scope of the document excerpts. Response in the same language as the question.",
	)
}

func buildInitialMessages(relevantText, userMessage string) []ChatMessage {
	return []ChatMessage{
		*buildSystemMessage(),
		*NewChatMessage("user", "Here are the relevant excerpts from the medical document:\n"+relevantText),
		*NewChatMessage("user", userMessage),
	}
}

// Request üzerindeki metodlar
func (r *ChatRequest) AppendUserMessage(content string) {
	r.Messages = append(r.Messages, *NewChatMessage("user", content))
}

func (r *ChatRequest) AppendAssistantMessage(content string) { // AIMessage yerine AssistantMessage
	r.Messages = append(r.Messages, *NewChatMessage("assistant", content))
}

func (r *ChatRequest) AppendMessages(messages []ChatMessage) { // PastMessages yerine Messages
	r.Messages = append(r.Messages, messages...)
}
