package file

import (
	"context"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type GetFilesRequest struct {
}

type GetFilesResponse []File

type File struct {
	Id        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Size      int64     `json:"size"`
	Url       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type GetFilesHandler struct {
	repo Repository
	fs   FileService
}

func NewGetFilesHandler(repo Repository, fs FileService) *GetFilesHandler {
	return &GetFilesHandler{
		repo: repo,
		fs:   fs,
	}
}

// godoc
// Handle processes a GetFilesRequest and returns a GetFilesResponse.
//
//	@Summary		Retrieve files for the authenticated user
//	@Description	Retrieves a list of files associated with the authenticated user.
//	@Tags			Files
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetFilesResponse
//	@Failure		500
//	@Security		BearerAuth
//	@Router			/files [get]
func (h *GetFilesHandler) Handle(ctx context.Context, req *GetFilesRequest) (*GetFilesResponse, int, error) {
	userId := domain.GetUserID(ctx)
	files, err := h.repo.GetFilesByUserID(ctx, userId)
	if err != nil {
		return nil, 500, err
	}

	if len(files) > 0 {
		var paths []string
		for _, f := range files {
			paths = append(paths, f.Url)
		}

		urls, err := h.fs.GetFiles(ctx, paths)
		if err != nil {
			return nil, 500, err
		}

		for i := range files {
			files[i].Url = urls[i]
		}
	}

	response := GetFilesResponse(files)
	return &response, 200, nil
}
