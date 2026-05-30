package client

import (
	"context"
	"fmt"
	"net/http"
)

var ValidAPIScopes = map[string]struct{}{
	"organizations:read":  {},
	"organizations:write": {},
	"builders:read":       {},
	"builders:write":      {},
}

type UserAPIKey struct {
	ID         string
	Name       string
	KeyPrefix  string
	Scopes     []string
	CreatedAt  int64
	LastUsedAt int64
	ExpiresAt  int64
}

type CreateAPIKeyResult struct {
	Key   UserAPIKey
	Token string
}

func ValidateScopes(scopes []string) error {
	for _, s := range scopes {
		if _, ok := ValidAPIScopes[s]; !ok {
			return fmt.Errorf("invalid scope %q (valid: organizations:read, organizations:write, builders:read, builders:write)", s)
		}
	}
	return nil
}

func (c *Client) ListUserAPIKeys(ctx context.Context) ([]UserAPIKey, error) {
	var resp map[string]any
	if err := c.Do(ctx, http.MethodGet, "/v1/auth/api-keys", nil, &resp); err != nil {
		return nil, err
	}
	keysRaw, _ := resp["keys"].([]any)
	out := make([]UserAPIKey, 0, len(keysRaw))
	for _, item := range keysRaw {
		raw, _ := item.(map[string]any)
		out = append(out, mapAPIKey(raw))
	}
	return out, nil
}

func (c *Client) CreateUserAPIKey(ctx context.Context, name string, scopes []string, expiresInDays int) (*CreateAPIKeyResult, error) {
	if err := ValidateScopes(scopes); err != nil {
		return nil, err
	}
	var resp map[string]any
	if err := c.Do(ctx, http.MethodPost, "/v1/auth/api-keys", map[string]any{
		"name":             name,
		"scopes":           scopes,
		"expires_in_days":  expiresInDays,
	}, &resp); err != nil {
		return nil, err
	}
	keyRaw, _ := resp["key"].(map[string]any)
	return &CreateAPIKeyResult{
		Key:   mapAPIKey(keyRaw),
		Token: stringField(resp, "token"),
	}, nil
}

func (c *Client) RevokeUserAPIKey(ctx context.Context, id string) error {
	return c.Do(ctx, http.MethodDelete, "/v1/auth/api-keys/"+id, nil, nil)
}

func mapAPIKey(raw map[string]any) UserAPIKey {
	if raw == nil {
		return UserAPIKey{}
	}
	return UserAPIKey{
		ID:         stringField(raw, "id"),
		Name:       stringField(raw, "name"),
		KeyPrefix:  stringField(raw, "key_prefix", "keyPrefix"),
		Scopes:     stringSliceField(raw, "scopes"),
		CreatedAt:  intField(raw, "created_at", "createdAt"),
		LastUsedAt: intField(raw, "last_used_at", "lastUsedAt"),
		ExpiresAt:  intField(raw, "expires_at", "expiresAt"),
	}
}
