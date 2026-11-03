package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultBaseURL   = "https://api.planningcenteronline.com"
	DefaultUploadURL = "https://upload.planningcenteronline.com"
	DefaultTimeout   = 30 * time.Second
)

type Config struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	UploadURL    string
	Timeout      time.Duration
	Mock         bool
	MockFixture  string
	Color        string
	UserAgent    string
}

func Load() (Config, error) {
	if err := loadDotenvFiles(); err != nil {
		return Config{}, err
	}

	v := viper.New()
	v.SetEnvPrefix("PCCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	setDefault(v, "base-url", DefaultBaseURL)
	setDefault(v, "upload-url", DefaultUploadURL)
	setDefault(v, "timeout", DefaultTimeout.String())
	setDefault(v, "mock-fixture", "testdata/mocks/services-library.json")
	setDefault(v, "color", "auto")
	setDefault(v, "mock", "false")
	setDefault(v, "user-agent", "pccli/dev")

	timeoutValue := v.GetString("timeout")
	timeout, err := time.ParseDuration(timeoutValue)
	if err != nil {
		return Config{}, fmt.Errorf("invalid PCCLI_TIMEOUT %q: %w", timeoutValue, err)
	}

	mock, err := strconv.ParseBool(v.GetString("mock"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid PCCLI_MOCK: expected true or false")
	}

	cfg := Config{
		ClientID:     v.GetString("client-id"),
		ClientSecret: v.GetString("client-secret"),
		BaseURL:      strings.TrimRight(v.GetString("base-url"), "/"),
		UploadURL:    strings.TrimRight(v.GetString("upload-url"), "/"),
		Timeout:      timeout,
		Mock:         mock,
		MockFixture:  v.GetString("mock-fixture"),
		Color:        strings.ToLower(v.GetString("color")),
		UserAgent:    v.GetString("user-agent"),
	}

	if cfg.Color != "auto" && cfg.Color != "always" && cfg.Color != "never" {
		return Config{}, fmt.Errorf("invalid PCCLI_COLOR %q: expected auto, always, or never", cfg.Color)
	}
	if cfg.Timeout <= 0 {
		return Config{}, fmt.Errorf("PCCLI_TIMEOUT must be positive")
	}
	for _, endpoint := range []struct{ name, value string }{{"PCCLI_BASE_URL", cfg.BaseURL}, {"PCCLI_UPLOAD_URL", cfg.UploadURL}} {
		if err := validateURL(endpoint.name, endpoint.value); err != nil {
			return Config{}, err
		}
	}
	if !cfg.Mock && (cfg.ClientID == "" || cfg.ClientSecret == "") {
		return Config{}, fmt.Errorf("missing Planning Center credentials: set PCCLI_CLIENT_ID and PCCLI_CLIENT_SECRET in .env or the environment")
	}

	return cfg, nil
}

func setDefault(v *viper.Viper, key string, value any) {
	v.SetDefault(key, value)
}

// Shell values win over project values, which win over user-level values.
func loadDotenvFiles() error {
	if err := loadIfExists(".env"); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return loadIfExists(filepath.Join(home, ".pccli", ".env"))
}

func loadIfExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("cannot access configuration file %s", path)
	}
	if err := godotenv.Load(path); err != nil {
		// Parser errors can include source text; never echo credential lines.
		return fmt.Errorf("cannot load configuration file %s", path)
	}
	return nil
}

func validateURL(name, value string) error {
	endpoint, err := url.Parse(value)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Path != "" {
		return fmt.Errorf("%s must be an HTTP or HTTPS origin without credentials, path, query, or fragment", name)
	}
	return nil
}
