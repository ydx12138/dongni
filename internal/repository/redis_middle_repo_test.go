package repository

import (
	"testing"
	"time"
)

func TestPCUserSessionKey(t *testing.T) {
	if got := pcUserSessionKey(7); got != "pc:user:session:7" {
		t.Fatalf("unexpected session key: %q", got)
	}
}

func TestRedisMiddleRepoRejectsNilClient(t *testing.T) {
	repo := &redisMiddleRepo{}
	if err := repo.SaveUserSession(7, "session-a", time.Minute); err == nil {
		t.Fatal("expected SaveUserSession to reject nil redis client")
	}
	if _, err := repo.GetUserSession(7); err == nil {
		t.Fatal("expected GetUserSession to reject nil redis client")
	}
}
