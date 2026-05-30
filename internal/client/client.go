package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type APIError struct {
	Status  int
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d", e.Status)
}

type TokenStore interface {
	Token() string
	RefreshToken() string
	SetTokens(access, refresh string) error
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	store      TokenStore
	token      string
}

func New(baseURL string, store TokenStore, tokenOverride string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: http.DefaultClient,
		store:      store,
		token:      tokenOverride,
	}
}

func (c *Client) tokenForRequest() string {
	if c.token != "" {
		return c.token
	}
	if c.store != nil {
		return c.store.Token()
	}
	return ""
}

func (c *Client) Do(ctx context.Context, method, path string, body any, out any) error {
	return c.do(ctx, method, path, body, out, false)
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any, retried bool) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token := c.tokenForRequest(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	respBody, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusUnauthorized && !retried && c.canRefresh() {
		if err := c.refreshAccessToken(ctx); err != nil {
			return parseAPIError(res.StatusCode, respBody)
		}
		return c.do(ctx, method, path, body, out, true)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return parseAPIError(res.StatusCode, respBody)
	}

	if out == nil || len(respBody) == 0 {
		return nil
	}
	return json.Unmarshal(respBody, out)
}

func (c *Client) canRefresh() bool {
	if c.store == nil {
		return false
	}
	if c.token != "" {
		return false
	}
	return c.store.RefreshToken() != ""
}

func (c *Client) refreshAccessToken(ctx context.Context) error {
	reqBody := map[string]string{"refresh_token": c.store.RefreshToken()}
	var resp refreshTokenResponse
	if err := c.do(ctx, http.MethodPost, "/v1/auth/refresh", reqBody, &resp, true); err != nil {
		return err
	}
	return c.store.SetTokens(resp.AccessToken(), "")
}

func parseAPIError(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	var parsed struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &parsed) == nil && parsed.Message != "" {
		msg = parsed.Message
	}
	return &APIError{Status: status, Message: msg, Body: string(body)}
}

type refreshTokenResponse struct {
	AccessTokenSnake string `json:"access_token"`
	AccessTokenCamel string `json:"accessToken"`
}

func (r refreshTokenResponse) AccessToken() string {
	if r.AccessTokenSnake != "" {
		return r.AccessTokenSnake
	}
	return r.AccessTokenCamel
}

func stringField(raw map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := raw[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
	}
	return ""
}

func intField(raw map[string]any, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := raw[k]; ok {
			switch n := v.(type) {
			case float64:
				return int64(n)
			case int64:
				return n
			case int:
				return int64(n)
			}
		}
	}
	return 0
}

func stringSliceField(raw map[string]any, keys ...string) []string {
	for _, k := range keys {
		v, ok := raw[k]
		if !ok || v == nil {
			continue
		}
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	return nil
}
