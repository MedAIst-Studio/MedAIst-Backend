package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MedAIst-Studio/MedAIist-Backend/app/quiz"
	"github.com/MedAIst-Studio/MedAIist-Backend/domain"
	"github.com/google/uuid"
)

func (r *Repository) SaveQuiz(ctx context.Context, quiz *domain.Quiz) error {
	fmt.Println("Saving quiz to database...")
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	quizId := uuid.New()
	quizQuery := `
		INSERT INTO quizzes (id, user_id, document_id, title, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`
	_, err = tx.ExecContext(ctx, quizQuery, quizId, quiz.UserId, quiz.DocumentId, quiz.Title)
	if err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}

	questionQuery := `
		INSERT INTO quiz_questions (id, quiz_id, question_no, question_text)
		VALUES ($1, $2, $3, $4)
	`
	optionQuery := `
		INSERT INTO quiz_question_options (id, question_id, option_label, option_text, is_correct, display_order)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	for i, question := range quiz.Questions {
		questionId := uuid.New()

		_, err = tx.ExecContext(ctx, questionQuery,
			questionId,
			quizId,
			i+1, // question_no starts from 1
			question.QuestionText,
		)
		if err != nil {
			return fmt.Errorf("failed to insert question %d: %w", i+1, err)
		}

		for _, option := range question.Options {
			optionId := uuid.New()

			_, err = tx.ExecContext(ctx, optionQuery,
				optionId,
				questionId,
				option.Label,
				option.Text,
				option.IsCorrect,
				option.No,
			)
			if err != nil {
				return fmt.Errorf("failed to insert option %s for question %d: %w", option.Label, i+1, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *Repository) GetQuizzesByUserID(ctx context.Context, userId uuid.UUID) (*quiz.GetQuizzesResponse, error) {
	query := `
		SELECT id, title, created_at
		FROM quizzes
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to query quizzes: %w", err)
	}
	defer rows.Close()
	var quizzes quiz.GetQuizzesResponse
	for rows.Next() {
		var q quiz.QuizSummary
		var createdAt sql.NullTime
		if err := rows.Scan(&q.QuizId, &q.Title, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan quiz row: %w", err)
		}
		q.CreatedAt = createdAt.Time.Format("2006-01-02 15:04:05")
		quizzes = append(quizzes, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over quiz rows: %w", err)
	}

	return &quizzes, nil

}

func (r *Repository) GetQuizById(ctx context.Context, quizId uuid.UUID) (*quiz.GetQuizByIdResponse, error) {
	query := `
        SELECT 
            q.id,
            q.title,
            qq.question_no,
            qq.question_text,
            qo.option_label,
            qo.option_text,
            qo.is_correct
        FROM quizzes q
        INNER JOIN quiz_questions qq ON q.id = qq.quiz_id
        INNER JOIN quiz_question_options qo ON qq.id = qo.question_id
        WHERE q.id = $1
        ORDER BY qq.question_no, qo.display_order
    `

	rows, err := r.db.QueryContext(ctx, query, quizId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response *quiz.GetQuizByIdResponse
	questionsMap := make(map[int]*quiz.Question)

	for rows.Next() {
		var (
			quizID       uuid.UUID
			title        string
			questionNo   int
			questionText string
			optionLabel  string
			optionText   string
			isCorrect    bool
		)

		err := rows.Scan(
			&quizID,
			&title,
			&questionNo,
			&questionText,
			&optionLabel,
			&optionText,
			&isCorrect,
		)
		if err != nil {
			return nil, err
		}

		// İlk satırda response'u oluştur
		if response == nil {
			response = &quiz.GetQuizByIdResponse{
				ID:        quizID,
				Title:     title,
				Questions: []quiz.Question{},
			}
		}

		// Question map'te yoksa ekle
		if _, exists := questionsMap[questionNo]; !exists {
			questionsMap[questionNo] = &quiz.Question{
				No:            questionNo,
				Text:          questionText,
				Options:       []quiz.Option{},
				CorrectOption: "",
			}
		}

		// Option'ı ekle
		questionsMap[questionNo].Options = append(
			questionsMap[questionNo].Options,
			quiz.Option{
				Text:  optionText,
				Label: optionLabel,
			},
		)

		// Correct option label'ı bul (A, B, C, D, E)
		if isCorrect {
			questionsMap[questionNo].CorrectOption = optionLabel
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	// Response bulunamadıysa quiz yok demektir
	if response == nil {
		return nil, errors.New("quiz not found")
	}

	// Questions map'i slice'a çevir ve sırala
	for i := 1; i <= len(questionsMap); i++ {
		if q, exists := questionsMap[i]; exists {
			response.Questions = append(response.Questions, *q)
		}
	}

	return response, nil
}
