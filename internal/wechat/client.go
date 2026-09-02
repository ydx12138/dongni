package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	defaultSessionURL     = "https://api.weixin.qq.com/sns/jscode2session"
	defaultAccessTokenURL = "https://api.weixin.qq.com/cgi-bin/token"
	defaultPhoneNumberURL = "https://api.weixin.qq.com/wxa/business/getuserphonenumber"
)

var (
	ErrInvalidCode      = errors.New("wechat login code is required")
	ErrInvalidPhoneCode = errors.New("wechat phone code is required")
	ErrWechatLogin      = errors.New("wechat code exchange failed")
	ErrMissingOpenID    = errors.New("wechat response missing openid")
	ErrMissingPhone     = errors.New("wechat response missing phone number")
)

type Session struct {
	OpenID string
}

type Exchanger interface {
	ExchangeCode(context.Context, string) (Session, error)
	ExchangePhoneCode(context.Context, string) (string, error)
}

type Client struct {
	appID                string
	appSecret            string
	sessionURL           string
	accessTokenURL       string
	phoneNumberURL       string
	httpClient           *http.Client
	accessTokenMu        sync.Mutex
	accessToken          string
	accessTokenExpiresAt time.Time
}

func NewClient(appID, appSecret, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultSessionURL
	}
	return NewClientWithEndpoints(appID, appSecret, baseURL, defaultAccessTokenURL, defaultPhoneNumberURL)
}

func NewClientWithEndpoints(appID, appSecret, sessionURL, accessTokenURL, phoneNumberURL string) *Client {
	if sessionURL == "" {
		sessionURL = defaultSessionURL
	}
	if accessTokenURL == "" {
		accessTokenURL = defaultAccessTokenURL
	}
	if phoneNumberURL == "" {
		phoneNumberURL = defaultPhoneNumberURL
	}
	return &Client{
		appID:          appID,
		appSecret:      appSecret,
		sessionURL:     sessionURL,     //code换取open_id的请求地址
		accessTokenURL: accessTokenURL, //appID、appSecret换取access_token的请求地址
		phoneNumberURL: phoneNumberURL, //access_token、phone_code换取phone的请求地址
		httpClient:     http.DefaultClient,
	}
}

func (c *Client) ExchangeCode(ctx context.Context, code string) (Session, error) {
	if strings.TrimSpace(code) == "" {
		return Session{}, ErrInvalidCode
	}
	//准备请求路径
	endpoint, err := url.Parse(c.sessionURL)
	if err != nil {
		return Session{}, fmt.Errorf("parse wechat session endpoint: %w", err)
	}
	//设置请求参数
	query := endpoint.Query()
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")
	endpoint.RawQuery = query.Encode()
	//准备请求
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return Session{}, fmt.Errorf("build wechat request: %w", err)
	}
	//发起请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Session{}, fmt.Errorf("exchange wechat code: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Session{}, fmt.Errorf("%w: http status %d", ErrWechatLogin, resp.StatusCode)
	}
	//响应
	var result struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	//解析响应，把响应参数解析到result结构体里
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return Session{}, fmt.Errorf("decode wechat response: %w", err)
	}
	if result.ErrCode != 0 {
		return Session{}, fmt.Errorf("%w: %d %s", ErrWechatLogin, result.ErrCode, result.ErrMsg)
	}
	if result.OpenID == "" {
		return Session{}, ErrMissingOpenID
	}
	return Session{OpenID: result.OpenID}, nil
}

func (c *Client) ExchangePhoneCode(ctx context.Context, code string) (string, error) {
	//检查code
	if strings.TrimSpace(code) == "" {
		return "", ErrInvalidPhoneCode
	}
	//向微信服务器申请access_token，一会申请获取手机号需要这个token
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return "", err
	}
	//请求url
	endpoint, err := url.Parse(c.phoneNumberURL)
	if err != nil {
		return "", fmt.Errorf("parse wechat phone endpoint: %w", err)
	}
	//请求参数access_token
	query := endpoint.Query()
	query.Set("access_token", accessToken)
	endpoint.RawQuery = query.Encode()
	//请求体里的参数：code
	payload, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return "", fmt.Errorf("marshal wechat phone request: %w", err)
	}
	//组装请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build wechat phone request: %w", err)
	}
	//设置请求头
	req.Header.Set("Content-Type", "application/json")
	//发请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchange wechat phone code: %w", err)
	}
	defer resp.Body.Close()
	//检查响应
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%w: http status %d", ErrWechatLogin, resp.StatusCode)
	}
	//创建结构体，一会用来存放响应里的值
	var result struct {
		PhoneInfo struct {
			PurePhoneNumber string `json:"purePhoneNumber"`
		} `json:"phone_info"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	//解析响应到result结构体
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode wechat phone response: %w", err)
	}
	if result.ErrCode != 0 {
		return "", fmt.Errorf("%w: %d %s", ErrWechatLogin, result.ErrCode, result.ErrMsg)
	}
	if result.PhoneInfo.PurePhoneNumber == "" {
		return "", ErrMissingPhone
	}
	return result.PhoneInfo.PurePhoneNumber, nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.accessTokenMu.Lock()
	defer c.accessTokenMu.Unlock()
	if c.accessToken != "" && time.Now().Before(c.accessTokenExpiresAt) {
		return c.accessToken, nil
	}
	endpoint, err := url.Parse(c.accessTokenURL)
	if err != nil {
		return "", fmt.Errorf("parse wechat access token endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("appid", c.appID)
	query.Set("secret", c.appSecret)
	query.Set("grant_type", "client_credential")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("build wechat access token request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("get wechat access token: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%w: http status %d", ErrWechatLogin, resp.StatusCode)
	}
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode wechat access token response: %w", err)
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("%w: %d %s", ErrWechatLogin, result.ErrCode, result.ErrMsg)
	}
	expiresIn := time.Duration(result.ExpiresIn) * time.Second
	if expiresIn > time.Minute {
		expiresIn -= time.Minute
	}
	c.accessToken = result.AccessToken
	c.accessTokenExpiresAt = time.Now().Add(expiresIn)
	return c.accessToken, nil
}
