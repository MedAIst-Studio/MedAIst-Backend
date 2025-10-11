package file

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type Repository interface {
	SaveFileInformation(ctx context.Context, file *domain.File) error
}

type VectorDB interface {
	InsertDocument(ctx context.Context, id uuid.UUID, content []float32) error
}

type EmbeddingService interface {
	GenerateEmbeddingBatch(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error)
}

type FileService interface {
	UploadFile(ctx context.Context, file *domain.File, data []byte) error
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
