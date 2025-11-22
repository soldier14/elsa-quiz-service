package domain

import "errors"

var (
	// ErrSessionNotFound is returned when a quiz session has not been initialized.
	ErrSessionNotFound = errors.New("quiz session not found")
	// ErrParticipantNotFound is returned when a user tries to act before joining.
	ErrParticipantNotFound = errors.New("participant not found in quiz")
	// ErrQuizNotFound indicates the quiz content could not be loaded.
	ErrQuizNotFound = errors.New("quiz not found")
	// ErrQuestionNotFound indicates a submitted question ID is invalid.
	ErrQuestionNotFound = errors.New("question not found")
	// ErrOptionNotFound indicates a submitted option ID is invalid.
	ErrOptionNotFound = errors.New("option not found")
)
