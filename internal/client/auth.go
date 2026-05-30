package client

import (
	"context"
	"net/http"
)

type User struct {
	ID        string
	Email     string
	Name      string
	CreatedAt int64
}

type LoginResult struct {
	User         User
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

func (c *Client) Register(ctx context.Context, email, password, name string) (*LoginResult, error) {
	return c.authExchange(ctx, "/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
		"name":     name,
	})
}

func (c *Client) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	return c.authExchange(ctx, "/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
}

func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (string, int64, error) {
	var resp map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, &resp, false); err != nil {
		return "", 0, err
	}
	return stringField(resp, "access_token", "accessToken"), intField(resp, "expires_in", "expiresIn"), nil
}

func (c *Client) GetMe(ctx context.Context) (*User, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, "/v1/auth/me", nil, &resp); err != nil {
		return nil, err
	}
	userRaw, _ := resp["user"].(map[string]any)
	return mapUser(userRaw), nil
}

func (c *Client) UpdateProfile(ctx context.Context, name string) (*User, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPatch, "/v1/auth/me", map[string]string{"name": name}, &resp); err != nil {
		return nil, err
	}
	userRaw, _ := resp["user"].(map[string]any)
	return mapUser(userRaw), nil
}

func (c *Client) authExchange(ctx context.Context, path string, body map[string]string) (*LoginResult, error) {
	var resp map[string]any
	if err := c.do(ctx, http.MethodPost, path, body, &resp, false); err != nil {
		return nil, err
	}
	userRaw, _ := resp["user"].(map[string]any)
	return &LoginResult{
		User:         *mapUser(userRaw),
		AccessToken:  stringField(resp, "access_token", "accessToken"),
		RefreshToken: stringField(resp, "refresh_token", "refreshToken"),
		ExpiresIn:    intField(resp, "expires_in", "expiresIn"),
	}, nil
}

func mapUser(raw map[string]any) *User {
	if raw == nil {
		return &User{}
	}
	return &User{
		ID:        stringField(raw, "id"),
		Email:     stringField(raw, "email"),
		Name:      stringField(raw, "name"),
		CreatedAt: intField(raw, "created_at", "createdAt"),
	}
}
