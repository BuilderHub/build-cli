package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	f := defaultFile()
	f.Profiles["default"] = Profile{
		APIURL:       "http://localhost:8090",
		Organization: "org-1",
		AccessToken:  "jwt-token",
		RefreshToken: "refresh",
	}
	if err := Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := loaded.Profile("default")
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if p.APIURL != "http://localhost:8090" {
		t.Fatalf("APIURL = %q", p.APIURL)
	}
	if p.Organization != "org-1" {
		t.Fatalf("Organization = %q", p.Organization)
	}

	path := filepath.Join(dir, ConfigDirName, ConfigFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 0600", info.Mode().Perm())
	}
}

func TestSetKeyAPIURL(t *testing.T) {
	f := defaultFile()
	if err := f.SetKey("default", "api-url", "https://api.builder-hub.dev/"); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	p, _ := f.Profile("default")
	if p.APIURL != "https://api.builder-hub.dev" {
		t.Fatalf("APIURL = %q", p.APIURL)
	}
}

func TestSetKeyOrganization(t *testing.T) {
	f := defaultFile()
	if err := f.SetKey("default", "organization", "org-abc"); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	p, _ := f.Profile("default")
	if p.Organization != "org-abc" {
		t.Fatalf("Organization = %q", p.Organization)
	}
}

func TestResolveAPIURL(t *testing.T) {
	t.Setenv(EnvAPIURL, "")
	profile := Profile{APIURL: "https://api.mycompany.io"}

	if got := ResolveAPIURL("https://api.other.io", profile); got != "https://api.other.io" {
		t.Fatalf("flag: got %q", got)
	}
	if got := ResolveAPIURL("", profile); got != "https://api.mycompany.io" {
		t.Fatalf("profile: got %q", got)
	}
	t.Setenv(EnvAPIURL, "https://api.env.io")
	if got := ResolveAPIURL("", Profile{}); got != "https://api.env.io" {
		t.Fatalf("env: got %q", got)
	}
	t.Setenv(EnvAPIURL, "")
	if got := ResolveAPIURL("", Profile{}); got != DefaultAPIURL {
		t.Fatalf("default: got %q", got)
	}
}

func TestResolveToken(t *testing.T) {
	t.Setenv(EnvToken, "")
	profile := Profile{AccessToken: "jwt-from-profile", APIKey: "bh_key123"}

	if got := ResolveToken("flag-token", profile, false); got != "flag-token" {
		t.Fatalf("flag: got %q", got)
	}
	t.Setenv(EnvToken, "env-token")
	if got := ResolveToken("", profile, false); got != "env-token" {
		t.Fatalf("env: got %q", got)
	}
	t.Setenv(EnvToken, "")
	if got := ResolveToken("", profile, false); got != "jwt-from-profile" {
		t.Fatalf("profile access token: got %q", got)
	}
	profile.AccessToken = ""
	if got := ResolveToken("", profile, false); got != "bh_key123" {
		t.Fatalf("profile api key: got %q", got)
	}
	if got := ResolveToken("", profile, true); got != "" {
		t.Fatalf("envOnly: got %q", got)
	}
}

func TestIsAPIKey(t *testing.T) {
	if !IsAPIKey("bh_abc123") {
		t.Fatal("expected bh_ prefix to be API key")
	}
	if IsAPIKey("jwt-token") {
		t.Fatal("JWT should not be detected as API key")
	}
	if IsAPIKey("bh") {
		t.Fatal("short token should not be API key")
	}
}

func TestSetKeyAPIKey(t *testing.T) {
	f := defaultFile()
	if err := f.SetKey("default", "api-key", "bh_secret"); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	p, _ := f.Profile("default")
	if p.APIKey != "bh_secret" {
		t.Fatalf("APIKey = %q", p.APIKey)
	}
}

func TestSetKeyUnknown(t *testing.T) {
	f := defaultFile()
	err := f.SetKey("default", "unknown-key", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !errors.Is(err, ErrUnknownConfigKey) {
		t.Fatalf("error = %v", err)
	}
}

func TestProfileNotFound(t *testing.T) {
	f := defaultFile()
	_, err := f.Profile("missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("error = %v", err)
	}
}
