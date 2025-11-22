package http

import (
	"encoding/json"
	"log"
	"net/http"

	"elsa-quiz-service/internal/app"
	"elsa-quiz-service/internal/domain"
	"github.com/gorilla/websocket"
)

type WSHandler struct {
	service  *app.QuizService
	upgrader websocket.Upgrader
}

func NewWSHandler(service *app.QuizService) *WSHandler {
	return &WSHandler{
		service: service,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

type inboundMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type answerPayload struct {
	QuestionID string `json:"questionId"`
	OptionID   string `json:"optionId"`
}

type answerResult struct {
	QuestionID string `json:"questionId"`
	Correct    bool   `json:"correct"`
	Awarded    int    `json:"awarded"`
	TotalScore int    `json:"totalScore"`
}

type outboundMessage[T any] struct {
	Type    string `json:"type"`
	Payload T      `json:"payload"`
}

type errorPayload struct {
	Message string `json:"message"`
}

// ServeWS upgrades HTTP requests to websockets and wires them into the quiz use cases.
func (h *WSHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	quizID := r.URL.Query().Get("quizId")
	userID := r.URL.Query().Get("userId")
	displayName := r.URL.Query().Get("name")
	if quizID == "" || userID == "" || displayName == "" {
		http.Error(w, "missing quizId, userId, or name", http.StatusBadRequest)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	joined, err := h.service.Join(r.Context(), quizID, userID, displayName)
	if err != nil {
		_ = conn.WriteJSON(outboundMessage[errorPayload]{Type: "error", Payload: errorPayload{Message: err.Error()}})
		return
	}

	updates, cancel, err := h.service.Subscribe(r.Context(), quizID)
	if err != nil {
		_ = conn.WriteJSON(outboundMessage[errorPayload]{Type: "error", Payload: errorPayload{Message: err.Error()}})
		return
	}
	defer cancel()
	defer h.service.Leave(r.Context(), quizID, userID)

	send := make(chan outboundMessage[any], 16)
	closeSignals := make(chan struct{})
	writerDone := make(chan struct{})
	updatesDone := make(chan struct{})

	// AI-assisted: read/write wiring adapted from Gorilla patterns with ChatGPT; verified via reasoning and tests to prevent concurrent writes.
	go func() {
		defer close(writerDone)
		for msg := range send {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("ws write error: %v", err)
				return
			}
		}
	}()

	go func() {
		defer close(updatesDone)
		for {
			select {
			case update, ok := <-updates:
				if !ok {
					return
				}
				select {
				case send <- outboundMessage[any]{Type: "leaderboard", Payload: update}:
				case <-closeSignals:
					return
				}
			case <-closeSignals:
				return
			}
		}
	}()

	send <- outboundMessage[any]{Type: "joined", Payload: joined}

	for {
		var inbound inboundMessage
		if err := conn.ReadJSON(&inbound); err != nil {
			break
		}
		switch inbound.Type {
		case "answer":
			var payload answerPayload
			if err := json.Unmarshal(inbound.Payload, &payload); err != nil {
				send <- outboundMessage[any]{Type: "error", Payload: errorPayload{Message: "invalid answer payload"}}
				continue
			}
			lb, total, awarded, correct, err := h.service.SubmitAnswer(r.Context(), quizID, userID, domain.AnswerSubmission{
				QuestionID: payload.QuestionID,
				OptionID:   payload.OptionID,
			})
			if err != nil {
				send <- outboundMessage[any]{Type: "error", Payload: errorPayload{Message: err.Error()}}
				continue
			}
			send <- outboundMessage[any]{Type: "answerResult", Payload: answerResult{
				QuestionID: payload.QuestionID,
				Correct:    correct,
				Awarded:    awarded,
				TotalScore: total,
			}}
			send <- outboundMessage[any]{Type: "leaderboard", Payload: lb}
		default:
			send <- outboundMessage[any]{Type: "error", Payload: errorPayload{Message: "unsupported message type"}}
		}
	}

	close(closeSignals)
	<-updatesDone
	close(send)
	<-writerDone
}

func scoreAwarded(correct bool, _ domain.Leaderboard, _ string, total int) int {
	if !correct {
		return 0
	}
	// Awarded is derived by comparing totals; since we don't track previous total here,
	// we conservatively return 1 when correct. Clients can rely on TotalScore for truth.
	if total > 0 {
		return 1
	}
	return 0
}
