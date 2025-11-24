package domain

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type QueryType string

const (
	PdfSearchQuery     QueryType = "pdf_search_query"
	ClarificationQuery QueryType = "clarification_query"
	GeneralQuery       QueryType = "general_query"
)

const (
	generalQuerySystemMessage = "You are just a general assistant. Answer the user's questions to the best of your ability. Response in the same language as the question."

	clarificationQuerySystemMessage = "You are a medical assistant. Provide clarifications based only on the previous assistant response. Do not add new medical information beyond what was already stated. Reply in the user's language."

	pdfSearchQuerySystemMessage = "You are a helpful medical assistant. Answer the user's questions based on the provided medical document excerpts. If the information is not available in the excerpts, respond with 'I don't know.' Do not provide any medical advice beyond the scope of the document excerpts. Response in the same language as the question."

	QueryClassificationSystemPrompt = `Classify the user's query into one of three types:

	- PDFSearch: Questions requesting specific information from the document (e.g., dosages, side effects, procedures, study results)
	- Clarification: Follow-up questions asking for explanation of previous responses (e.g., "explain more", "what does that mean", "can you clarify")
	- General: Greetings, casual conversation, or questions unrelated to the document (e.g., "hello", "who are you")

	Respond with exactly one word: "PDFSearch", "Clarification", or "General". When uncertain, default to "PDFSearch" for any medical-related content.`
)

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

type ChatRequestManager struct {
	QueryType   QueryType
	UserContent string
	Req         *ChatRequest
}

func NewChatRequestManager(queryType QueryType, userContent string) *ChatRequestManager {
	return &ChatRequestManager{
		QueryType:   queryType,
		UserContent: userContent,
		Req:         NewChatRequest(queryType, userContent),
	}
}

func NewChatMessage(role Role, content string) ChatMessage {
	return ChatMessage{
		Role:    role,
		Content: content,
	}
}

func NewChatRequest(queryType QueryType, userContent string) *ChatRequest {
	systemMessage := getSystemMessageForQueryType(queryType)

	return &ChatRequest{
		Messages: []ChatMessage{
			NewChatMessage(RoleSystem, systemMessage),
			NewChatMessage(RoleUser, userContent),
		},
	}
}

func getSystemMessageForQueryType(queryType QueryType) string {
	switch queryType {
	case GeneralQuery:
		return generalQuerySystemMessage
	case ClarificationQuery:
		return clarificationQuerySystemMessage
	case PdfSearchQuery:
		return pdfSearchQuerySystemMessage
	default:
		return generalQuerySystemMessage
	}
}

func (r *ChatRequest) AddRelevantExcerpts(excerpts []string) {
	if len(excerpts) == 0 {
		return
	}

	for _, excerpt := range excerpts {
		r.Messages = append(r.Messages, NewChatMessage(RoleUser, excerpt))
	}
}

func (r *ChatRequest) AppendMessages(messages []ChatMessage) {
	r.Messages = append(r.Messages, messages...)
}

func (r *ChatRequest) RemoveUserMessageToEnd() {
	if len(r.Messages) < 2 {
		return
	}
	second := r.Messages[1]
	r.Messages = append(r.Messages[:1], r.Messages[2:]...)
	r.Messages = append(r.Messages, second)
}

func (cm *ChatMessage) UpdateRole(isUser bool) {
	if isUser {
		cm.Role = RoleUser
		return
	}
	cm.Role = RoleAssistant
}

func NewChatTitleRequest(firstUserMessage string) *ChatRequest {
	return &ChatRequest{
		Stream:   false,
		Messages: buildTitleGenerationMessage(firstUserMessage),
	}
}

func buildTitleGenerationMessage(firstUserMessage string) []ChatMessage {
	return []ChatMessage{
		NewChatMessage(
			RoleSystem,
			"Generate a 3-7 word title for this medical document chat: "+firstUserMessage,
		),
	}
}

func (crm *ChatRequestManager) IsGeneralQuery() bool {
	return crm.QueryType == GeneralQuery
}

func (crm *ChatRequestManager) IsClarificationQuery() bool {
	return crm.QueryType == ClarificationQuery
}

func (crm *ChatRequestManager) IsPdfSearchQuery() bool {
	return crm.QueryType == PdfSearchQuery
}
