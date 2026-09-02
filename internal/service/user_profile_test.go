package service

import (
	"blog/models"
	"errors"
	"testing"
	"time"
)

func TestUserProfileFromUser(t *testing.T) {
	phone := "13800000000"
	user := models.User{
		ID:           7,
		Email:        "a@example.com",
		Avatar:       "https://example.com/avatar.jpg",
		Nickname:     "阿明",
		Phone:        &phone,
		Password:     "secret",
		WechatOpenID: "openid",
		CreatedAt:    time.Date(2026, 8, 8, 3, 4, 5, 0, time.UTC),
	}

	profile := UserProfileFromUser(user)
	if profile.ID != 7 || profile.Email != "a@example.com" || profile.Nickname != "阿明" {
		t.Fatalf("unexpected profile identity: %#v", profile)
	}
	if profile.Phone == nil || *profile.Phone != phone {
		t.Fatalf("unexpected profile phone: %#v", profile.Phone)
	}
	if profile.Avatar != "https://example.com/avatar.jpg" {
		t.Fatalf("unexpected profile avatar: %q", profile.Avatar)
	}
	if !profile.CreatedAt.Equal(user.CreatedAt) {
		t.Fatalf("unexpected profile creation time: %v", profile.CreatedAt)
	}
}

// TestUpdateCurrentUserAvatarRejectsInvalidURL 确保非法头像地址不会进入数据库更新；参数为测试服务；无返回值。
func TestUpdateCurrentUserAvatarRejectsInvalidURL(t *testing.T) {
	service := New(nil, nil, nil)

	err := service.UpdateCurrentUserAvatar(7, "javascript:alert(1)")

	if !errors.Is(err, ErrInvalidUserAvatar) {
		t.Fatalf("UpdateCurrentUserAvatar() error = %v, want ErrInvalidUserAvatar", err)
	}
}

func TestUserProfileFromUserKeepsNilPhone(t *testing.T) {
	profile := UserProfileFromUser(models.User{})
	if profile.Phone != nil {
		t.Fatal("expected nil phone")
	}
}
