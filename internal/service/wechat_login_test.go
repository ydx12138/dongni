package service

import (
	"strings"
	"testing"
)

func TestBuildWechatUserCreatesInternalIdentity(t *testing.T) {
	user, err := BuildWechatUser("openid-1")
	if err != nil {
		t.Fatalf("BuildWechatUser returned error: %v", err)
	}
	if user.Email != "wechat-openid-1@wechat.local" {
		t.Fatalf("unexpected email: %q", user.Email)
	}
	if user.WechatOpenID != "openid-1" {
		t.Fatalf("unexpected openid: %q", user.WechatOpenID)
	}
	if user.Status != 1 {
		t.Fatalf("unexpected status: %d", user.Status)
	}
	if !strings.HasPrefix(user.Nickname, "微信用户") {
		t.Fatalf("unexpected nickname: %q", user.Nickname)
	}
	if !strings.HasPrefix(user.Password, "$2") {
		t.Fatal("expected a bcrypt password hash")
	}
}

func TestWechatLoginResponseRequiresPhoneWithoutToken(t *testing.T) {
	data := phoneRequiredResponse("ticket-1")
	if data["phone_required"] != true || data["phone_ticket"] != "ticket-1" {
		t.Fatalf("unexpected response: %#v", data)
	}
	if _, ok := data["access_token"]; ok {
		t.Fatal("phone-required response must not include an access token")
	}
}
