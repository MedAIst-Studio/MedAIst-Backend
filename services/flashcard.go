package services

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

type flashcardService struct {
	repo domain.FlashcardRepository
}

func NewFlashcardService(repo domain.FlashcardRepository) domain.FlashcardService {
	return &flashcardService{
		repo: repo,
	}
}

func (s *flashcardService) CreateDeck(ctx context.Context, userId uuid.UUID, req *domain.CreateDeckRequest) (*domain.CreateDeckResponse, error) {
	deck := &domain.Deck{
		Id:          uuid.New(),
		UserId:      userId,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := s.repo.SaveDeck(ctx, deck)
	if err != nil {
		return nil, err
	}

	return &domain.CreateDeckResponse{Id: deck.Id}, nil
}

func (s *flashcardService) GetDecks(ctx context.Context, userId uuid.UUID) (*domain.GetDecksResponse, error) {
	decks, err := s.repo.GetDecksByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &domain.GetDecksResponse{Decks: decks}, nil
}

func (s *flashcardService) AddCard(ctx context.Context, userId uuid.UUID, deckId uuid.UUID, req *domain.AddCardRequest) (*domain.AddCardResponse, error) {
	// First, check if the deck belongs to the user
	deck, err := s.repo.GetDeckById(ctx, deckId)
	if err != nil {
		return nil, err
	}
	if deck.UserId != userId {
		return nil, errors.New("unauthorized: deck does not belong to user")
	}

	card := &domain.Flashcard{
		Id:        uuid.New(),
		DeckId:    deckId,
		Front:     req.Front,
		Back:      req.Back,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.repo.SaveCard(ctx, card)
	if err != nil {
		return nil, err
	}

	return &domain.AddCardResponse{Id: card.Id}, nil
}

func (s *flashcardService) GetCards(ctx context.Context, userId uuid.UUID, deckId uuid.UUID) (*domain.GetCardsResponse, error) {
	// Optional: verify deck ownership here, though GetCardsByDeckId uses deckId.
	// Actually, GetCardsByDeckId requires userId to left join the correct reviews.
	deck, err := s.repo.GetDeckById(ctx, deckId)
	if err != nil {
		return nil, err
	}
	if deck.UserId != userId {
		return nil, errors.New("unauthorized: deck does not belong to user")
	}

	cards, err := s.repo.GetCardsByDeckId(ctx, deckId, userId)
	if err != nil {
		return nil, err
	}

	return &domain.GetCardsResponse{Cards: cards}, nil
}

func (s *flashcardService) GetDueCards(ctx context.Context, userId uuid.UUID, deckId uuid.UUID) (*domain.GetCardsResponse, error) {
	deck, err := s.repo.GetDeckById(ctx, deckId)
	if err != nil {
		return nil, err
	}
	if deck.UserId != userId {
		return nil, errors.New("unauthorized: deck does not belong to user")
	}

	cards, err := s.repo.GetDueCardsForDeck(ctx, deckId, userId, time.Now())
	if err != nil {
		return nil, err
	}

	return &domain.GetCardsResponse{Cards: cards}, nil
}

func (s *flashcardService) ReviewCard(ctx context.Context, userId uuid.UUID, cardId uuid.UUID, req *domain.ReviewCardRequest) (*domain.ReviewCardResponse, error) {
	// Verify card exists
	card, err := s.repo.GetCardById(ctx, cardId)
	if err != nil {
		return nil, err
	}

	// Verify deck ownership
	deck, err := s.repo.GetDeckById(ctx, card.DeckId)
	if err != nil {
		return nil, err
	}
	if deck.UserId != userId {
		return nil, errors.New("unauthorized: cannot review card not belonging to you")
	}

	review, err := s.repo.GetReviewByCardAndUserId(ctx, cardId, userId)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	if review == nil {
		review = &domain.FlashcardReview{
			Id:          uuid.New(),
			FlashcardId: cardId,
			UserId:      userId,
			EaseFactor:  2.5,
			Interval:    0,
			Repetitions: 0,
			CreatedAt:   now,
		}
	}

	// SM-2 Algorithm Calculation
	q := float32(req.Quality)
	
	if q < 3 {
		review.Repetitions = 0
		review.Interval = 1
	} else {
		if review.Repetitions == 0 {
			review.Interval = 1
		} else if review.Repetitions == 1 {
			review.Interval = 6
		} else {
			review.Interval = review.Interval * review.EaseFactor
		}
		review.Repetitions++
	}

	// Update EaseFactor
	review.EaseFactor = review.EaseFactor + (0.1 - (5-q)*(0.08+(5-q)*0.02))
	if review.EaseFactor < 1.3 {
		review.EaseFactor = 1.3
	}

	// Calculate next review date by rounding up days
	daysToAdd := math.Round(float64(review.Interval))
	review.NextReviewDate = now.Add(time.Duration(daysToAdd*24) * time.Hour)
	review.UpdatedAt = now

	err = s.repo.SaveReview(ctx, review)
	if err != nil {
		return nil, err
	}

	return &domain.ReviewCardResponse{
		NextReviewDate: review.NextReviewDate,
	}, nil
}
