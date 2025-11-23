package app

import (
	"context"
	"sort"
	"sync"
	"time"

	"elsa-quiz-service/internal/domain"
)

// SessionRepository abstracts how quiz sessions are stored (in-memory, Redis, etc).
type SessionRepository interface {
	GetOrCreate(quizID string) *Session
	Get(quizID string) (*Session, bool)
	DeleteIfEmpty(quizID string)
}

// QuizRepository loads quiz content (from cache/backing store).
type QuizRepository interface {
	GetQuiz(ctx context.Context, quizID string) (domain.Quiz, error)
}

// QuizService contains the core quiz use cases.
type QuizService struct {
	sessions SessionRepository
	quizzes  QuizRepository
}

func NewQuizService(store SessionRepository, quizzes QuizRepository) *QuizService {
	return &QuizService{sessions: store, quizzes: quizzes}
}

// NewSession is exported for infrastructure layers that need to seed sessions.
func NewSession(id string) *Session {
	return newSession(id)
}

// NewSessionWithClock is test-only for deterministic timestamps.
func NewSessionWithClock(id string, now func() time.Time) *Session {
	return newSessionWithClock(id, now)
}

// Join registers or refreshes a participant in a quiz session.
func (s *QuizService) Join(ctx context.Context, quizID, userID, displayName string) (domain.Leaderboard, error) {
	// Preload quiz into cache; users cannot join unknown quizzes.
	if _, err := s.quizzes.GetQuiz(ctx, quizID); err != nil {
		return domain.Leaderboard{}, err
	}

	session := s.sessions.GetOrCreate(quizID)
	return session.join(userID, displayName), nil
}

// SubmitAnswer records an answer for a participant and updates the leaderboard.
func (s *QuizService) SubmitAnswer(ctx context.Context, quizID, userID string, submission domain.AnswerSubmission) (domain.Leaderboard, int, int, bool, error) {
	session, ok := s.sessions.Get(quizID)
	if !ok {
		return domain.Leaderboard{}, 0, 0, false, domain.ErrSessionNotFound
	}

	quiz, err := s.quizzes.GetQuiz(ctx, quizID)
	if err != nil {
		return domain.Leaderboard{}, 0, 0, false, err
	}

	correct, points, err := scoreSubmission(quiz, submission)
	if err != nil {
		return domain.Leaderboard{}, 0, 0, false, err
	}

	lb, total, err := session.applyScore(userID, correct, points)
	awarded := 0
	if correct {
		if points > 0 {
			awarded = points
		} else {
			awarded = 1
		}
	}
	return lb, total, awarded, correct, err
}

// Subscribe returns a channel that receives leaderboard updates for a quiz.
// The caller must invoke the returned cancel function to avoid leaks.
func (s *QuizService) Subscribe(_ context.Context, quizID string) (<-chan domain.Leaderboard, func(), error) {
	session, ok := s.sessions.Get(quizID)
	if !ok {
		return nil, nil, domain.ErrSessionNotFound
	}
	ch, cancel := session.subscribe()
	return ch, cancel, nil
}

// Leave removes a participant from the session and drops the session if empty.
func (s *QuizService) Leave(_ context.Context, quizID, userID string) {
	session, ok := s.sessions.Get(quizID)
	if !ok {
		return
	}
	session.leave(userID)
	if session.isEmpty() {
		s.sessions.DeleteIfEmpty(quizID)
	}
}

// Session is an in-memory representation of a quiz.
type Session struct {
	id           string
	createdAt    time.Time
	now          func() time.Time
	mu           sync.RWMutex
	participants map[string]*domain.Participant
	subscribers  map[chan domain.Leaderboard]struct{}
}

func newSession(id string) *Session {
	return newSessionWithClock(id, time.Now)
}

// newSessionWithClock allows deterministic timestamps in tests.
func newSessionWithClock(id string, now func() time.Time) *Session {
	return &Session{
		id:           id,
		createdAt:    now(),
		now:          now,
		participants: make(map[string]*domain.Participant),
		subscribers:  make(map[chan domain.Leaderboard]struct{}),
	}
}

func (s *Session) join(userID, displayName string) domain.Leaderboard {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if participant, ok := s.participants[userID]; ok {
		participant.DisplayName = displayName
		participant.LastUpdated = now
	} else {
		s.participants[userID] = &domain.Participant{
			UserID:      userID,
			DisplayName: displayName,
			Score:       0,
			LastUpdated: now,
		}
	}
	return s.broadcastLocked()
}

func (s *Session) applyScore(userID string, correct bool, points int) (domain.Leaderboard, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	participant, ok := s.participants[userID]
	if !ok {
		return domain.Leaderboard{}, 0, domain.ErrParticipantNotFound
	}

	if correct && points > 0 {
		participant.Score += points
	} else if correct && points == 0 {
		participant.Score++
	}
	participant.LastUpdated = now

	return s.broadcastLocked(), participant.Score, nil
}

func (s *Session) leave(userID string) domain.Leaderboard {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.participants, userID)
	return s.broadcastLocked()
}

func (s *Session) isEmpty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.participants) == 0
}

// IsEmpty reports whether the session has no participants.
func (s *Session) IsEmpty() bool {
	return s.isEmpty()
}

func (s *Session) subscribe() (<-chan domain.Leaderboard, func()) {
	ch := make(chan domain.Leaderboard, 8)

	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	initial := s.snapshotLocked()
	s.mu.Unlock()

	ch <- initial

	cancel := func() {
		s.mu.Lock()
		if _, ok := s.subscribers[ch]; ok {
			delete(s.subscribers, ch)
			close(ch)
		}
		s.mu.Unlock()
	}
	return ch, cancel
}

func (s *Session) broadcastLocked() domain.Leaderboard {
	lb := s.snapshotLocked()
	for ch := range s.subscribers {
		select {
		case ch <- lb:
		default:
			// AI-assisted: dropping stale updates prevents slow clients from blocking broadcast; verified via subscription tests.
			select {
			case <-ch:
			default:
			}
			ch <- lb
		}
	}
	return lb
}

func (s *Session) snapshotLocked() domain.Leaderboard {
	entries := make([]domain.LeaderboardEntry, 0, len(s.participants))
	for _, participant := range s.participants {
		entries = append(entries, domain.LeaderboardEntry{
			UserID:      participant.UserID,
			DisplayName: participant.DisplayName,
			Score:       participant.Score,
		})
	}

	// AI-assisted per your guidance: tie-breaker logic (score desc, then earliest completion, then name) drafted with ChatGPT to prioritize faster finishers.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		// Tie-break by who reached the score earlier (lower LastUpdated), then name.
		pi := s.participants[entries[i].UserID]
		pj := s.participants[entries[j].UserID]
		if pi != nil && pj != nil && !pi.LastUpdated.Equal(pj.LastUpdated) {
			return pi.LastUpdated.Before(pj.LastUpdated)
		}
		return entries[i].DisplayName < entries[j].DisplayName
	})

	return domain.Leaderboard{
		QuizID:    s.id,
		Entries:   entries,
		UpdatedAt: s.now(),
	}
}

// scoreSubmission validates the answer against quiz content and returns (correct, points).
func scoreSubmission(quiz domain.Quiz, submission domain.AnswerSubmission) (bool, int, error) {
	var question *domain.Question
	for i := range quiz.Questions {
		if quiz.Questions[i].ID == submission.QuestionID {
			question = &quiz.Questions[i]
			break
		}
	}
	if question == nil {
		return false, 0, domain.ErrQuestionNotFound
	}

	var selected *domain.Option
	for i := range question.Options {
		if question.Options[i].ID == submission.OptionID {
			selected = &question.Options[i]
			break
		}
	}
	if selected == nil {
		return false, 0, domain.ErrOptionNotFound
	}

	points := question.Points
	if points == 0 {
		points = 1
	}
	if selected.Correct {
		return true, points, nil
	}
	return false, 0, nil
}
