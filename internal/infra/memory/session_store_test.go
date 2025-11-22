package memory

import "testing"

func TestSessionStoreLifecycle(t *testing.T) {
	store := NewSessionStore()

	session := store.GetOrCreate("quiz-1")
	if session == nil {
		t.Fatalf("expected session")
	}
	if _, ok := store.Get("quiz-1"); !ok {
		t.Fatalf("expected session present")
	}

	store.DeleteIfEmpty("quiz-1")
	if _, ok := store.Get("quiz-1"); ok {
		t.Fatalf("expected session removed when empty")
	}
}
