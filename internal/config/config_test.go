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

func TestSetKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

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
	t.Setenv("BUILDERHUB_API_URL", "")
	if got := ResolveAPIURL("http://flag", "http://profile"); got != "http://flag" {
		t.Fatalf("flag: got %q", got)
	}
	t.Setenv("BUILDERHUB_API_URL", "http://env")
	if got := ResolveAPIURL("", "http://profile"); got != "http://env" {
		t.Fatalf("env: got %q", got)
	}
}
