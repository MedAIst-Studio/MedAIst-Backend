package flashcard

import (
	"context"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
)

type FlashcardConfig struct {
	FlashcardService domain.FlashcardService
}

type FlashcardHandler struct {
	service domain.FlashcardService
}

func NewFlashcardHandler(config *FlashcardConfig) *FlashcardHandler {
	return &FlashcardHandler{
		service: config.FlashcardService,
	}
}

// @Summary		Create Deck
// @Description	Create a new flashcard deck
// @Tags			Flashcards
// @Accept			json
// @Produce		json
// @Param			request	body		domain.CreateDeckRequest	true	"Create Deck request"
// @Success		200		{object}	domain.CreateDeckResponse
// @Failure		400
// @Failure		500
// @Router			/decks [post]
func (h *FlashcardHandler) HandleCreateDeck(ctx context.Context, req *domain.CreateDeckRequest) (*domain.CreateDeckResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.CreateDeck(ctx, userId, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Decks
// @Description	Get all flashcard decks for the current user
// @Tags			Flashcards
// @Produce		json
// @Success		200	{object}	domain.GetDecksResponse
// @Failure		500
// @Router			/decks [get]
func (h *FlashcardHandler) HandleGetDecks(ctx context.Context, req *domain.GetDecksRequest) (*domain.GetDecksResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.GetDecks(ctx, userId)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Add Card
// @Description	Add a new flashcard to a deck
// @Tags			Flashcards
// @Accept			json
// @Produce		json
// @Param			id		path		string					true	"Deck ID"
// @Param			request	body		domain.AddCardRequest	true	"Add Card request"
// @Success		200		{object}	domain.AddCardResponse
// @Failure		400
// @Failure		500
// @Router			/decks/{id}/cards [post]
func (h *FlashcardHandler) HandleAddCard(ctx context.Context, req *domain.AddCardRequest) (*domain.AddCardResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.AddCard(ctx, userId, req.DeckId, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Cards
// @Description	Get all cards in a deck
// @Tags			Flashcards
// @Produce		json
// @Param			id	path		string	true	"Deck ID"
// @Success		200	{object}	domain.GetCardsResponse
// @Failure		500
// @Router			/decks/{id}/cards [get]
func (h *FlashcardHandler) HandleGetCards(ctx context.Context, req *domain.GetCardsRequest) (*domain.GetCardsResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.GetCards(ctx, userId, req.DeckId)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Get Due Cards
// @Description	Get cards due for review in a deck based on spaced repetition algorithm
// @Tags			Flashcards
// @Produce		json
// @Param			id	path		string	true	"Deck ID"
// @Success		200	{object}	domain.GetCardsResponse
// @Failure		500
// @Router			/decks/{id}/review [get]
func (h *FlashcardHandler) HandleGetDueCards(ctx context.Context, req *domain.GetDueCardsRequest) (*domain.GetCardsResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.GetDueCards(ctx, userId, req.DeckId)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}

// @Summary		Review Card
// @Description	Submit a review for a flashcard (0-5 quality score)
// @Tags			Flashcards
// @Accept			json
// @Produce		json
// @Param			id		path		string					true	"Card ID"
// @Param			request	body		domain.ReviewCardRequest	true	"Review Card request"
// @Success		200		{object}	domain.ReviewCardResponse
// @Failure		400
// @Failure		500
// @Router			/cards/{id}/review [post]
func (h *FlashcardHandler) HandleReviewCard(ctx context.Context, req *domain.ReviewCardRequest) (*domain.ReviewCardResponse, int, error) {
	userId := domain.GetUserID(ctx)
	resp, err := h.service.ReviewCard(ctx, userId, req.CardId, req)
	if err != nil {
		return nil, 500, err
	}
	return resp, 200, nil
}
