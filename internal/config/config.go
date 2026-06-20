package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultBaseURL = "https://api.planningcenteronline.com"
	DefaultTimeout = 30 * time.Second
)

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	Timeout      time.Duration
	Mock         bool
	MockFixture  string
	Color        string
	UserAgent    string
}

func Load() (Config, error) {
	loadDotenvFiles()

	v := viper.New()
	v.SetEnvPrefix("PCCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	setDefault(v, "base-url", DefaultBaseURL)
	setDefault(v, "timeout", DefaultTimeout.String())
	setDefault(v, "mock-fixture", "testdata/mocks/services-library.json")
	setDefault(v, "color", "auto")
	setDefault(v, "user-agent", "pccli/dev")

	timeoutValue := getString(v, "timeout", "PCO_TIMEOUT")
	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil {
		return Config{}, fmt.Errorf("invalid PCCLI_TIMEOUT %q: %w", timeoutValue, err)
	}

	cfg := Config{
		ClientID:     getString(v, "client-id", "PCO_CLIENT_ID"),
		ClientSecret: getString(v, "client-secret", "PCO_CLIENT_SECRET"),
		BaseURL:      strings.TrimRight(getString(v, "base-url", "PCO_BASE_URL"), "/"),
		Timeout:      timeout,
		Mock:         getBool(v, "mock", "PCO_MOCK"),
		MockFixture:  getString(v, "mock-fixture", "PCO_MOCK_FIXTURE"),
		Color:        strings.ToLower(getString(v, "color", "PCO_COLOR")),
		UserAgent:    getString(v, "user-agent", "PCO_USER_AGENT"),
	}

	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("PLANNING_CENTER_CLIENT_ID")
	}
	if cfg.ClientSecret == "" {
		cfg.ClientSecret = os.Getenv("PLANNING_CENTER_SECRET")
	}

	if cfg.Color != "auto" && cfg.Color != "always" && cfg.Color != "never" {
		return Config{}, fmt.Errorf("invalid PCCLI_COLOR %q: expected auto, always, or never", cfg.Color)
	}
	if cfg.BaseURL == "" {
		return Config{}, fmt.Errorf("PCCLI_BASE_URL cannot be empty")
	}
	if !cfg.Mock && (cfg.ClientID == "" || cfg.ClientSecret == "") {
		return Config{}, fmt.Errorf("missing Planning Center credentials: set PCCLI_CLIENT_ID and PCCLI_CLIENT_SECRET in .env or the environment")
	}

	return cfg, nil
}

func setDefault(v *viper.Viper, key string, value any) {
	v.SetDefault(key, value)
}

func loadDotenvFiles() {
	loadIfExists(".env")
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return
	}
	loadIfExists(home + string(os.PathSeparator) + ".pccli" + string(os.PathSeparator) + ".env")
}

func loadIfExists(path string) {
	if _, err := os.Stat(path); err == nil {
		_ = godotenv.Load(path)
	}
}

func getString(v *viper.Viper, key string, aliases ...string) string {
	if value := os.Getenv("PCCLI_" + envKey(key)); value != "" {
		return value
	}
	for _, alias := range aliases {
		if value := os.Getenv(alias); value != "" {
			return value
		}
	}
	return v.GetString(key)
}

func getBool(v *viper.Viper, key string, aliases ...string) bool {
	if value, ok := parseBoolEnv(os.Getenv("PCCLI_" + envKey(key))); ok {
		return value
	}
	for _, alias := range aliases {
		if value, ok := parseBoolEnv(os.Getenv(alias)); ok {
			return value
		}
	}
	return v.GetBool(key)
}

func envKey(key string) string {
	return strings.ToUpper(strings.NewReplacer("-", "_", ".", "_").Replace(key))
}

func parseBoolEnv(value string) (bool, bool) {
	switch strings.ToLower(value) {
	case "1", "t", "true", "y", "yes", "on":
		return true, true
	case "0", "f", "false", "n", "no", "off":
		return false, true
	default:
		return false, false
	}
}
