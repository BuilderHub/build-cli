package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type memStore struct {
	access  string
	refresh string
}

func (m *memStore) Token() string                 { return m.access }
func (m *memStore) RefreshToken() string          { return m.refresh }
func (m *memStore) SetTokens(access, refresh string) error {
	if access != "" {
		m.access = access
	}
	if refresh != "" {
		m.refresh = refresh
	}
	return nil
}

func TestLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/login" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":          map[string]any{"id": "u1", "email": "a@b.com", "name": "Alice"},
			"access_token":  "access-1",
			"refresh_token": "refresh-1",
			"expires_in":    3600,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{}, "")
	result, err := c.Login(context.Background(), "a@b.com", "secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if result.AccessToken != "access-1" {
		t.Fatalf("AccessToken = %q", result.AccessToken)
	}
	if result.User.Email != "a@b.com" {
		t.Fatalf("Email = %q", result.User.Email)
	}
}

func TestListBuilders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/v1/namespaces/org-1/builders" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"builders": []any{
				map[string]any{
					"name": "b1",
					"spec": map[string]any{"mode": "sleepy", "replicas": float64(1), "expose": true},
					"status": map[string]any{
						"phase":              "Ready",
						"endpoint":           "10.0.0.1",
						"external_endpoint":  "tcp://b1.example.com:443",
					},
				},
			},
		})
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{access: "test-token"}, "")
	builders, err := c.ListBuilders(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListBuilders: %v", err)
	}
	if len(builders) != 1 || builders[0].Name != "b1" {
		t.Fatalf("builders = %+v", builders)
	}
	if builders[0].Spec.Mode != "sleepy" {
		t.Fatalf("mode = %q", builders[0].Spec.Mode)
	}
	if builders[0].Spec.Expose == nil || !*builders[0].Spec.Expose {
		t.Fatalf("expose = %+v", builders[0].Spec.Expose)
	}
	if builders[0].Status.ExternalEndpoint != "tcp://b1.example.com:443" {
		t.Fatalf("external_endpoint = %q", builders[0].Status.ExternalEndpoint)
	}
}

func TestGenerateBuilderCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		if r.URL.Path != "/v1/namespaces/org-1/builders/b1/credentials" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ca_pem":          "ca-data",
			"client_cert_pem": "cert-data",
			"client_key_pem":  "key-data",
			"endpoint":        "tcp://b1.example.com:443",
			"server_name":     "b1.example.com",
			"expires_at":      float64(1750000000),
		})
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{access: "test-token"}, "")
	creds, err := c.GenerateBuilderCredentials(context.Background(), "org-1", "b1")
	if err != nil {
		t.Fatalf("GenerateBuilderCredentials: %v", err)
	}
	if creds.CAPEM != "ca-data" || creds.ClientCertPEM != "cert-data" || creds.ClientKeyPEM != "key-data" {
		t.Fatalf("creds = %+v", creds)
	}
	if creds.Endpoint != "tcp://b1.example.com:443" || creds.ServerName != "b1.example.com" {
		t.Fatalf("endpoint/server_name = %q / %q", creds.Endpoint, creds.ServerName)
	}
	if creds.ExpiresAt != 1750000000 {
		t.Fatalf("expires_at = %d", creds.ExpiresAt)
	}
}

func TestAPIErrorParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid builder name"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{access: "token"}, "")
	_, err := c.GetBuilder(context.Background(), "org", "bad")
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if apiErr.Message != "invalid builder name" {
		t.Fatalf("Message = %q", apiErr.Message)
	}
}

func TestRefreshRetry(t *testing.T) {
	attempts := 0
	store := &memStore{access: "expired", refresh: "refresh-1"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/refresh":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "fresh-token"})
		case "/v1/namespaces/org/builders":
			attempts++
			if attempts == 1 {
				if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer expired") {
					t.Fatalf("first auth = %q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if r.Header.Get("Authorization") != "Bearer fresh-token" {
				t.Fatalf("second auth = %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"builders": []any{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, store, "")
	if _, err := c.ListBuilders(context.Background(), "org"); err != nil {
		t.Fatalf("ListBuilders: %v", err)
	}
	if store.access != "fresh-token" {
		t.Fatalf("store.access = %q", store.access)
	}
}

func TestValidateScopes(t *testing.T) {
	if err := ValidateScopes([]string{"builders:read"}); err != nil {
		t.Fatalf("valid scope rejected: %v", err)
	}
	if err := ValidateScopes([]string{"invalid:scope"}); err == nil {
		t.Fatal("invalid scope accepted")
	}
}

func TestRegister(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/register" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user":          map[string]any{"id": "u2", "email": "new@b.com", "name": "Bob"},
			"access_token":  "access-2",
			"refresh_token": "refresh-2",
			"expires_in":    7200,
		})
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{}, "")
	result, err := c.Register(context.Background(), "new@b.com", "secret", "Bob")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if result.AccessToken != "access-2" {
		t.Fatalf("AccessToken = %q", result.AccessToken)
	}
	if result.User.Name != "Bob" {
		t.Fatalf("Name = %q", result.User.Name)
	}
}

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/health" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, &memStore{}, "")
	status, err := c.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck: %v", err)
	}
	if status != "ok" {
		t.Fatalf("status = %q", status)
	}
}
