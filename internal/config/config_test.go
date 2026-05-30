package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	f := defaultFile()
	f.Profiles["default"] = Profile{
		Domain:       "localhost",
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
	if p.Domain != "localhost" {
		t.Fatalf("Domain = %q", p.Domain)
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

func TestSetKeyDomain(t *testing.T) {
	f := defaultFile()
	if err := f.SetKey("default", "domain", "https://api.builder-hub.dev"); err != nil {
		t.Fatalf("SetKey: %v", err)
	}
	p, _ := f.Profile("default")
	if p.Domain != "builder-hub.dev" {
		t.Fatalf("Domain = %q", p.Domain)
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

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"builder-hub.dev", "builder-hub.dev"},
		{"https://builder-hub.dev", "builder-hub.dev"},
		{"api.builder-hub.dev", "builder-hub.dev"},
		{"https://api.mycompany.io/", "mycompany.io"},
	}
	for _, tc := range tests {
		got, err := NormalizeDomain(tc.in)
		if err != nil {
			t.Fatalf("NormalizeDomain(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeDomain(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if _, err := NormalizeDomain(""); err == nil {
		t.Fatal("expected error for empty domain")
	}
}

func TestAPIURLFromDomain(t *testing.T) {
	if got := APIURLFromDomain("builder-hub.dev"); got != "https://api.builder-hub.dev" {
		t.Fatalf("got %q", got)
	}
	if got := APIURLFromDomain(""); got != "https://api.builder-hub.dev" {
		t.Fatalf("default got %q", got)
	}
}

func TestResolveAPIURL(t *testing.T) {
	t.Setenv(EnvDomain, "")
	profile := Profile{Domain: "mycompany.io"}

	if got := ResolveAPIURL("other.io", profile); got != "https://api.other.io" {
		t.Fatalf("flag: got %q", got)
	}
	if got := ResolveAPIURL("", profile); got != "https://api.mycompany.io" {
		t.Fatalf("profile: got %q", got)
	}
	t.Setenv(EnvDomain, "env.io")
	if got := ResolveAPIURL("", Profile{}); got != "https://api.env.io" {
		t.Fatalf("env: got %q", got)
	}
	t.Setenv(EnvDomain, "")
	if got := ResolveAPIURL("", Profile{}); got != "https://api.builder-hub.dev" {
		t.Fatalf("default: got %q", got)
	}
}
