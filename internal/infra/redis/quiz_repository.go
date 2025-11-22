package redis

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	"elsa-quiz-service/internal/domain"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

// QuizLoader fetches quiz content from a backing store (e.g., document DB).
type QuizLoader interface {
	LoadQuiz(ctx context.Context, quizID string) (domain.Quiz, error)
}

// QuizRepository caches quiz answers in Redis (hash per quiz) and falls back to a loader on cache miss.
// Answers are stored as: HSET quiz:{quizID}:answers {questionID} {optionID}
// Points are stored as:  HSET quiz:{quizID}:points  {questionID} {points}
type QuizRepository struct {
	client *redis.Client
	loader QuizLoader
	ttl    time.Duration
	sf     singleflight.Group
	rnd    *rand.Rand
}

func NewQuizRepository(client *redis.Client, loader QuizLoader, ttl time.Duration) *QuizRepository {
	return &QuizRepository{
		client: client,
		loader: loader,
		ttl:    ttl,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *QuizRepository) GetQuiz(ctx context.Context, quizID string) (domain.Quiz, error) {
	answerKey := r.answersKey(quizID)
	pointKey := r.pointsKey(quizID)

	answers, err := r.client.HGetAll(ctx, answerKey).Result()
	if err == nil && len(answers) > 0 {
		pointsMap, _ := r.client.HGetAll(ctx, pointKey).Result()
		return buildQuizFromCache(quizID, answers, pointsMap), nil
	}

	result, err, _ := r.sf.Do(quizID, func() (interface{}, error) {
		// Re-check cache in case another goroutine filled it.
		answers, err := r.client.HGetAll(ctx, answerKey).Result()
		if err == nil && len(answers) > 0 {
			pointsMap, _ := r.client.HGetAll(ctx, pointKey).Result()
			return buildQuizFromCache(quizID, answers, pointsMap), nil
		}

		quiz, err := r.loader.LoadQuiz(ctx, quizID)
		if err != nil {
			return domain.Quiz{}, err
		}

		ttl := r.ttlWithJitter()
		pipe := r.client.Pipeline()
		for _, q := range quiz.Questions {
			points := q.Points
			if points == 0 {
				points = 1
			}
			pipe.HSet(ctx, answerKey, q.ID, firstCorrectOption(q))
			pipe.HSet(ctx, pointKey, q.ID, points)
		}
		if ttl > 0 {
			pipe.Expire(ctx, answerKey, ttl)
			pipe.Expire(ctx, pointKey, ttl)
		}
		_, _ = pipe.Exec(ctx)

		return quiz, nil
	})
	if err != nil {
		return domain.Quiz{}, err
	}
	return result.(domain.Quiz), nil
}

func (r *QuizRepository) answersKey(quizID string) string {
	return "quiz:" + quizID + ":answers"
}

func (r *QuizRepository) pointsKey(quizID string) string {
	return "quiz:" + quizID + ":points"
}

func buildQuizFromCache(quizID string, answers map[string]string, pointsMap map[string]string) domain.Quiz {
	questions := make([]domain.Question, 0, len(answers))
	for questionID, optionID := range answers {
		points := 1
		if pStr, ok := pointsMap[questionID]; ok {
			if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
				points = p
			}
		}
		questions = append(questions, domain.Question{
			ID:     questionID,
			Prompt: "", // prompt not cached in this lightweight form
			Options: []domain.Option{
				{ID: optionID, Correct: true},
			},
			Points: points,
		})
	}
	return domain.Quiz{ID: quizID, Questions: questions}
}

func firstCorrectOption(q domain.Question) string {
	for _, opt := range q.Options {
		if opt.Correct {
			return opt.ID
		}
	}
	// Fallback to first option ID if no correct flag is set.
	if len(q.Options) > 0 {
		return q.Options[0].ID
	}
	return ""
}

func (r *QuizRepository) ttlWithJitter() time.Duration {
	if r.ttl <= 0 {
		return 0
	}
	jitterMax := int64(r.ttl) / 10
	return r.ttl + time.Duration(r.rnd.Int63n(jitterMax+1))
}
