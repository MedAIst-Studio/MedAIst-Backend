package file

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type UploadFileRequest struct {
	FileTitle string                `form:"file_title"`
	FileData  *multipart.FileHeader `form:"file_data"`
}

type UploadFileResponse struct {
}

type UploadFileHandler struct {
	repo Repository
	fl   FileService
	vb   VectorDB
	ems  EmbeddingService
}

func NewUploadFileHandler(repo Repository, fl FileService) *UploadFileHandler {
	return &UploadFileHandler{
		repo: repo,
		fl:   fl,
		// vb:   vb,
		// ems:  ems,
	}
}

// Handle godoc
//
//	@Summary		Upload a file
//	@Description	Upload a file to the server
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Param			request	body		UploadFileRequest	true	"Request body"
//	@Success		200		{object}	UploadFileResponse
//	@Failure		400		{object}	domain.Error
//	@Failure		500		{object}	domain.Error
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

	// chunk files
	// we will get something like this
	// var chunks []string

	// // embedding
	// var embeddings [][]float32
	// for _, chunk := range chunks {
	// 	// is each loop a go routine and api call to embedding service?
	// 	embedding, err := h.ems.GenerateEmbedding(chunk)
	// 	if err != nil {
	// 		if !errors.Is(err, domain.ErrInternalServer) {
	// 			return nil, 400, err
	// 		}
	// 		return nil, 500, err
	// 	}
	// 	embeddings = append(embeddings, embedding)
	// }

	// // vector db
	// for _, embedding := range embeddings {
	// 	if err := h.vb.InsertDocument(file.Id, embedding); err != nil {
	// 		if !errors.Is(err, domain.ErrInternalServer) {
	// 			return nil, 400, err
	// 		}
	// 		return nil, 500, err
	// 	}
	// }

	return nil, 200, nil
}
