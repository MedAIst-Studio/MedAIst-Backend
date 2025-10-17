package domain

type ThirdPartyAIRequest struct {
	Stream bool                  `json:"stream"`
	Input  []ThirdPartyAIMessage `json:"input"`
}

type ThirdPartyAIResponse struct {
	Text string `json:"text"`
}

type ThirdPartyAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewThirdPartyAIRequest(stream bool, relevantText, userMessage string) *ThirdPartyAIRequest {
	return &ThirdPartyAIRequest{
		Stream: stream,
		Input:  getDefaultInputMessages(relevantText, userMessage),
	}
}

func NewThirdPartyAIMessage(role, content string) *ThirdPartyAIMessage {
	return &ThirdPartyAIMessage{
		Role:    role,
		Content: content,
	}
}

func getSystemMessage() *ThirdPartyAIMessage {
	return NewThirdPartyAIMessage(
		"system",
		"You are a helpful medical assistant. Answer the user's questions based on the provided medical document excerpts. If the information is not available in the excerpts, respond with 'I don't know.' Do not provide any medical advice beyond the scope of the document excerpts. Response in the same language as the question.",
	)
}

func getDefaultInputMessages(relevantText, userMessage string) []ThirdPartyAIMessage {
	return []ThirdPartyAIMessage{
		*getSystemMessage(),
		*NewThirdPartyAIMessage("user", "Here are the relevant excerpts from the medical document:\n"+relevantText),
		*NewThirdPartyAIMessage("user", userMessage),
	}
}

func (r *ThirdPartyAIRequest) AppendUserMessage(content string) {
	r.Input = append(r.Input, *NewThirdPartyAIMessage("user", content))
}

func (r *ThirdPartyAIRequest) AppendAIMessage(content string) {
	r.Input = append(r.Input, *NewThirdPartyAIMessage("assistant", content))
}

func (r *ThirdPartyAIRequest) AppendPastMessages(messages []ThirdPartyAIMessage) {
	r.Input = append(r.Input, messages...)
}
