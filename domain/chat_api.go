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
	ImageQuery         QueryType = "image_query"
)

const (
	generalQuerySystemMessage = "You are just a general assistant. Answer the user's questions to the best of your ability. Response in the same language as the question."

	clarificationQuerySystemMessage = "You are a medical assistant. Provide clarifications based only on the previous assistant response. Do not add new medical information beyond what was already stated. Reply in the user's language."

	imageQuerySystemMessage = "You are a medical assistant capable of analyzing medical images. Analyze the provided image and answer the user's question. If the image is not medical related, politely refuse to answer."

	pdfSearchQuerySystemMessage = `You are a professional Medical Tutor. Answer questions using ONLY the provided Context below.

CRITICAL RULES:
1. **Answer Availability Check:**
   - FIRST, carefully search the entire Context for relevant information
   - If ANY part of the Context contains information that helps answer the question, you MUST use it
   - ONLY respond with "Yüklediğiniz dokümanda bu bilgi yer almamaktadır" if you have thoroughly checked and the answer is completely absent
2. **Answer Construction:**
   - Use **Bold** for key medical terms
   - Use bullet points for lists
   - Provide comprehensive answers using ALL relevant information from the Context
   - Quote or paraphrase directly from the Context when possible
3. **Language Matching:**
   - Respond in the same language as the question
4. **Important:** 
   - Partial information IS still valid - use whatever relevant details exist
   - Related information counts - if the Context discusses the topic generally, include it
   - Do NOT be overly strict - err on the side of providing available information
`

	QueryClassificationSystemPrompt = `Classify the user's query into exactly one of the following categories.
Return only one word: PDFSearch, Clarification, or General.

Follow these strict rules in order:

1. PDFSearch
   The query must be classified as PDFSearch if ANY of the following are true:
     - The user asks for information that could be inside the document.
       (Examples: "what is X", "explain Y", "tell me about Z", "summarize section about A")
     - The query references content, topics, definitions, explanations, or facts.
     - The query is medical, scientific, legal, historical, or technical in nature.
     - You are uncertain which category it fits. (Default to PDFSearch)
   Examples:
     "Explain glycolysis"
     "What is classical conditioning?"
     "Find information about Osmanlı tarihi"

2. Clarification
   Classify as Clarification ONLY IF:
     - The query explicitly refers to your previous answer or user's previous question.
     - The user asks for elaboration, simplification, or more detail.
   Examples:
     "Explain more"
     "What do you mean?"
     "Can you clarify the last part?"
     "Continue"

3. General
   Classify as General ONLY IF:
     - The user is greeting, chatting, or asking unrelated personal/AI questions.
   Examples:
     "hello"
     "who are you?"
     "how are you?"
     "tell me a joke"
     "where do you live?"

	 Instructions:
	Respond with exactly one word: "PDFSearch", "Clarification", or "General". When uncertain, default to "PDFSearch" for any medical-related content.`

	QueryExpansionSystemPrompt = `You are a search query expansion specialist. Your task is to generate exactly 5 diverse search queries that capture different aspects of the user's original intent.

	Requirements:
	1. Preserve the core intent - do not change what the user is asking about
	2. Use different phrasings, synonyms, and perspectives
	3. Vary specificity (some broader, some more specific)
	4. Include different keyword combinations
	5. Do NOT add information not present in the original query
	6. Do NOT make assumptions about user context

	Respond ONLY with valid JSON in this exact format:
	{"queries": ["...", "...", "...", "...", "..."]}
`
)

type ChatRequest struct {
	Stream   bool          `json:"stream"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Text string `json:"text"`
}

type QueryExpansionResponse struct {
	Queries []string `json:"queries"`
}

type ChatMessage struct {
	Role     Role     `json:"role"`
	Content  string   `json:"content"`
	ImageUrl []string `json:"image_url,omitempty"`
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
	case ImageQuery:
		return imageQuerySystemMessage
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
