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

func (r *Repository) SaveQuiz(ctx context.Context, quiz *domain.Quiz) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	quizQuery := `
        INSERT INTO quizzes (id, user_id, document_id, title, created_at)
        VALUES ($1, $2, $3, $4, NOW())
    `
	_, err = tx.ExecContext(ctx, quizQuery, quiz.Id, quiz.UserId, quiz.DocumentId, quiz.Title)
	if err != nil {
		return fmt.Errorf("failed to insert quiz: %w", err)
	}

	questionQuery := `
        INSERT INTO quiz_questions (id, quiz_id, question_no, question_text, explanation)
        VALUES ($1, $2, $3, $4, $5)
    `

	optionQuery := `
        INSERT INTO quiz_question_options (id, question_id, option_label, option_text, is_correct, display_order)
        VALUES ($1, $2, $3, $4, $5, $6)
    `

	for i, question := range quiz.Questions {
		questionId := uuid.New()
		_, err = tx.ExecContext(ctx, questionQuery,
			questionId,
			quiz.Id,
			i+1, // question_no starts from 1
			question.QuestionText,
			question.Explanation, // Yeni alan
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

func (r *Repository) GetQuizzesByUserID(ctx context.Context, userId uuid.UUID) (*domain.GetQuizzesResponse, error) {
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
	var quizzes domain.GetQuizzesResponse
	for rows.Next() {
		var q domain.QuizResponseDTO
		var createdAt sql.NullTime
		if err := rows.Scan(&q.Id, &q.Title, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan quiz row: %w", err)
		}
		q.CreatedAt = createdAt.Time
		quizzes = append(quizzes, q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over quiz rows: %w", err)
	}

	return &quizzes, nil

}

func (r *Repository) GetQuizById(ctx context.Context, quizId uuid.UUID) (*domain.GetQuizByIdResponse, error) {
	query := `
        SELECT 
            q.id,
            q.title,
            qq.question_no,
            qq.question_text,
            qq.explanation,
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

	var response *domain.GetQuizByIdResponse
	questionsMap := make(map[int]*domain.QuizQuestion)

	for rows.Next() {
		var (
			quizID       uuid.UUID
			title        string
			questionNo   int
			questionText string
			explanation  string // Yeni alan
			optionLabel  string
			optionText   string
			isCorrect    bool
		)

		err := rows.Scan(
			&quizID,
			&title,
			&questionNo,
			&questionText,
			&explanation, // Yeni scan
			&optionLabel,
			&optionText,
			&isCorrect,
		)
		if err != nil {
			return nil, err
		}

		// İlk satırda response'u oluştur
		if response == nil {
			response = &domain.GetQuizByIdResponse{
				Id:        quizID,
				Title:     title,
				Questions: []domain.QuizQuestion{},
			}
		}

		// Question map'te yoksa ekle
		if _, exists := questionsMap[questionNo]; !exists {
			questionsMap[questionNo] = &domain.QuizQuestion{
				QuestionText:       questionText,
				Explanation:        explanation, // Yeni alan
				Options:            []domain.Option{},
				CorrectOptionLabel: "",
			}
		}

		// Option'ı ekle
		questionsMap[questionNo].Options = append(
			questionsMap[questionNo].Options,
			domain.Option{
				Text:  optionText,
				Label: optionLabel,
			},
		)

		// Correct option label'ı bul (A, B, C, D, E)
		if isCorrect {
			questionsMap[questionNo].CorrectOptionLabel = optionLabel
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

func (r *Repository) GetQuizAnswersById(ctx context.Context, quizID uuid.UUID) (map[int]string, int, error) {
	query := `
		SELECT 
			qq.question_no,
			qo.option_label,
			COUNT(*) OVER() as total_questions
		FROM quiz_questions qq
		INNER JOIN quiz_question_options qo ON qq.id = qo.question_id
		WHERE qq.quiz_id = $1 AND qo.is_correct = true
		ORDER BY qq.question_no
	`

	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	answers := make(map[int]string)
	totalQuestions := 0

	for rows.Next() {
		var questionNo int
		var correctOption string
		var total int

		if err := rows.Scan(&questionNo, &correctOption, &total); err != nil {
			return nil, 0, err
		}

		answers[questionNo] = correctOption
		totalQuestions = total
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(answers) == 0 {
		return nil, 0, errors.New("quiz not found")
	}

	return answers, totalQuestions, nil
}

func (r *Repository) SaveQuizAttempt(ctx context.Context, attempt *domain.QuizAttempt) error {
	query := `
		INSERT INTO quiz_attempts (
			id, 
			quiz_id, 
			user_id, 
			score, 
			total_questions
		) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		attempt.ID,
		attempt.QuizID,
		attempt.UserID,
		attempt.Score,
		attempt.TotalQuestions,
	)

	return err
}

// ============ REPOSITORY METHODS ============

// GetQuizDataForSolving quiz'i çözmek için gerekli tüm bilgiyi getirir
func (r *Repository) GetQuizDataForSolving(ctx context.Context, quizID uuid.UUID) (*domain.QuizSolvingData, error) {
	query := `
        SELECT 
            qq.question_no,
            qo.option_label,
            qq.explanation,
            COUNT(*) OVER() as total_questions
        FROM quiz_questions qq
        INNER JOIN quiz_question_options qo ON qq.id = qo.question_id
        WHERE qq.quiz_id = $1 AND qo.is_correct = true
        ORDER BY qq.question_no
    `

	rows, err := r.db.QueryContext(ctx, query, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := &domain.QuizSolvingData{
		Questions: make(map[int]domain.QuestionData),
	}

	for rows.Next() {
		var questionNo int
		var correctOption string
		var explanation string
		var total int

		if err := rows.Scan(&questionNo, &correctOption, &explanation, &total); err != nil {
			return nil, err
		}

		data.Questions[questionNo] = domain.QuestionData{
			CorrectOption: correctOption,
			Explanation:   explanation,
		}
		data.TotalQuestions = total
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(data.Questions) == 0 {
		return nil, errors.New("quiz not found")
	}

	return data, nil
}

// SaveQuizAttemptWithAnswers attempt'i ve kullanıcı cevaplarını transaction içinde kaydeder
func (r *Repository) SaveQuizAttemptWithAnswers(ctx context.Context, attempt *domain.QuizAttempt, answers []domain.QuizAttemptAnswer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Attempt'i kaydet
	attemptQuery := `
        INSERT INTO quiz_attempts (
            id, 
            quiz_id, 
            user_id, 
            score, 
            total_questions,
            completed_at
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err = tx.ExecContext(
		ctx,
		attemptQuery,
		attempt.ID,
		attempt.QuizID,
		attempt.UserID,
		attempt.Score,
		attempt.TotalQuestions,
		attempt.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save attempt: %w", err)
	}

	// Cevapları kaydet
	answerQuery := `
        INSERT INTO quiz_attempt_answers (
            id,
            attempt_id,
            question_no,
            selected_option,
            correct_option,
            is_correct
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `

	for _, answer := range answers {
		answer.AttemptID = attempt.ID // Set attempt ID
		_, err = tx.ExecContext(
			ctx,
			answerQuery,
			answer.ID,
			answer.AttemptID,
			answer.QuestionNo,
			answer.SelectedOption,
			answer.CorrectOption,
			answer.IsCorrect,
		)
		if err != nil {
			return fmt.Errorf("failed to save answer for question %d: %w", answer.QuestionNo, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetQuizAttemptsByUserId returns all quiz attempts for a user with pagination
func (r *Repository) GetQuizAttemptsByUserId(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.QuizAttemptDetail, int, error) {
	query := `
        SELECT 
            qa.id,
            qa.quiz_id,
            q.title as quiz_title,
            qa.score,
            qa.total_questions,
            qa.completed_at,
            COUNT(*) OVER() as total_count
        FROM quiz_attempts qa
        INNER JOIN quizzes q ON qa.quiz_id = q.id
        WHERE qa.user_id = $1
        ORDER BY qa.completed_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var attempts []domain.QuizAttemptDetail
	totalCount := 0

	for rows.Next() {
		var attempt domain.QuizAttemptDetail
		var total int

		err := rows.Scan(
			&attempt.ID,
			&attempt.QuizID,
			&attempt.QuizTitle,
			&attempt.Score,
			&attempt.TotalQuestions,
			&attempt.CompletedAt,
			&total,
		)
		if err != nil {
			return nil, 0, err
		}

		if attempt.TotalQuestions > 0 {
			attempt.Percentage = (float64(attempt.Score) / float64(attempt.TotalQuestions)) * 100
		}

		attempts = append(attempts, attempt)
		totalCount = total
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	if attempts == nil {
		attempts = []domain.QuizAttemptDetail{}
	}

	return attempts, totalCount, nil
}

// GetAttemptDetails bir attempt'in detaylı bilgilerini getirir (cevaplar + açıklamalar dahil)
func (r *Repository) GetAttemptDetails(ctx context.Context, attemptID uuid.UUID, userID uuid.UUID) (*domain.AttemptDetailResponse, error) {
	query := `
        SELECT 
            qa.id,
            qa.quiz_id,
            q.title as quiz_title,
            qa.score,
            qa.total_questions,
            qa.completed_at,
            qaa.question_no,
            qq.question_text,
            qaa.selected_option,
            qaa.correct_option,
            qaa.is_correct,
            qq.explanation
        FROM quiz_attempts qa
        INNER JOIN quizzes q ON qa.quiz_id = q.id
        INNER JOIN quiz_attempt_answers qaa ON qa.id = qaa.attempt_id
        INNER JOIN quiz_questions qq ON q.id = qq.quiz_id AND qq.question_no = qaa.question_no
        WHERE qa.id = $1 AND qa.user_id = $2
        ORDER BY qaa.question_no
    `

	rows, err := r.db.QueryContext(ctx, query, attemptID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var response *domain.AttemptDetailResponse
	var answers []domain.AttemptAnswerDetail

	for rows.Next() {
		var (
			attemptID      uuid.UUID
			quizID         uuid.UUID
			quizTitle      string
			score          int
			totalQuestions int
			completedAt    time.Time
			questionNo     int
			questionText   string
			selectedOption string
			correctOption  string
			isCorrect      bool
			explanation    string
		)

		err := rows.Scan(
			&attemptID,
			&quizID,
			&quizTitle,
			&score,
			&totalQuestions,
			&completedAt,
			&questionNo,
			&questionText,
			&selectedOption,
			&correctOption,
			&isCorrect,
			&explanation,
		)
		if err != nil {
			return nil, err
		}

		if response == nil {
			response = &domain.AttemptDetailResponse{
				ID:             attemptID,
				QuizID:         quizID,
				QuizTitle:      quizTitle,
				Score:          score,
				TotalQuestions: totalQuestions,
				CompletedAt:    completedAt,
			}
			if totalQuestions > 0 {
				response.Percentage = (float64(score) / float64(totalQuestions)) * 100
			}
		}

		answers = append(answers, domain.AttemptAnswerDetail{
			QuestionNo:     questionNo,
			QuestionText:   questionText,
			SelectedOption: selectedOption,
			CorrectOption:  correctOption,
			IsCorrect:      isCorrect,
			Explanation:    explanation,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if response == nil {
		return nil, errors.New("attempt not found")
	}

	response.Answers = answers
	return response, nil
}

func (r *Repository) GetQuizAttemptById(ctx context.Context, attemptId uuid.UUID) (*domain.QuizAttemptDetail, error) {
	query := `
		SELECT 
			qa.id,
			qa.quiz_id,
			q.title as quiz_title,
			qa.score,
			qa.total_questions,
			ROUND((qa.score::numeric / qa.total_questions::numeric * 100), 2) as percentage,
			qa.completed_at 
		FROM quiz_attempts qa
		JOIN quizzes q ON qa.quiz_id = q.id
		WHERE qa.id = $1
	`

	var attempt domain.QuizAttemptDetail
	err := r.db.QueryRowContext(ctx, query, attemptId).Scan(
		&attempt.ID,
		&attempt.QuizID,
		&attempt.QuizTitle,
		&attempt.Score,
		&attempt.TotalQuestions,
		&attempt.Percentage,
		&attempt.CompletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("quiz attempt not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quiz attempt: %w", err)
	}

	return &attempt, nil
}

// GetAttemptAnswers - Belirli bir attempt'e ait tüm cevapları getirir
func (r *Repository) GetAttemptAnswers(ctx context.Context, attemptId uuid.UUID) ([]domain.AttemptAnswerDetail, error) {
	query := `
		SELECT 
			qaa.question_no,
			qq.question_text,
			qaa.selected_option,
			qaa.correct_option,
			qaa.is_correct,
			COALESCE(qq.explanation, '') as explanation
		FROM quiz_attempt_answers qaa
		JOIN quiz_attempts qa ON qaa.attempt_id = qa.id
		JOIN quiz_questions qq ON qq.quiz_id = qa.quiz_id AND qq.question_no = qaa.question_no
		WHERE qaa.attempt_id = $1
		ORDER BY qaa.question_no ASC
	`

	rows, err := r.db.QueryContext(ctx, query, attemptId)
	if err != nil {
		return nil, fmt.Errorf("failed to query attempt answers: %w", err)
	}
	defer rows.Close()

	var answers []domain.AttemptAnswerDetail
	for rows.Next() {
		var answer domain.AttemptAnswerDetail
		err := rows.Scan(
			&answer.QuestionNo,
			&answer.QuestionText,
			&answer.SelectedOption,
			&answer.CorrectOption,
			&answer.IsCorrect,
			&answer.Explanation,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attempt answer: %w", err)
		}
		answers = append(answers, answer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attempt answers: %w", err)
	}

	return answers, nil
}
