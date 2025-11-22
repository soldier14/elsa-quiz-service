package domain

import "time"

// Participant represents a quiz participant and their accumulated score.
type Participant struct {
	UserID      string
	DisplayName string
	Score       int
	LastUpdated time.Time
}

// LeaderboardEntry is a snapshot-friendly view of a participant.
type LeaderboardEntry struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	Score       int    `json:"score"`
}

// Leaderboard captures the ordered scoreboard for a quiz session.
type Leaderboard struct {
	QuizID    string             `json:"quizId"`
	Entries   []LeaderboardEntry `json:"entries"`
	UpdatedAt time.Time          `json:"updatedAt"`
}

// AnswerSubmission models the scoring signal from clients.
type AnswerSubmission struct {
	QuestionID string
	OptionID   string
}

// AnswerResult summarizes the outcome of a submission for a single user.
type AnswerResult struct {
	QuestionID string `json:"questionId"`
	Correct    bool   `json:"correct"`
	Awarded    int    `json:"awarded"`
	TotalScore int    `json:"totalScore"`
}

// Option represents a possible answer for a question.
type Option struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	Correct bool   `json:"correct"`
}

// Question models an MCQ question with exactly one correct option.
type Question struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Options []Option `json:"options"`
	Points  int      `json:"points"` // defaults to 1 if zero
}

// Quiz is a collection of questions.
type Quiz struct {
	ID        string     `json:"id"`
	Questions []Question `json:"questions"`
}
