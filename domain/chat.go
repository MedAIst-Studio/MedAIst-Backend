package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Chat struct {
	Id       uuid.UUID `json:"id"`
	UserId   uuid.UUID `json:"user_id"`
	Title    string    `json:"title"`
	CreateAt time.Time `json:"create_at"`
}

func NewChat(userId uuid.UUID, title string) *Chat {
	return &Chat{
		Id:       uuid.New(),
		UserId:   userId,
		Title:    title,
		CreateAt: time.Now(),
	}
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type QueryType string

const (
	PdfSearchQuery   QueryType = "pdf_search_query"
	ImageQuery       QueryType = "image_query"
	GeneralChatQuery QueryType = "general_chat_query"
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

type UpdateChatRequest struct {
	Title string `json:"title"`
}

type UpdateChatResponse struct {
}

type ChatMessage struct {
	Role     Role     `json:"role"`
	Content  string   `json:"content"`
	ImageUrl []string `json:"image_url,omitempty"`
}

// ---------- Chat DTOs ----------

type GetChatsRequest struct{}

type GetChatsResponse []ChatResponseDTO

type ChatResponseDTO struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type GetChatByIDRequest struct {
	ChatID uuid.UUID `params:"id"`
}

type GetChatByIDResponse []MessageDTO

type MessageDTO struct {
	IsUser          bool             `json:"is_user"`
	Content         string           `json:"content"`
	Timestamp       time.Time        `json:"timestamp"`
	ContextSnapshot *ContextSnapshot `json:"context_snapshot,omitempty"`
	Attachments     []Attachment     `json:"attachments,omitempty"`
}

type UpdateChatDTO struct {
	Id    uuid.UUID `params:"id"`
	Title string    `json:"title"`
}

type DeleteChatRequest struct {
	Id uuid.UUID `params:"id"`
}

type DeleteChatResponse struct{}

// ---------- Chat Interfaces ----------

type ChatRepository interface {
	CreateChat(ctx context.Context, chat *Chat) error
	GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]ChatResponseDTO, error)
	GetChatByID(ctx context.Context, chatId uuid.UUID) (*GetChatByIDResponse, error)
	DeleteChat(ctx context.Context, chatId uuid.UUID) error
	UpdateChat(ctx context.Context, chatId uuid.UUID, title string) error
}

type ChatService interface {
	GetChatsByUserID(ctx context.Context, userId uuid.UUID) ([]ChatResponseDTO, error)
	GetChatByID(ctx context.Context, chatId uuid.UUID) (*GetChatByIDResponse, error)
	UpdateChat(ctx context.Context, chatId uuid.UUID, title string) error
	DeleteChat(ctx context.Context, chatId uuid.UUID) error
}

type AIService interface {
	SendChatRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	GenerateChatTitle(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	ExpandQuery(ctx context.Context, content string) ([]string, error)
	RerankChunks(ctx context.Context, chunks []string, userQuestion string) ([]string, error)
	ExtractTextFromImage(ctx context.Context, imageUrl string) (string, error)
	SolveQuestionWithReasoning(ctx context.Context, questionText string) (string, error)
	GenerateQuizQuestion(ctx context.Context, chunk string) (*QuizQuestion, error)
}

type SingleEmbeddingRequest struct {
	Text string `json:"text"`
}

type SingleEmbeddingResponse struct {
	Vector []float32 `json:"vector"`
}

type SingleEmbeddingService interface {
	GenerateEmbedding(ctx context.Context, req *SingleEmbeddingRequest) (*SingleEmbeddingResponse, error)
}
