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
