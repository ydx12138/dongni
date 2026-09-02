package wechat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientExchangeCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("appid") != "app-id" {
			t.Fatal("missing appid")
		}
		if r.URL.Query().Get("secret") != "app-secret" {
			t.Fatal("missing secret")
		}
		if r.URL.Query().Get("js_code") != "login-code" {
			t.Fatal("missing code")
		}
		if r.URL.Query().Get("grant_type") != "authorization_code" {
			t.Fatal("missing grant type")
		}
		_, _ = w.Write([]byte(`{"openid":"openid-1","session_key":"secret"}`))
	}))
	defer server.Close()

	got, err := NewClient("app-id", "app-secret", server.URL).ExchangeCode(context.Background(), "login-code")
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if got.OpenID != "openid-1" {
		t.Fatalf("expected openid-1, got %q", got.OpenID)
	}
}

func TestClientExchangeCodeRejectsWechatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
	}))
	defer server.Close()

	_, err := NewClient("app-id", "app-secret", server.URL).ExchangeCode(context.Background(), "login-code")
	if err == nil {
		t.Fatal("expected WeChat error")
	}
}

func TestClientExchangePhoneCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/token":
			if r.URL.Query().Get("appid") != "app-id" || r.URL.Query().Get("secret") != "app-secret" {
				t.Fatal("missing app credentials")
			}
			_, _ = w.Write([]byte(`{"access_token":"access-1","expires_in":7200}`))
		case "/wxa/business/getuserphonenumber":
			if r.URL.Query().Get("access_token") != "access-1" {
				t.Fatal("missing access token")
			}
			_, _ = w.Write([]byte(`{"phone_info":{"purePhoneNumber":"13800138000"}}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClientWithEndpoints(
		"app-id", "app-secret", server.URL+"/sns/jscode2session",
		server.URL+"/cgi-bin/token", server.URL+"/wxa/business/getuserphonenumber",
	)
	phone, err := client.ExchangePhoneCode(context.Background(), "phone-code")
	if err != nil {
		t.Fatalf("ExchangePhoneCode returned error: %v", err)
	}
	if phone != "13800138000" {
		t.Fatalf("expected phone number, got %q", phone)
	}
}

func TestClientExchangePhoneCodeRejectsWechatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/token" {
			_, _ = w.Write([]byte(`{"access_token":"access-1","expires_in":7200}`))
			return
		}
		_, _ = w.Write([]byte(`{"errcode":40029,"errmsg":"invalid code"}`))
	}))
	defer server.Close()

	client := NewClientWithEndpoints(
		"app-id", "app-secret", server.URL+"/sns/jscode2session",
		server.URL+"/cgi-bin/token", server.URL+"/wxa/business/getuserphonenumber",
	)
	if _, err := client.ExchangePhoneCode(context.Background(), "phone-code"); err == nil {
		t.Fatal("expected WeChat error")
	}
}

func TestClientExchangePhoneCodeRejectsEmptyPhone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cgi-bin/token" {
			_, _ = w.Write([]byte(`{"access_token":"access-1","expires_in":7200}`))
			return
		}
		_, _ = w.Write([]byte(`{"phone_info":{}}`))
	}))
	defer server.Close()

	client := NewClientWithEndpoints(
		"app-id", "app-secret", server.URL+"/sns/jscode2session",
		server.URL+"/cgi-bin/token", server.URL+"/wxa/business/getuserphonenumber",
	)
	if _, err := client.ExchangePhoneCode(context.Background(), "phone-code"); err == nil {
		t.Fatal("expected empty phone error")
	}
}
