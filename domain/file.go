package domain

import (
	"context"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ---------- File Type Constants ----------

type FileType string

const (
	FileTypePDF   FileType = "pdf"
	FileTypeImage FileType = "image"
)

// DetectFileType returns the file type based on the file extension.
func DetectFileType(filename string) FileType {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return FileTypeImage
	default:
		return FileTypePDF
	}
}

// ---------- File Model ----------

type File struct {
	Id        uuid.UUID
	UserId    uuid.UUID
	Name      string
	Size      int64
	Path      string
	CreatedAt time.Time
	FileType  FileType
	IsSaved   bool
}

// ---------- File DTOs ----------

type UploadFileRequest struct {
	FileData *multipart.FileHeader `form:"file_data"`
}

type UploadFileResponse struct {
	Url string `json:"url"`
}

type GetFilesRequest struct{}

type GetFilesResponse []FileResponseDTO

type FileResponseDTO struct {
	Id        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Size      int64     `json:"size"`
	Url       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	FileType  FileType  `json:"file_type"`
	IsSaved   bool      `json:"is_saved"`
}

type GetFileRequest struct {
	Id uuid.UUID `params:"id"`
}

type GetFileResponse struct {
	FileBytes []byte
}

type DeleteFileRequest struct {
	Id uuid.UUID `params:"id"`
}

type DeleteFileResponse struct{}

type ToggleFileSavedRequest struct {
	Id uuid.UUID `params:"id"`
}

type ToggleFileSavedResponse struct {
	IsSaved bool `json:"is_saved"`
}

// ---------- File Interfaces ----------

type FileRepository interface {
	SaveFileInformation(ctx context.Context, file *File) error
	GetFilesByUserID(ctx context.Context, userId uuid.UUID) ([]FileResponseDTO, error)
	DeleteFileById(ctx context.Context, id uuid.UUID) error
	DoesUserHaveFile(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (bool, error)
	GetFileMetadata(ctx context.Context, fileId uuid.UUID) (*File, error)
	ToggleFileSaved(ctx context.Context, fileId uuid.UUID) (bool, error)
	GetExpiredUnsavedImages(ctx context.Context, olderThan time.Time) ([]File, error)
	DeleteFilesByIDs(ctx context.Context, ids []uuid.UUID) error
	GetUserPDFCount(ctx context.Context, userID uuid.UUID) (int, error)
}

type FileStorageService interface {
	UploadFile(ctx context.Context, file *File, data []byte) (string, error)
	DeleteFile(ctx context.Context, filePath string) error
	GetFile(ctx context.Context, filePath string) ([]byte, error)
	DeleteFiles(ctx context.Context, filePaths []string) error
	GetFiles(ctx context.Context, filePaths []string) ([]string, error)
}

type FileService interface {
	UploadFile(ctx context.Context, userId uuid.UUID, fileTitle string, fileBytes []byte) (*UploadFileResponse, error)
	GetFilesByUserID(ctx context.Context, userId uuid.UUID) (*GetFilesResponse, error)
	GetFile(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (*GetFileResponse, error)
	DeleteFile(ctx context.Context, fileId uuid.UUID) error
	ToggleFileSaved(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (*ToggleFileSavedResponse, error)
}

// ---------- Embedding & Chunker Interfaces and Models ----------

type FileMetadata struct {
	Id          uuid.UUID
	Text        string
	PageNumbers []int
	Vector      []float32
}

type VectorDB interface {
	InsertEmbeddings(ctx context.Context, fileId uuid.UUID, docs []*FileMetadata) error
	DeleteEmbeddingsByFileId(ctx context.Context, fileId uuid.UUID) error
	DeleteEmbeddingsByFileIDs(ctx context.Context, fileIds []string) error
	SearchByEmbedding(ctx context.Context, fileId uuid.UUID, embedding []float32) ([]string, error)
}

type EmbeddingService interface {
	GenerateEmbeddingBatch(ctx context.Context, req *EmbeddingBatchRequest) (*EmbeddingResponse, error)
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

// ---------- File Processor Interfaces ----------

type FileProcessor interface {
	Process(ctx context.Context, file *File, content []byte) error
}
