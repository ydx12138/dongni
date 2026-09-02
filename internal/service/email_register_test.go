package service

import (
	"errors"
	"testing"
)

func TestBuildEmailUserUsesConfiguredAvatar(t *testing.T) {
	user, err := BuildEmailUser("user@example.com", "hashed-password", "test-user", "https://example.com/default.jpg")
	if err != nil {
		t.Fatalf("BuildEmailUser returned error: %v", err)
	}
	if user.Avatar != "https://example.com/default.jpg" {
		t.Fatalf("unexpected avatar: %q", user.Avatar)
	}
	if user.Email != "user@example.com" || user.Password != "hashed-password" || user.Nickname != "test-user" {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestBuildEmailUserRejectsEmptyAvatar(t *testing.T) {
	_, err := BuildEmailUser("user@example.com", "hashed-password", "test-user", " ")
	if !errors.Is(err, ErrDefaultAvatar) {
		t.Fatalf("expected ErrDefaultAvatar, got %v", err)
	}
}
