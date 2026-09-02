package service

import (
	"blog/internal/repository"
	"blog/internal/utils"
	"errors"
	"testing"
	"time"
)

type sessionRepositoryStub struct {
	value string
}

type userStatusRepositoryStub struct {
	status uint64
	err    error
}

// GetUserStatus 返回测试用户状态；参数为用户 ID；返回预设状态和错误。
func (s *userStatusRepositoryStub) GetUserStatus(uint64) (uint64, error) {
	return s.status, s.err
}

func (s *sessionRepositoryStub) SaveUserSession(uint64, string, time.Duration) error {
	return nil
}

func (s *sessionRepositoryStub) GetUserSession(uint64) (string, error) {
	return s.value, nil
}

var _ repository.RedisMiddleRepository = (*sessionRepositoryStub)(nil)

func TestGenerateUserTokenWithSessionKeepsSessionID(t *testing.T) {
	token, err := utils.GenerateUserTokenWithSession(7, time.Minute, "access", "session-a")
	if err != nil {
		t.Fatalf("GenerateUserTokenWithSession returned error: %v", err)
	}
	claims, err := utils.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken returned error: %v", err)
	}
	if claims.UserID != 7 || claims.Type != "access" || claims.SessionID != "session-a" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
}

func TestValidateSessionRejectsReplacedSession(t *testing.T) {
	svc := New(nil, nil, nil, &sessionRepositoryStub{value: "session-new"})
	if !errors.Is(svc.ValidateSession(7, "session-old"), ErrSessionInvalid) {
		t.Fatal("expected replaced session to be rejected")
	}
}

func TestValidateSessionAcceptsCurrentSession(t *testing.T) {
	svc := New(nil, nil, nil, &sessionRepositoryStub{value: "session-current"})
	if err := svc.ValidateSession(7, "session-current"); err != nil {
		t.Fatalf("expected current session to pass, got %v", err)
	}
}

func TestIsUserActiveRejectsBannedUser(t *testing.T) {
	svc := New(nil, nil, nil)
	svc.SetUserStatusRepository(&userStatusRepositoryStub{status: 2})

	active, err := svc.IsUserActive(7)

	if err != nil {
		t.Fatalf("IsUserActive returned error: %v", err)
	}
	if active {
		t.Fatal("expected banned user to be inactive")
	}
}
