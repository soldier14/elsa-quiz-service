package redis

import (
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestSessionStoreSetsAndClearsKeys(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewSessionStore(client, time.Minute)

	_ = store.GetOrCreate("quiz-1")
	if !mr.Exists("quiz:session:quiz-1") {
		t.Fatalf("expected redis key to be set")
	}

	store.DeleteIfEmpty("quiz-1")
	if mr.Exists("quiz:session:quiz-1") {
		t.Fatalf("expected redis key to be removed")
	}
}
