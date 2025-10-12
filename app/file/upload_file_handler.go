package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type UploadFileRequest struct {
	FileTitle string                `form:"file_title"`
	FileData  *multipart.FileHeader `form:"file_data"`
}

type UploadFileResponse struct {
}

type UploadFileHandler struct {
	BatchSize int
	Normalize bool
	repo      Repository
	fl        FileService
	vb        VectorDB
	ems       EmbeddingService
	chunker   Chunker
}

type EmbeddingRequest struct {
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
	Model     string      `json:"model"`
}

type Embedding struct {
	Id          uuid.UUID `json:"id"`
	Text        string    `json:"text"`
	Vector      []float32 `json:"vector"`
	PageNumbers []int     `json:"page_numbers"`
}

func NewUploadFileHandler(repo Repository, fl FileService, vb VectorDB, ems EmbeddingService, chunker Chunker, batchSize int, normalize bool) *UploadFileHandler {
	return &UploadFileHandler{
		repo:      repo,
		fl:        fl,
		vb:        vb,
		ems:       ems,
		chunker:   chunker,
		BatchSize: batchSize,
		Normalize: normalize,
	}
}

// Handle godoc
//
//	@Summary		Upload a file
//	@Description	Upload a file to the server
//	@Description	This endpoint allows authenticated users to upload a file. The file will be processed and stored.
//	@Tags			Files
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			request	body		UploadFileRequest	true	"Request body"
//	@Success		200		{object}	UploadFileResponse
//	@Failure		400
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/files/upload [post]
func (h *UploadFileHandler) Handle(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, int, error) {
	file, err := req.FileData.Open()
	if err != nil {
		return nil, 500, err
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, 500, err
	}

	newFile, err := domain.NewFile(&domain.FileRequires{
		UserId:    domain.GetUserID(ctx),
		Name:      req.FileTitle,
		Size:      int64(len(fileBytes)),
		Extension: filepath.Ext(req.FileTitle),
	})

	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	// to local via min.io
	if err := h.fl.UploadFile(ctx, newFile, fileBytes); err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	// postgres
	if err := h.repo.SaveFileInformation(ctx, newFile); err != nil {
		// TODO delete from min.io
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	chunks, err := h.chunker.ChunkPDF(ctx, fileBytes, newFile.Id.String())
	if err != nil {
		return nil, 500, err
	}

	fmt.Printf("Generated %d chunks for file %s\n", len(chunks), newFile.Name)
	fmt.Printf("Chunk metadata for file: %v\n", chunks[10].Metadata)
	fmt.Printf("Chunk text for file: %v\n", chunks[10].Text)

	for start := 0; start < len(chunks); start += h.BatchSize {
		end := min(start+h.BatchSize, len(chunks))

		batch := chunks[start:end]

		// text ve embedding vectorleri bir arada tutabilmek icin EmbeddingRequest e id veriyoruz.
		var embeddings []Embedding
		for _, chunk := range batch {
			var embedding Embedding
			embedding.Id = uuid.New()
			embedding.PageNumbers = chunk.Metadata.PageNumbers
			embedding.Text = chunk.Text
			embeddings = append(embeddings, embedding)
		}

		// embedding api nin ihtiyaci olan text yapisi
		var texts []Text
		for _, emb := range embeddings {
			texts = append(texts, Text{
				Id:   emb.Id,
				Text: emb.Text,
			})
		}

		// requestimiz burada
		embeddingReq := &EmbeddingRequest{
			Texts:     texts,
			Normalize: h.Normalize,
			BatchSize: h.BatchSize,
		}

		embeddingRes, err := h.ems.GenerateEmbeddingBatch(ctx, embeddingReq)
		if err != nil {
			return nil, 500, err
		}

		// id ler uzerinden ilerleyecegimiz icin map e ceviriyoruz
		embeddingMap := make(map[uuid.UUID]*Embedding)
		for i := range embeddings {
			embeddingMap[embeddings[i].Id] = &embeddings[i]
		}

		for _, resultEmb := range embeddingRes.Results {
			if emb, exists := embeddingMap[resultEmb.Id]; exists {
				emb.Vector = resultEmb.Vector
			} else {
				return nil, 500, fmt.Errorf("embedding ID %s not found in batch", resultEmb.Id)
			}
		}

		for _, emb := range embeddings {
			if len(emb.Vector) == 0 {
				return nil, 500, fmt.Errorf("embedding %s has no vector", emb.Id)
			}
		}

		var fileMetadatas []*FileMetadata
		for _, emb := range embeddings {
			fm := &FileMetadata{
				Id:          uuid.New(),
				Text:        emb.Text,
				PageNumbers: emb.PageNumbers,
				Vector:      emb.Vector,
			}
			fileMetadatas = append(fileMetadatas, fm)
		}

		if err := h.vb.InsertEmbeddings(ctx, newFile.Id, fileMetadatas); err != nil {
			return nil, 500, err
		}

	}

	return nil, 200, nil
}
