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
	UploadFile(ctx context.Context, file *domain.File, data []byte) error
	DeleteFile(ctx context.Context, filePath string) error
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
