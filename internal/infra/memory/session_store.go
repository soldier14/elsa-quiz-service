package memory

import (
	"sync"

	"elsa-quiz-service/internal/app"
)

// SessionStore is an in-memory implementation of app.SessionRepository.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*app.Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
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
	}
}
