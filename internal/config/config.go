package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	DefaultAPIURL  = "https://api.builder-hub.dev"
	DefaultProfile = "default"
	ConfigDirName  = "builderhub"
	ConfigFileName = "config.yaml"
	EnvAPIURL      = "BUILDERHUB_API_URL"
	EnvToken       = "BUILDERHUB_TOKEN"
)

var (
	ErrProfileNotFound  = errors.New("profile not found")
	ErrUnknownConfigKey = errors.New("unknown config key")
)

type Profile struct {
	APIURL       string `mapstructure:"api_url" yaml:"api_url,omitempty"`
	Organization string `mapstructure:"organization" yaml:"organization,omitempty"`
	AccessToken  string `mapstructure:"access_token" yaml:"access_token,omitempty"`
	RefreshToken string `mapstructure:"refresh_token" yaml:"refresh_token,omitempty"`
	APIKey       string `mapstructure:"api_key" yaml:"api_key,omitempty"`
}

type File struct {
	CurrentProfile string             `mapstructure:"current_profile" yaml:"current_profile"`
	Profiles       map[string]Profile `mapstructure:"profiles" yaml:"profiles"`
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

func newConfigViper() (*viper.Viper, error) {
	dir, err := ConfigDir()
	if err != nil {
		return nil, err
	}
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigName(strings.TrimSuffix(ConfigFileName, filepath.Ext(ConfigFileName)))
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	v.SetConfigFile(path)

	v.SetEnvPrefix("BUILDERHUB")
	v.AutomaticEnv()
	_ = v.BindEnv("api_url")
	_ = v.BindEnv("token")

	v.SetDefault("current_profile", DefaultProfile)
	v.SetDefault("profiles", map[string]any{
		DefaultProfile: map[string]any{
			"api_url": DefaultAPIURL,
		},
	})

	return v, nil
}

func readConfigViper() (*viper.Viper, error) {
	v, err := newConfigViper()
	if err != nil {
		return nil, err
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	return v, nil
}

func normalizeFile(f *File) {
	if f.Profiles == nil {
		f.Profiles = map[string]Profile{}
	}
	if f.CurrentProfile == "" {
		f.CurrentProfile = DefaultProfile
	}
	if _, ok := f.Profiles[f.CurrentProfile]; !ok {
		f.Profiles[f.CurrentProfile] = defaultProfile()
	}
}

func Load() (*File, error) {
	v, err := readConfigViper()
	if err != nil {
		return nil, err
	}
	var f File
	if err := v.Unmarshal(&f); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	normalizeFile(&f)
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

	v, err := newConfigViper()
	if err != nil {
		return err
	}
	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) && !os.IsNotExist(err) {
			return fmt.Errorf("read config: %w", err)
		}
	}

	v.Set("current_profile", f.CurrentProfile)
	v.Set("profiles", f.Profiles)

	if err := v.WriteConfigAs(path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func defaultProfile() Profile {
	return Profile{APIURL: DefaultAPIURL}
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
	if p.APIURL == "" {
		p.APIURL = DefaultAPIURL
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
	if key == "api-url" || key == "api_url" {
		value = strings.TrimRight(strings.TrimSpace(value), "/")
	}
	return f.UpdateProfile(profileName, func(p *Profile) {
		switch key {
		case "api-url", "api_url":
			p.APIURL = value
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
	case "api-url", "api_url", "organization", "org", "access-token", "access_token", "refresh-token", "refresh_token", "api-key", "api_key":
		return true
	default:
		return false
	}
}

func envViper() (*viper.Viper, error) {
	return newConfigViper()
}

func ResolveAPIURL(apiURLFlag string, profile Profile) string {
	if v := strings.TrimRight(strings.TrimSpace(apiURLFlag), "/"); v != "" {
		return v
	}
	v, err := envViper()
	if err == nil {
		if envURL := strings.TrimRight(strings.TrimSpace(v.GetString("api_url")), "/"); envURL != "" {
			return envURL
		}
	}
	if profile.APIURL != "" {
		return profile.APIURL
	}
	return DefaultAPIURL
}

func ResolveToken(flagValue string, profile Profile, envOnly bool) string {
	if v := flagValue; v != "" {
		return v
	}
	v, err := envViper()
	if err == nil {
		if envToken := v.GetString("token"); envToken != "" {
			return envToken
		}
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
