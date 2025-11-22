package memory

import (
	"context"
	"testing"
	"time"

	"elsa-quiz-service/internal/domain"
)

func TestQuizRepositoryCaches(t *testing.T) {
	loader := &countingLoader{
		QuizLoader: NewStaticQuizLoader(map[string]domain.Quiz{
			"quiz-1": sampleQuiz(),
		}),
	}
	repo := NewQuizRepository(loader, time.Minute)

	if _, err := repo.GetQuiz(context.Background(), "quiz-1"); err != nil {
		t.Fatalf("get quiz: %v", err)
	}
	if loader.calls != 1 {
		t.Fatalf("expected loader once, got %d", loader.calls)
	}

	if _, err := repo.GetQuiz(context.Background(), "quiz-1"); err != nil {
		t.Fatalf("get quiz 2: %v", err)
	}
	if loader.calls != 1 {
		t.Fatalf("expected cache hit, loader calls %d", loader.calls)
	}
}

type countingLoader struct {
	QuizLoader
	calls int
}

func (l *countingLoader) LoadQuiz(ctx context.Context, quizID string) (domain.Quiz, error) {
	l.calls++
	return l.QuizLoader.LoadQuiz(ctx, quizID)
}

func sampleQuiz() domain.Quiz {
	return domain.Quiz{
		ID: "quiz-1",
		Questions: []domain.Question{
			{
				ID:     "q1",
				Prompt: "What is 2 + 2?",
				Options: []domain.Option{
					{ID: "o1", Text: "3", Correct: false},
					{ID: "o2", Text: "4", Correct: true},
				},
				Points: 1,
			},
		},
	}
}
