package app_test

import (
	"context"
	"testing"
	"time"

	"elsa-quiz-service/internal/app"
	"elsa-quiz-service/internal/domain"
	"elsa-quiz-service/internal/infra/memory"
)

func TestJoinAndScoring(t *testing.T) {
	ctx := context.Background()
	service := newTestService()

	if _, err := service.Join(ctx, "quiz-1", "u1", "Alice"); err != nil {
		t.Fatalf("join failed: %v", err)
	}
	if _, err := service.Join(ctx, "quiz-1", "u2", "Bob"); err != nil {
		t.Fatalf("join failed: %v", err)
	}

	lb, _, _, _, err := service.SubmitAnswer(ctx, "quiz-1", "u2", domain.AnswerSubmission{
		QuestionID: "q1",
		OptionID:   "o2", // correct
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}
	if len(lb.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lb.Entries))
	}
	if lb.Entries[0].UserID != "u2" || lb.Entries[0].Score != 1 {
		t.Fatalf("expected Bob to lead with 1 point, got %+v", lb.Entries[0])
	}
}

func TestSubscribeReceivesUpdates(t *testing.T) {
	ctx := context.Background()
	service := newTestService()

	if _, err := service.Join(ctx, "quiz-1", "u1", "Alice"); err != nil {
		t.Fatalf("join failed: %v", err)
	}
	ch, cancel, err := service.Subscribe(ctx, "quiz-1")
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}
	defer cancel()

	<-ch // initial snapshot

	_, _, _, _, err = service.SubmitAnswer(ctx, "quiz-1", "u1", domain.AnswerSubmission{
		QuestionID: "q1",
		OptionID:   "o2",
	})
	if err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	update := <-ch
	if len(update.Entries) != 1 || update.Entries[0].Score != 1 {
		t.Fatalf("expected updated score 1, got %+v", update.Entries)
	}
}

func TestSubmitRequiresParticipant(t *testing.T) {
	ctx := context.Background()
	service := newTestService()

	_, _, _, _, err := service.SubmitAnswer(ctx, "quiz-unknown", "u1", domain.AnswerSubmission{QuestionID: "q1", OptionID: "o1"})
	if err != domain.ErrSessionNotFound {
		t.Fatalf("expected session error, got %v", err)
	}

	_, _ = service.Join(ctx, "quiz-1", "u1", "Alice")
	_, _, _, _, err = service.SubmitAnswer(ctx, "quiz-1", "u2", domain.AnswerSubmission{QuestionID: "q1", OptionID: "o2"})
	if err != domain.ErrParticipantNotFound {
		t.Fatalf("expected participant error, got %v", err)
	}
}

func newTestService() *app.QuizService {
	sessionStore := memory.NewSessionStore()
	quizRepo := memory.NewQuizRepository(memory.NewStaticQuizLoader(map[string]domain.Quiz{
		"quiz-1": {
			ID: "quiz-1",
			Questions: []domain.Question{
				{
					ID:     "q1",
					Prompt: "Select the right option",
					Options: []domain.Option{
						{ID: "o1", Text: "Wrong", Correct: false},
						{ID: "o2", Text: "Right", Correct: true},
					},
					Points: 1,
				},
			},
		},
	}), 5*time.Minute)
	return app.NewQuizService(sessionStore, quizRepo)
}
