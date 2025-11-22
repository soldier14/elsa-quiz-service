package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"elsa-quiz-service/internal/app"
	"elsa-quiz-service/internal/domain"
	"elsa-quiz-service/internal/infra/memory"
	"github.com/gorilla/websocket"
)

func TestWebSocketAnswerFlow(t *testing.T) {
	store := memory.NewSessionStore()
	quizRepo := memory.NewQuizRepository(memory.NewStaticQuizLoader(sampleQuiz()), time.Minute)
	service := app.NewQuizService(store, quizRepo)
	wsHandler := NewWSHandler(service)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler.ServeWS)
	server := httptest.NewServer(mux)
	defer server.Close()

	u := "ws" + server.URL[len("http"):] + "/ws?quizId=quiz-1&userId=u1&name=Alice"
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Expect joined event first.
	msgType, payload := readNext(conn, t, "joined")
	if msgType != "joined" {
		t.Fatalf("expected joined, got %s", msgType)
	}
	if payload == nil {
		t.Fatalf("expected joined payload, got nil")
	}

	// Send an answer.
	answer := map[string]any{
		"type": "answer",
		"payload": map[string]any{
			"questionId": "q1",
			"optionId":   "o2",
		},
	}
	if err := conn.WriteJSON(answer); err != nil {
		t.Fatalf("write answer: %v", err)
	}

	// Expect answerResult then leaderboard.
	answerSeen := false
	leaderboardSeen := false
	for i := 0; i < 3; i++ {
		typ, _ := readNext(conn, t, "")
		switch typ {
		case "answerResult":
			answerSeen = true
		case "leaderboard":
			leaderboardSeen = true
		}
		if answerSeen && leaderboardSeen {
			break
		}
	}
	if !answerSeen || !leaderboardSeen {
		t.Fatalf("expected answerResult and leaderboard, got answerResult=%v leaderboard=%v", answerSeen, leaderboardSeen)
	}
}

func readNext(conn *websocket.Conn, t *testing.T, expect string) (string, map[string]any) {
	t.Helper()
	var msg struct {
		Type    string         `json:"type"`
		Payload map[string]any `json:"payload"`
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("read json: %v", err)
	}
	if expect != "" && msg.Type != expect {
		t.Fatalf("expected type %s, got %s", expect, msg.Type)
	}
	return msg.Type, msg.Payload
}

func sampleQuiz() map[string]domain.Quiz {
	return map[string]domain.Quiz{
		"quiz-1": {
			ID: "quiz-1",
			Questions: []domain.Question{
				{
					ID:     "q1",
					Prompt: "What is 2 + 2?",
					Options: []domain.Option{
						{ID: "o1", Text: "3", Correct: false},
						{ID: "o2", Text: "4", Correct: true},
						{ID: "o3", Text: "5", Correct: false},
					},
					Points: 1,
				},
			},
		},
	}
}
