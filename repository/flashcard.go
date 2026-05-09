package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) SaveDeck(ctx context.Context, deck *domain.Deck) error {
	query := `
		INSERT INTO flashcard_decks (id, user_id, title, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, deck.Id, deck.UserId, deck.Title, deck.Description, deck.CreatedAt, deck.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save deck: %w", err)
	}
	return nil
}

func (r *Repository) GetDecksByUserId(ctx context.Context, userId uuid.UUID) ([]domain.Deck, error) {
	query := `
		SELECT id, user_id, title, description, created_at, updated_at
		FROM flashcard_decks
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query decks: %w", err)
	}
	defer rows.Close()

	var decks []domain.Deck
	for rows.Next() {
		var deck domain.Deck
		var desc sql.NullString
		if err := rows.Scan(&deck.Id, &deck.UserId, &deck.Title, &desc, &deck.CreatedAt, &deck.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan deck: %w", err)
		}
		if desc.Valid {
			deck.Description = desc.String
		}
		decks = append(decks, deck)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over decks: %w", err)
	}
	if decks == nil {
		decks = make([]domain.Deck, 0)
	}
	return decks, nil
}

func (r *Repository) GetDeckById(ctx context.Context, deckId uuid.UUID) (*domain.Deck, error) {
	query := `
		SELECT id, user_id, title, description, created_at, updated_at
		FROM flashcard_decks
		WHERE id = $1
	`
	var deck domain.Deck
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx, query, deckId).Scan(&deck.Id, &deck.UserId, &deck.Title, &desc, &deck.CreatedAt, &deck.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("deck not found")
		}
		return nil, fmt.Errorf("failed to query deck: %w", err)
	}
	if desc.Valid {
		deck.Description = desc.String
	}
	return &deck, nil
}

func (r *Repository) DeleteDeck(ctx context.Context, deckId uuid.UUID) error {
	query := `DELETE FROM flashcard_decks WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, deckId)
	return err
}

func (r *Repository) SaveCard(ctx context.Context, card *domain.Flashcard) error {
	query := `
		INSERT INTO flashcards (id, deck_id, front, back, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query, card.Id, card.DeckId, card.Front, card.Back, card.CreatedAt, card.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save card: %w", err)
	}
	return nil
}

func (r *Repository) GetCardsByDeckId(ctx context.Context, deckId uuid.UUID, userId uuid.UUID) ([]domain.FlashcardWithReview, error) {
	query := `
		SELECT 
			f.id, f.deck_id, f.front, f.back, f.created_at, f.updated_at,
			r.id, r.flashcard_id, r.user_id, r.next_review_date, r.interval, r.ease_factor, r.repetitions, r.created_at, r.updated_at
		FROM flashcards f
		LEFT JOIN flashcard_reviews r ON f.id = r.flashcard_id AND r.user_id = $2
		WHERE f.deck_id = $1
		ORDER BY f.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, deckId, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards: %w", err)
	}
	defer rows.Close()

	var cards []domain.FlashcardWithReview
	for rows.Next() {
		var c domain.FlashcardWithReview
		var rId, rCardId, rUserId sql.NullString
		var rNextReview, rCreatedAt, rUpdatedAt sql.NullTime
		var rInterval, rEaseFactor sql.NullFloat64
		var rRepetitions sql.NullInt32

		err := rows.Scan(
			&c.Id, &c.DeckId, &c.Front, &c.Back, &c.CreatedAt, &c.UpdatedAt,
			&rId, &rCardId, &rUserId, &rNextReview, &rInterval, &rEaseFactor, &rRepetitions, &rCreatedAt, &rUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}

		if rId.Valid {
			parsedUserId, _ := uuid.Parse(rUserId.String)
			parsedId, _ := uuid.Parse(rId.String)
			parsedCardId, _ := uuid.Parse(rCardId.String)
			
			c.Review = &domain.FlashcardReview{
				Id:             parsedId,
				FlashcardId:    parsedCardId,
				UserId:         parsedUserId,
				NextReviewDate: rNextReview.Time,
				Interval:       float32(rInterval.Float64),
				EaseFactor:     float32(rEaseFactor.Float64),
				Repetitions:    int(rRepetitions.Int32),
				CreatedAt:      rCreatedAt.Time,
				UpdatedAt:      rUpdatedAt.Time,
			}
		}

		cards = append(cards, c)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over cards: %w", err)
	}
	if cards == nil {
		cards = make([]domain.FlashcardWithReview, 0)
	}

	return cards, nil
}

func (r *Repository) GetCardById(ctx context.Context, cardId uuid.UUID) (*domain.Flashcard, error) {
	query := `
		SELECT id, deck_id, front, back, created_at, updated_at
		FROM flashcards
		WHERE id = $1
	`
	var card domain.Flashcard
	err := r.db.QueryRowContext(ctx, query, cardId).Scan(&card.Id, &card.DeckId, &card.Front, &card.Back, &card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("card not found")
		}
		return nil, fmt.Errorf("failed to query card: %w", err)
	}
	return &card, nil
}

func (r *Repository) DeleteCard(ctx context.Context, cardId uuid.UUID) error {
	query := `DELETE FROM flashcards WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, cardId)
	return err
}

func (r *Repository) SaveReview(ctx context.Context, review *domain.FlashcardReview) error {
	query := `
		INSERT INTO flashcard_reviews (id, flashcard_id, user_id, next_review_date, interval, ease_factor, repetitions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (flashcard_id, user_id) 
		DO UPDATE SET 
			next_review_date = EXCLUDED.next_review_date,
			interval = EXCLUDED.interval,
			ease_factor = EXCLUDED.ease_factor,
			repetitions = EXCLUDED.repetitions,
			updated_at = EXCLUDED.updated_at
	`
	_, err := r.db.ExecContext(ctx, query, review.Id, review.FlashcardId, review.UserId, review.NextReviewDate, review.Interval, review.EaseFactor, review.Repetitions, review.CreatedAt, review.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save flashcard review: %w", err)
	}
	return nil
}

func (r *Repository) GetReviewByCardAndUserId(ctx context.Context, cardId uuid.UUID, userId uuid.UUID) (*domain.FlashcardReview, error) {
	query := `
		SELECT id, flashcard_id, user_id, next_review_date, interval, ease_factor, repetitions, created_at, updated_at
		FROM flashcard_reviews
		WHERE flashcard_id = $1 AND user_id = $2
	`
	var rev domain.FlashcardReview
	err := r.db.QueryRowContext(ctx, query, cardId, userId).Scan(&rev.Id, &rev.FlashcardId, &rev.UserId, &rev.NextReviewDate, &rev.Interval, &rev.EaseFactor, &rev.Repetitions, &rev.CreatedAt, &rev.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No review exists yet
		}
		return nil, fmt.Errorf("failed to query review: %w", err)
	}
	return &rev, nil
}

func (r *Repository) GetDueCardsForDeck(ctx context.Context, deckId uuid.UUID, userId uuid.UUID, currentTime time.Time) ([]domain.FlashcardWithReview, error) {
	// A card is due if it has no review yet OR its next_review_date <= currentTime
	query := `
		SELECT 
			f.id, f.deck_id, f.front, f.back, f.created_at, f.updated_at,
			r.id, r.flashcard_id, r.user_id, r.next_review_date, r.interval, r.ease_factor, r.repetitions, r.created_at, r.updated_at
		FROM flashcards f
		LEFT JOIN flashcard_reviews r ON f.id = r.flashcard_id AND r.user_id = $2
		WHERE f.deck_id = $1 AND (r.id IS NULL OR r.next_review_date <= $3)
		ORDER BY COALESCE(r.next_review_date, '1970-01-01'::timestamp) ASC, f.created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, deckId, userId, currentTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query due cards: %w", err)
	}
	defer rows.Close()

	var cards []domain.FlashcardWithReview
	for rows.Next() {
		var c domain.FlashcardWithReview
		var rId, rCardId, rUserId sql.NullString
		var rNextReview, rCreatedAt, rUpdatedAt sql.NullTime
		var rInterval, rEaseFactor sql.NullFloat64
		var rRepetitions sql.NullInt32

		err := rows.Scan(
			&c.Id, &c.DeckId, &c.Front, &c.Back, &c.CreatedAt, &c.UpdatedAt,
			&rId, &rCardId, &rUserId, &rNextReview, &rInterval, &rEaseFactor, &rRepetitions, &rCreatedAt, &rUpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan due card: %w", err)
		}

		if rId.Valid {
			parsedUserId, _ := uuid.Parse(rUserId.String)
			parsedId, _ := uuid.Parse(rId.String)
			parsedCardId, _ := uuid.Parse(rCardId.String)
			
			c.Review = &domain.FlashcardReview{
				Id:             parsedId,
				FlashcardId:    parsedCardId,
				UserId:         parsedUserId,
				NextReviewDate: rNextReview.Time,
				Interval:       float32(rInterval.Float64),
				EaseFactor:     float32(rEaseFactor.Float64),
				Repetitions:    int(rRepetitions.Int32),
				CreatedAt:      rCreatedAt.Time,
				UpdatedAt:      rUpdatedAt.Time,
			}
		}

		cards = append(cards, c)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over due cards: %w", err)
	}
	if cards == nil {
		cards = make([]domain.FlashcardWithReview, 0)
	}

	return cards, nil
}
