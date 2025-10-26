package user

import (
	"context"
	"net/http"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type DeleteAccountRequest struct {
}

type DeleteAccountResponse struct{}

type DeleteAccountHandler struct {
	repo   Repository
	logger domain.Logger
	fs     FileService
	vb     VectorRepository
}

func NewDeleteAccountHandler(repo Repository, logger domain.Logger, fs FileService, vb VectorRepository) *DeleteAccountHandler {
	return &DeleteAccountHandler{repo: repo, logger: logger, fs: fs, vb: vb}
}

// Handle processes the request to delete a user's account.
//
//	@Summary		Delete User Account
//
//	@Description	Delete a user's account
//	@Tags			User
//	@Security		BearerAuth
//	@Success		204
//	@Failure		400
//	@Failure		401
//	@Failure		500
//	@Router			/users/account [delete]
func (h *DeleteAccountHandler) Handle(ctx context.Context, req *DeleteAccountRequest) (*DeleteAccountResponse, int, error) {

	userId := domain.GetUserID(ctx)

	fileUUIDs, err := h.repo.DeleteAccount(ctx, userId)
	if err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	fileIds := convertToStringSlice(fileUUIDs)

	if err := h.fs.DeleteFiles(ctx, fileIds); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	if err := h.vb.DeleteEmbeddingsByFileIDs(ctx, fileIds); err != nil {
		return nil, http.StatusInternalServerError, domain.ErrInternalServer
	}

	return nil, http.StatusNoContent, nil
}

func convertToStringSlice(uuids []uuid.UUID) []string {
	var strIds []string
	for _, id := range uuids {
		strIds = append(strIds, id.String())
	}
	return strIds
}
