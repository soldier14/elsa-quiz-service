package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"elsa-quiz-service/internal/domain"
	"github.com/jackc/pgx/v4/pgxpool"
)

// QuizLoader loads quiz JSONB from Postgres.
type QuizLoader struct {
	pool *pgxpool.Pool
}

func NewQuizLoader(pool *pgxpool.Pool) *QuizLoader {
	return &QuizLoader{pool: pool}
}

func (l *QuizLoader) LoadQuiz(ctx context.Context, quizID string) (domain.Quiz, error) {
	var raw []byte
	err := l.pool.QueryRow(ctx, `SELECT data FROM quizzes WHERE id=$1`, quizID).Scan(&raw)
	if err != nil {
		return domain.Quiz{}, fmt.Errorf("load quiz: %w", err)
	}
	var quiz domain.Quiz
	if err := json.Unmarshal(raw, &quiz); err != nil {
		return domain.Quiz{}, fmt.Errorf("unmarshal quiz: %w", err)
	}
	return quiz, nil
}
