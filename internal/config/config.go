package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultDomain  = "builder-hub.dev"
	DefaultProfile = "default"
	ConfigDirName  = "builderhub"
	ConfigFileName = "config.yaml"
	EnvDomain      = "BUILDERHUB_DOMAIN"
	EnvToken       = "BUILDERHUB_TOKEN"
)

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrUnknownConfigKey = errors.New("unknown config key")
	ErrInvalidDomain    = errors.New("invalid domain")
)

type Profile struct {
	Domain       string `yaml:"domain,omitempty"`
	Organization string `yaml:"organization,omitempty"`
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	APIKey       string `yaml:"api_key,omitempty"`
}

type File struct {
	CurrentProfile string             `yaml:"current_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, ConfigDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", ConfigDirName), nil
}

func Load() (*File, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultFile(), nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	if f.CurrentProfile == "" {
		f.CurrentProfile = DefaultProfile
	}
	if _, ok := f.Profiles[f.CurrentProfile]; !ok {
		f.Profiles[f.CurrentProfile] = defaultProfile()
	}
	return &f, nil
}

func Save(f *File) error {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	if f.CurrentProfile == "" {
		f.CurrentProfile = DefaultProfile
	}
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func defaultProfile() Profile {
	return Profile{Domain: DefaultDomain}
}

func defaultFile() *File {
	return &File{
		CurrentProfile: DefaultProfile,
		Profiles: map[string]Profile{
			DefaultProfile: defaultProfile(),
		},
	}
}

func (f *File) Profile(name string) (Profile, error) {
	if name == "" {
		name = f.CurrentProfile
	}
	p, ok := f.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("%w: %q", ErrProfileNotFound, name)
	}
	if p.Domain == "" {
		p.Domain = DefaultDomain
	}
	return p, nil
}

func (f *File) SetProfile(name string, p Profile) {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	f.Profiles[name] = p
}

func (f *File) UpdateProfile(name string, fn func(*Profile)) error {
	p, err := f.Profile(name)
	if err != nil {
		if !errors.Is(err, ErrProfileNotFound) {
			return err
		}
		p = defaultProfile()
	}
	fn(&p)
	f.SetProfile(name, p)
	return nil
}

func (f *File) SetKey(profileName, key, value string) error {
	if !isKnownConfigKey(key) {
		return fmt.Errorf("%w: %q", ErrUnknownConfigKey, key)
	}
	if key == "domain" {
		normalized, err := NormalizeDomain(value)
		if err != nil {
			return err
		}
		value = normalized
	}
	return f.UpdateProfile(profileName, func(p *Profile) {
		switch key {
		case "domain":
			p.Domain = value
		case "organization", "org":
			p.Organization = value
		case "access-token", "access_token":
			p.AccessToken = value
		case "refresh-token", "refresh_token":
			p.RefreshToken = value
		case "api-key", "api_key":
			p.APIKey = value
		}
	})
}

func isKnownConfigKey(key string) bool {
	switch key {
	case "domain", "organization", "org", "access-token", "access_token", "refresh-token", "refresh_token", "api-key", "api_key":
		return true
	default:
		return false
	}
}

func NormalizeDomain(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidDomain)
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrInvalidDomain, err)
		}
		s = u.Host
	}
	s = strings.TrimSuffix(s, "/")
	s = strings.TrimPrefix(s, "api.")
	if s == "" {
		return "", fmt.Errorf("%w: empty after normalization", ErrInvalidDomain)
	}
	return s, nil
}

func APIURLFromDomain(domain string) string {
	d := domain
	if d == "" {
		d = DefaultDomain
	}
	return "https://api." + d
}

func ResolveDomain(domainFlag string, profile Profile) string {
	if v := strings.TrimSpace(domainFlag); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv(EnvDomain)); v != "" {
		return v
	}
	if profile.Domain != "" {
		return profile.Domain
	}
	return DefaultDomain
}

func ResolveAPIURL(domainFlag string, profile Profile) string {
	return APIURLFromDomain(ResolveDomain(domainFlag, profile))
}

func ResolveToken(flagValue string, profile Profile, envOnly bool) string {
	if v := flagValue; v != "" {
		return v
	}
	if v := os.Getenv(EnvToken); v != "" {
		return v
	}
	if envOnly {
		return ""
	}
	if profile.AccessToken != "" {
		return profile.AccessToken
	}
	return profile.APIKey
}

func IsAPIKey(token string) bool {
	return len(token) >= 3 && token[:3] == "bh_"
}
