package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type FileService struct {
	repo      domain.FileRepository
	fl        domain.FileStorageService
	vb        domain.VectorDB
	streakSvc domain.StreakService
	factory   *FileProcessorFactory
}

func NewFileService(
	repo domain.FileRepository,
	fl domain.FileStorageService,
	vb domain.VectorDB,
	streakSvc domain.StreakService,
	factory *FileProcessorFactory,
) *FileService {
	return &FileService{
		repo:      repo,
		fl:        fl,
		vb:        vb,
		streakSvc: streakSvc,
		factory:   factory,
	}
}

func (s *FileService) UploadFile(ctx context.Context, userId uuid.UUID, fileTitle string, fileBytes []byte) (*domain.UploadFileResponse, error) {
	ext := strings.ToLower(filepath.Ext(fileTitle))
	newFileId := uuid.New()
	objectPath := fmt.Sprintf("uploads/%s%s", newFileId.String(), ext)

	newFile := &domain.File{
		Id:        newFileId,
		UserId:    userId,
		Name:      fileTitle,
		Size:      int64(len(fileBytes)),
		Path:      objectPath,
		CreatedAt: time.Now(),
	}

	processor, err := s.factory.GetProcessor(newFile.Name)
	if err != nil {
		return nil, err
	}

	fileUrl, err := s.fl.UploadFile(ctx, newFile, fileBytes)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveFileInformation(ctx, newFile); err != nil {
		s.fl.DeleteFile(ctx, newFile.Path)
		return nil, err
	}

	if err := processor.Process(ctx, newFile, fileBytes); err != nil {
		s.recover(ctx, newFile.Id, newFile.Path)
		return nil, err
	}

	s.streakSvc.CheckAndUpdateStreak(ctx, userId)

	return &domain.UploadFileResponse{Url: fileUrl}, nil
}

func (s *FileService) GetFilesByUserID(ctx context.Context, userId uuid.UUID) (*domain.GetFilesResponse, error) {
	files, err := s.repo.GetFilesByUserID(ctx, userId)
	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Url)
		}

		urls, err := s.fl.GetFiles(ctx, paths)
		if err != nil {
			return nil, err
		}

		for i := range files {
			files[i].Url = urls[i]
		}
	}

	response := domain.GetFilesResponse(files)
	return &response, nil
}

func (s *FileService) GetFile(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (*domain.GetFileResponse, error) {
	hasFile, err := s.repo.DoesUserHaveFile(ctx, userId, fileId)
	if err != nil {
		return nil, err
	}

	if !hasFile {
		return nil, domain.ErrForbidden
	}

	fileMeta, err := s.repo.GetFileMetadata(ctx, fileId)
	if err != nil {
		return nil, err
	}

	fileBytes, err := s.fl.GetFile(ctx, fileMeta.Path)
	if err != nil {
		return nil, err
	}

	res := &domain.GetFileResponse{
		FileBytes: fileBytes,
	}
	return res, nil
}

func (s *FileService) DeleteFile(ctx context.Context, fileId uuid.UUID) error {
	fileMeta, err := s.repo.GetFileMetadata(ctx, fileId)
	if err != nil {
		return err
	}

	if err := s.fl.DeleteFile(ctx, fileMeta.Path); err != nil {
		return err
	}

	if err := s.repo.DeleteFileById(ctx, fileId); err != nil {
		return err
	}

	if err := s.vb.DeleteEmbeddingsByFileId(ctx, fileId); err != nil {
		return err
	}

	return nil
}

func (s *FileService) ToggleFileSaved(ctx context.Context, userId uuid.UUID, fileId uuid.UUID) (*domain.ToggleFileSavedResponse, error) {
	hasFile, err := s.repo.DoesUserHaveFile(ctx, userId, fileId)
	if err != nil {
		return nil, err
	}

	if !hasFile {
		return nil, domain.ErrForbidden
	}

	isSaved, err := s.repo.ToggleFileSaved(ctx, fileId)
	if err != nil {
		return nil, err
	}

	return &domain.ToggleFileSavedResponse{IsSaved: isSaved}, nil
}

func (s *FileService) recover(ctx context.Context, fileId uuid.UUID, filePath string) {
	s.repo.DeleteFileById(ctx, fileId)
	s.fl.DeleteFile(ctx, filePath)
	s.vb.DeleteEmbeddingsByFileId(ctx, fileId)
}

// ---------- File Processors ----------

type FileProcessorFactory struct {
	PdfProcessor   domain.FileProcessor
	ImageProcessor domain.FileProcessor
}

func NewFileProcessorFactory(pdfProcessor domain.FileProcessor, imageProcessor domain.FileProcessor) *FileProcessorFactory {
	return &FileProcessorFactory{
		PdfProcessor:   pdfProcessor,
		ImageProcessor: imageProcessor,
	}
}

func (f *FileProcessorFactory) GetProcessor(filename string) (domain.FileProcessor, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".pdf":
		return f.PdfProcessor, nil
	case ".jpg", ".jpeg", ".png", ".webp":
		return f.ImageProcessor, nil
	default:
		return nil, errors.New("unsupported file type: " + ext)
	}
}

type PdfProcessor struct {
	Repo      domain.FileRepository
	Fl        domain.FileStorageService
	Vb        domain.VectorDB
	Ems       domain.EmbeddingService
	Chunker   domain.Chunker
	BatchSize int
	Normalize bool
}

func NewPdfProcessor(repo domain.FileRepository, fl domain.FileStorageService, vb domain.VectorDB, ems domain.EmbeddingService, chunker domain.Chunker, batchSize int, normalize bool) *PdfProcessor {
	return &PdfProcessor{
		Repo:      repo,
		Fl:        fl,
		Vb:        vb,
		Ems:       ems,
		Chunker:   chunker,
		BatchSize: batchSize,
		Normalize: normalize,
	}
}

func (p *PdfProcessor) Process(ctx context.Context, f *domain.File, content []byte) error {
	chunks, err := p.Chunker.ChunkPDF(ctx, content, f.Id.String())
	if err != nil {
		p.recover(ctx, f.Id)
		if !errors.Is(err, domain.ErrInternalServer) {
			return err
		}
		return err
	}

	for start := 0; start < len(chunks); start += p.BatchSize {
		end := int(math.Min(float64(start+p.BatchSize), float64(len(chunks))))

		batch := chunks[start:end]

		var embeddings []domain.Embedding
		for _, chunk := range batch {
			var embedding domain.Embedding
			embedding.Id = uuid.New()
			embedding.PageNumbers = chunk.Metadata.PageNumbers
			embedding.Text = chunk.Text
			embeddings = append(embeddings, embedding)
		}

		var texts []domain.Text
		for _, emb := range embeddings {
			texts = append(texts, domain.Text{
				Id:   emb.Id,
				Text: emb.Text,
			})
		}

		embeddingReq := &domain.EmbeddingBatchRequest{
			Texts:     texts,
			Normalize: p.Normalize,
			BatchSize: p.BatchSize,
		}

		embeddingRes, err := p.Ems.GenerateEmbeddingBatch(ctx, embeddingReq)
		if err != nil {
			p.recover(ctx, f.Id)
			return err
		}

		embeddingMap := make(map[uuid.UUID]*domain.Embedding)
		for i := range embeddings {
			embeddingMap[embeddings[i].Id] = &embeddings[i]
		}

		for _, resultEmb := range embeddingRes.Results {
			if emb, exists := embeddingMap[resultEmb.Id]; exists {
				emb.Vector = resultEmb.Vector
			} else {
				p.recover(ctx, f.Id)
				return errors.New("embedding ID not found in batch")
			}
		}

		for _, emb := range embeddings {
			if len(emb.Vector) == 0 {
				p.recover(ctx, f.Id)
				return errors.New("embedding has no vector")
			}
		}

		var fileMetadatas []*domain.FileMetadata
		for _, emb := range embeddings {
			fm := &domain.FileMetadata{
				Id:          uuid.New(),
				Text:        emb.Text,
				PageNumbers: emb.PageNumbers,
				Vector:      emb.Vector,
			}
			fileMetadatas = append(fileMetadatas, fm)
		}

		if err := p.Vb.InsertEmbeddings(ctx, f.Id, fileMetadatas); err != nil {
			p.recover(ctx, f.Id)
			return err
		}
	}

	return nil
}

func (p *PdfProcessor) recover(ctx context.Context, fileId uuid.UUID) {
	p.Repo.DeleteFileById(ctx, fileId)
	p.Fl.DeleteFile(ctx, "/uploads/"+fileId.String()+".pdf")
	p.Vb.DeleteEmbeddingsByFileId(ctx, fileId)
}

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func (p *ImageProcessor) Process(ctx context.Context, file *domain.File, content []byte) error {
	return nil
}

func deduplicateChunks(chunks []string) []string {
	seen := make(map[string]bool)
	unique := make([]string, 0, len(chunks))

	for _, chunk := range chunks {
		if !seen[chunk] {
			seen[chunk] = true
			unique = append(unique, chunk)
		}
	}

	return unique
}

func getFileExtension(filename string) string {
	if strings.Contains(filename, ".") {
		return filename[strings.LastIndex(filename, "."):]
	}
	return ""
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(getFileExtension(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	default:
		return false
	}
}
