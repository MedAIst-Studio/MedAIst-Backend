package file

import (
	"context"
	"errors"
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
	Factory *FileProcessorFactory
	repo    Repository
	fl      FileService
	vb      VectorDB // Still needed for recover? Or move recover to a service? Let's check.
}

func NewUploadFileHandler(repo Repository, fl FileService, vb VectorDB, factory *FileProcessorFactory) *UploadFileHandler {
	return &UploadFileHandler{
		repo:    repo,
		fl:      fl,
		vb:      vb,
		Factory: factory,
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

	processor, err := h.Factory.GetProcessor(newFile.Name)
	if err != nil {
		return nil, 400, err
	}

	if err := h.fl.UploadFile(ctx, newFile, fileBytes); err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	if err := h.repo.SaveFileInformation(ctx, newFile); err != nil {
		h.fl.DeleteFile(ctx, "uploads/"+newFile.Id.String())
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	if err := processor.Process(ctx, newFile, fileBytes); err != nil {
		// If processing fails, we should probably rollback the file upload/save
		h.recover(ctx, newFile.Id)
		return nil, 500, err
	}

	return nil, 200, nil
}

func (h *UploadFileHandler) recover(ctx context.Context, fileId uuid.UUID) {
	h.repo.DeleteFileById(ctx, fileId)
	h.fl.DeleteFile(ctx, "uploads/"+fileId.String())
	h.vb.DeleteEmbeddingsByFileId(ctx, fileId)
}
