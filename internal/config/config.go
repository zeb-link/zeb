// Package config owns CLI credential and context files.
// It preserves the existing ~/.zlink location so the Go CLI can share
// auth state with earlier local tools while it is being developed.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	envAPIKey     = "ZLINK_API_KEY"
	envAPIURL     = "ZLINK_API_URL"
	envCollection = "ZLINK_COLLECTION"
	envDomain     = "ZLINK_DOMAIN"
	envSpace      = "ZLINK_SPACE"
)

// defaultAPIURL is the production API every user lands on out of the box.
// When Zebra Link gets a dedicated API domain, change THIS line and rebuild —
// `zeb login` deliberately does not persist the default into config, so
// existing installs pick the new value up on their next build.
//
// Development override (owner-only, intentionally undocumented): the hidden
// `--api-url` flag or ZLINK_API_URL, e.g. `zeb login --api-url
// http://localhost:3000`.
const defaultAPIURL = "https://app.zeblink.io/api/v1"

// DefaultAPIURL exposes the built-in production API URL to command code
// (login uses it to decide whether an override is in play).
func DefaultAPIURL() string {
	return defaultAPIURL
}

type Credentials struct {
	APIKey   string `json:"apiKey"`
	StoredAt string `json:"storedAt"`
}

type Config struct {
	APIURL           string `json:"apiUrl,omitempty"`
	ActiveSpace      string `json:"activeSpace,omitempty"`
	ActiveCollection string `json:"activeCollection,omitempty"`
	ActiveDomain     string `json:"activeDomain,omitempty"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".zlink"), nil
}

func CredentialsPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func ConfigPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadCredentials() (*Credentials, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var credentials Credentials
	if err := json.Unmarshal(data, &credentials); err != nil {
		return nil, err
	}
	if credentials.APIKey == "" {
		return nil, nil
	}
	return &credentials, nil
}

func SaveCredentials(apiKey string) error {
	credentials := Credentials{
		APIKey:   apiKey,
		StoredAt: time.Now().UTC().Format(time.RFC3339),
	}
	return writeJSON("credentials", credentials)
}

func ClearCredentials() error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func SaveConfig(cfg Config) error {
	return writeJSON("config", cfg)
}

func ResolveAPIKey(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if value := os.Getenv(envAPIKey); value != "" {
		return value, nil
	}
	credentials, err := LoadCredentials()
	if err != nil {
		return "", err
	}
	if credentials != nil {
		return credentials.APIKey, nil
	}
	return "", nil
}

func ResolveAPIURL(flagValue string) (string, error) {
	if flagValue != "" {
		return NormalizeAPIURL(flagValue), nil
	}
	if value := os.Getenv(envAPIURL); value != "" {
		return NormalizeAPIURL(value), nil
	}
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.APIURL != "" {
		return NormalizeAPIURL(cfg.APIURL), nil
	}
	return defaultAPIURL, nil
}

func ResolveSpace(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if value := os.Getenv(envSpace); value != "" {
		return value, nil
	}
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.ActiveSpace, nil
}

func ResolveCollection(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if value := os.Getenv(envCollection); value != "" {
		return value, nil
	}
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.ActiveCollection, nil
}

func ResolveDomain(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if value := os.Getenv(envDomain); value != "" {
		return value, nil
	}
	cfg, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return cfg.ActiveDomain, nil
}

func NormalizeAPIURL(value string) string {
	trimmed := strings.TrimRight(value, "/")
	if strings.HasSuffix(trimmed, "/api/v1") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/api") {
		return trimmed + "/v1"
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed
	}
	return trimmed + "/api/v1"
}

func SetValue(key string, value string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	switch key {
	case "apiUrl":
		cfg.APIURL = NormalizeAPIURL(value)
	case "activeSpace":
		cfg.ActiveSpace = value
	case "activeCollection":
		cfg.ActiveCollection = value
	case "activeDomain":
		cfg.ActiveDomain = value
	default:
		return errors.New("unknown config key")
	}
	return SaveConfig(cfg)
}

func UnsetValue(key string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	switch key {
	case "apiUrl":
		cfg.APIURL = ""
	case "activeSpace":
		cfg.ActiveSpace = ""
	case "activeCollection":
		cfg.ActiveCollection = ""
	case "activeDomain":
		cfg.ActiveDomain = ""
	default:
		return errors.New("unknown config key")
	}
	return SaveConfig(cfg)
}

func writeJSON(kind string, value any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	var path string
	if kind == "credentials" {
		path, err = CredentialsPath()
	} else {
		path, err = ConfigPath()
	}
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
