package file

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type GetFileRequest struct {
	Id uuid.UUID `params:"id"`
}

type GetFileResponse struct {
	FileBytes []byte
}

type GetFileHandler struct {
	repo Repository
	fl   FileService
}

func NewGetFileHandler(repo Repository, fl FileService) *GetFileHandler {
	return &GetFileHandler{
		repo: repo,
		fl:   fl,
	}
}

// Handle godoc
//
//	@Summary		Get File
//	@Description	Retrieve a file by its ID
//	@Tags			files
//	@Accept			json
//	@Produce		application/pdf
//	@Param			id	path	string	true	"File ID"
//	@Success		200	{file}	file
//	@Failure		400
//	@Failure		404
//	@Failure		500
//	@Router			/files/{id} [get]
func (h *GetFileHandler) Handle(ctx context.Context, req *GetFileRequest) (*GetFileResponse, int, error) {
	userId := domain.GetUserID(ctx)

	hasFile, err := h.repo.DoesUserHaveFile(ctx, userId, req.Id)
	if err != nil {
		return nil, 500, err
	}

	if !hasFile {
		return nil, 403, domain.ErrForbidden
	}

	fileBytes, err := h.fl.GetFile(ctx, req.Id)
	if err != nil {
		return nil, 500, err
	}

	res := &GetFileResponse{
		FileBytes: fileBytes,
	}
	return res, 200, nil
}
