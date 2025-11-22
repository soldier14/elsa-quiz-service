package redis

import (
	"context"
	"testing"
	"time"

	"elsa-quiz-service/internal/domain"
	"elsa-quiz-service/internal/infra/memory"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestQuizRepositoryCachesInRedis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	defer mr.Close()

	client := newClient(mr)

	loader := &countingLoader{
		QuizLoader: memory.NewStaticQuizLoader(map[string]domain.Quiz{
			"quiz-1": sampleQuiz(),
		}),
	}
	repo := NewQuizRepository(client, loader, time.Minute)

	_, err = repo.GetQuiz(context.Background(), "quiz-1")
	if err != nil {
		t.Fatalf("get quiz: %v", err)
	}
	if loader.calls != 1 {
		t.Fatalf("expected loader called once, got %d", loader.calls)
	}

	// Second call should hit cache, loader not incremented.
	_, _ = repo.GetQuiz(context.Background(), "quiz-1")
	if loader.calls != 1 {
		t.Fatalf("expected cache hit, loader calls=%d", loader.calls)
	}
}

type countingLoader struct {
	memory.QuizLoader
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

func newClient(mr *miniredis.Miniredis) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
}
