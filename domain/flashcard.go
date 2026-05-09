package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ---------- Flashcard Models ----------

type Deck struct {
	Id          uuid.UUID `json:"id"`
	UserId      uuid.UUID `json:"userId"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Flashcard struct {
	Id        uuid.UUID `json:"id"`
	DeckId    uuid.UUID `json:"deckId"`
	Front     string    `json:"front"`
	Back      string    `json:"back"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FlashcardReview struct {
	Id             uuid.UUID `json:"id"`
	FlashcardId    uuid.UUID `json:"flashcardId"`
	UserId         uuid.UUID `json:"userId"`
	NextReviewDate time.Time `json:"nextReviewDate"`
	Interval       float32   `json:"interval"`   // in days
	EaseFactor     float32   `json:"easeFactor"` // SM-2 ease factor
	Repetitions    int       `json:"repetitions"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Struct to combine flashcard with its review state for the client
type FlashcardWithReview struct {
	Flashcard
	Review *FlashcardReview `json:"review,omitempty"`
}

// ---------- Request and Response DTOs ----------

type CreateDeckRequest struct {
	Title       string `json:"title" validate:"required"`
	Description string `json:"description,omitempty"`
}

type CreateDeckResponse struct {
	Id uuid.UUID `json:"id"`
}

type AddCardRequest struct {
	DeckId uuid.UUID `param:"id" validate:"required,uuid"`
	Front  string    `json:"front" validate:"required"`
	Back   string    `json:"back" validate:"required"`
}

type AddCardResponse struct {
	Id uuid.UUID `json:"id"`
}

type ReviewCardRequest struct {
	CardId  uuid.UUID `param:"id" validate:"required,uuid"`
	Quality int       `json:"quality" validate:"required,min=0,max=5"` // 0 to 5 score
}

type ReviewCardResponse struct {
	NextReviewDate time.Time `json:"nextReviewDate"`
}

type GetDecksRequest struct{}

type GetDecksResponse struct {
	Decks []Deck `json:"decks"`
}

type GetCardsRequest struct {
	DeckId uuid.UUID `param:"id" validate:"required,uuid"`
}

type GetCardsResponse struct {
	Cards []FlashcardWithReview `json:"cards"`
}

type GetDueCardsRequest struct {
	DeckId uuid.UUID `param:"id" validate:"required,uuid"`
}

// ---------- Interfaces ----------

type FlashcardRepository interface {
	SaveDeck(ctx context.Context, deck *Deck) error
	GetDecksByUserId(ctx context.Context, userId uuid.UUID) ([]Deck, error)
	GetDeckById(ctx context.Context, deckId uuid.UUID) (*Deck, error)
	DeleteDeck(ctx context.Context, deckId uuid.UUID) error

	SaveCard(ctx context.Context, card *Flashcard) error
	GetCardsByDeckId(ctx context.Context, deckId uuid.UUID, userId uuid.UUID) ([]FlashcardWithReview, error)
	GetCardById(ctx context.Context, cardId uuid.UUID) (*Flashcard, error)
	DeleteCard(ctx context.Context, cardId uuid.UUID) error

	SaveReview(ctx context.Context, review *FlashcardReview) error
	GetReviewByCardAndUserId(ctx context.Context, cardId uuid.UUID, userId uuid.UUID) (*FlashcardReview, error)
	GetDueCardsForDeck(ctx context.Context, deckId uuid.UUID, userId uuid.UUID, currentTime time.Time) ([]FlashcardWithReview, error)
}

type FlashcardService interface {
	CreateDeck(ctx context.Context, userId uuid.UUID, req *CreateDeckRequest) (*CreateDeckResponse, error)
	GetDecks(ctx context.Context, userId uuid.UUID) (*GetDecksResponse, error)
	
	AddCard(ctx context.Context, userId uuid.UUID, deckId uuid.UUID, req *AddCardRequest) (*AddCardResponse, error)
	GetCards(ctx context.Context, userId uuid.UUID, deckId uuid.UUID) (*GetCardsResponse, error)
	
	GetDueCards(ctx context.Context, userId uuid.UUID, deckId uuid.UUID) (*GetCardsResponse, error)
	ReviewCard(ctx context.Context, userId uuid.UUID, cardId uuid.UUID, req *ReviewCardRequest) (*ReviewCardResponse, error)
}
