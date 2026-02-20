package file

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveFileInformation(ctx context.Context, file *domain.File) error
	GetFilesByUserID(ctx context.Context, userId uuid.UUID) ([]File, error)
	DeleteFileById(ctx context.Context, id uuid.UUID) error
	DoesUserHaveFile(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (bool, error)
	GetFileMetadata(ctx context.Context, fileId uuid.UUID) (*domain.File, error)
}

type FileMetadata struct {
	Id          uuid.UUID
	Text        string
	PageNumbers []int
	Vector      []float32
}

type VectorDB interface {
	InsertEmbeddings(ctx context.Context, fileId uuid.UUID, docs []*FileMetadata) error
	DeleteEmbeddingsByFileId(ctx context.Context, fileId uuid.UUID) error
}

type EmbeddingService interface {
	GenerateEmbeddingBatch(ctx context.Context, req *EmbeddingBatchRequest) (*EmbeddingResponse, error)
}

type FileService interface {
	UploadFile(ctx context.Context, file *domain.File, data []byte) (string, error)
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	DeleteFiles(ctx context.Context, filePaths []string) error
	GetFiles(ctx context.Context, filePaths []string) ([]string, error)
}

type Chunker interface {
	ChunkPDF(ctx context.Context, fileBytes []byte, fileID string) ([]Chunk, error)
}

type Chunk struct {
	Id       int
	Text     string
	Metadata ChunkMetadata
}

type ChunkMetadata struct {
	FileID      string
	ChunkIndex  int
	PageNumbers []int
	CharCount   int
}

type ChunkOptions struct {
	MaxChunkSize    int
	OverlapSize     int
	PreserveContext bool
}

type EmbeddingBatchRequest struct {
	Texts     []Text `json:"texts"`
	Normalize bool   `json:"normalize"`
	BatchSize int    `json:"batch_size"`
}

type Text struct {
	Id   uuid.UUID `json:"id"`
	Text string    `json:"text"`
}

type EmbeddingResponse struct {
	Results   []Embedding `json:"results"`
	Dimension int         `json:"dimension"`
	Count     int         `json:"count"`
}

type Embedding struct {
	Id          uuid.UUID `json:"id"`
	Text        string    `json:"text"`
	Vector      []float32 `json:"vector"`
	PageNumbers []int     `json:"page_numbers"`
}
