package memory

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"elsa-quiz-service/internal/domain"
	"golang.org/x/sync/singleflight"
)

// QuizLoader fetches quiz content from a backing store (e.g., document DB).
type QuizLoader interface {
	LoadQuiz(ctx context.Context, quizID string) (domain.Quiz, error)
}

// QuizRepository caches quizzes with TTL to avoid repeated DB hits.
type QuizRepository struct {
	loader QuizLoader
	ttl    time.Duration
	clock  func() time.Time
	sf     singleflight.Group
	rnd    *rand.Rand

	mu    sync.RWMutex
	cache map[string]cachedQuiz
}

type cachedQuiz struct {
	quiz      domain.Quiz
	expiresAt time.Time
}

func NewQuizRepository(loader QuizLoader, ttl time.Duration) *QuizRepository {
	return &QuizRepository{
		loader: loader,
		ttl:    ttl,
		clock:  time.Now,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
		cache:  make(map[string]cachedQuiz),
	}
}

func (r *QuizRepository) GetQuiz(ctx context.Context, quizID string) (domain.Quiz, error) {
	now := r.clock()

	r.mu.RLock()
	if entry, ok := r.cache[quizID]; ok && entry.expiresAt.After(now) {
		r.mu.RUnlock()
		return entry.quiz, nil
	}
	r.mu.RUnlock()

	result, err, _ := r.sf.Do(quizID, func() (interface{}, error) {
		now := r.clock()
		r.mu.RLock()
		if entry, ok := r.cache[quizID]; ok && entry.expiresAt.After(now) {
			r.mu.RUnlock()
			return entry.quiz, nil
		}
		r.mu.RUnlock()

		quiz, err := r.loader.LoadQuiz(ctx, quizID)
		if err != nil {
			return domain.Quiz{}, err
		}

		r.mu.Lock()
		r.cache[quizID] = cachedQuiz{
			quiz:      quiz,
			expiresAt: now.Add(r.ttlWithJitter()),
		}
		r.mu.Unlock()
		return quiz, nil
	})
	if err != nil {
		return domain.Quiz{}, err
	}
	return result.(domain.Quiz), nil
}

// StaticQuizLoader is a simple loader backed by an in-memory map (useful for tests/demos).
type StaticQuizLoader struct {
	quizzes map[string]domain.Quiz
}

func NewStaticQuizLoader(quizzes map[string]domain.Quiz) *StaticQuizLoader {
	return &StaticQuizLoader{quizzes: quizzes}
}

func (l *StaticQuizLoader) LoadQuiz(_ context.Context, quizID string) (domain.Quiz, error) {
	if quiz, ok := l.quizzes[quizID]; ok {
		return quiz, nil
	}
	return domain.Quiz{}, domain.ErrQuizNotFound
}

func (r *QuizRepository) ttlWithJitter() time.Duration {
	if r.ttl <= 0 {
		return 0
	}
	// add up to 10% jitter to spread expirations
	jitterMax := int64(r.ttl) / 10
	return r.ttl + time.Duration(r.rnd.Int63n(jitterMax+1))
}
