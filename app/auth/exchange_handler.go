package auth

import (
	"context"
	"errors"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type ExchangeRequest struct {
	SessionKey string `json:"session"`
}

type ExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ExchangeHandler struct {
	cache Cache
}

func NewExchangeHandler(cache Cache) *ExchangeHandler {
	return &ExchangeHandler{
		cache: cache,
	}
}

// godoc
//
//	@summary	Exchange OAuth2 tokens
//	@accept		json
//	@produce	json
//	@param		request	body		ExchangeRequest	true	"Exchange Request"
//	@success	200		{object}	ExchangeResponse
//	@failure	400
//	@failure	500
//	@router		/api/auth/exchange [post]
func (h *ExchangeHandler) Handle(ctx context.Context, req *ExchangeRequest) (*ExchangeResponse, int, error) {
	if req.SessionKey == "" {
		return nil, 400, errors.New("session key is required")
	}

	session, err := h.cache.GetOAuthSession(ctx, req.SessionKey)
	if err != nil {
		if !errors.Is(err, domain.ErrInternalServer) {
			return nil, 400, err
		}
		return nil, 500, err
	}

	h.cache.DeleteOAuthSession(ctx, req.SessionKey)

	return &ExchangeResponse{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
	}, 200, nil
}
