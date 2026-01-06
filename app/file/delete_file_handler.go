package file

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type DeleteFileRequest struct {
	Id uuid.UUID `params:"id"`
}

type DeleteFileResponse struct{}

type DeleteFileHandler struct {
	repo Repository
	fl   FileService
	vb   VectorDB
}

func NewDeleteFileHandler(repo Repository, fl FileService, vb VectorDB) *DeleteFileHandler {
	return &DeleteFileHandler{
		repo: repo,
		fl:   fl,
		vb:   vb,
	}
}

// Handle godoc
//
//	@Summary		Delete a file
//	@Description	Deletes a file and its associated data
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Param			id	path	string	true	"File ID"
//	@Success		204
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		500
//
//	@Security		BearerAuth
//
//	@Router			/files/{id} [delete]
func (h *DeleteFileHandler) Handle(ctx context.Context, req *DeleteFileRequest) (*DeleteFileResponse, int, error) {
	fileMeta, err := h.repo.GetFileMetadata(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrFileNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}

	// Delete from storage using path
	if err := h.fl.DeleteFile(ctx, fileMeta.Path); err != nil {
		return nil, 500, err
	}

	// Delete from DB
	if err := h.repo.DeleteFileById(ctx, req.Id); err != nil {
		if errors.Is(err, domain.ErrFileNotFound) {
			return nil, 404, err
		}
		return nil, 500, err
	}

	if err := h.vb.DeleteEmbeddingsByFileId(ctx, req.Id); err != nil {
		return nil, 500, err
	}

	return nil, 204, nil
}

func formatFilePath(fileId uuid.UUID) string {
	return "/uploads/" + fileId.String() + ".pdf"
}
