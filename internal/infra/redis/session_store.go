package redis

import (
	"context"
	"sync"
	"time"

	"elsa-quiz-service/internal/app"
	"github.com/redis/go-redis/v9"
)

// SessionStore is a Redis-aware implementation of SessionRepository.
// Notes:
//   - It still keeps a local in-memory map of sessions to reuse the existing
//     in-process broadcast logic.
//   - Redis is used to mark session liveness (and could be extended to share
//     snapshots or route cross-instance pub/sub).
//   - For true distribution you'd pair this with a pub/sub projector that fans out updates.
type SessionStore struct {
	client   *redis.Client
	ttl      time.Duration
	mu       sync.RWMutex
	sessions map[string]*app.Session
}

func NewSessionStore(client *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{
		client:   client,
		ttl:      ttl,
		sessions: make(map[string]*app.Session),
	}
}

func (s *SessionStore) GetOrCreate(quizID string) *app.Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.sessions[quizID]; ok {
		return session
	}
	session := app.NewSession(quizID)
	s.sessions[quizID] = session
	// best-effort liveness marker
	_ = s.client.Set(context.Background(), s.key(quizID), "1", s.ttl).Err()
	return session
}

func (s *SessionStore) Get(quizID string) (*app.Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[quizID]
	return session, ok
}

func (s *SessionStore) DeleteIfEmpty(quizID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[quizID]
	if !ok {
		return
	}
	if session.IsEmpty() {
		delete(s.sessions, quizID)
		_ = s.client.Del(context.Background(), s.key(quizID)).Err()
	}
}

func (s *SessionStore) key(quizID string) string {
	return "quiz:session:" + quizID
}
