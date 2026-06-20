package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesLocalDotenvBeforeHomeDotenv(t *testing.T) {
	work := t.TempDir()
	home := t.TempDir()
	t.Chdir(work)
	t.Setenv("HOME", home)
	clearConfigEnv(t)

	homeConfig := filepath.Join(home, ".pccli")
	if err := os.MkdirAll(homeConfig, 0o700); err != nil {
		t.Fatalf("create home config dir: %v", err)
	}
	writeFile(t, filepath.Join(homeConfig, ".env"), "PCCLI_CLIENT_ID=home-client\nPCCLI_CLIENT_SECRET=home-secret\nPCCLI_MOCK=true\n")
	writeFile(t, filepath.Join(work, ".env"), "PCCLI_CLIENT_ID=local-client\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ClientID != "local-client" {
		t.Fatalf("expected local client ID, got %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "home-secret" {
		t.Fatalf("expected home client secret fallback, got %q", cfg.ClientSecret)
	}
	if !cfg.Mock {
		t.Fatalf("expected home mock setting fallback")
	}
}

func TestLoadUsesHomeDotenvWhenLocalDotenvMissing(t *testing.T) {
	work := t.TempDir()
	home := t.TempDir()
	t.Chdir(work)
	t.Setenv("HOME", home)
	clearConfigEnv(t)

	homeConfig := filepath.Join(home, ".pccli")
	if err := os.MkdirAll(homeConfig, 0o700); err != nil {
		t.Fatalf("create home config dir: %v", err)
	}
	writeFile(t, filepath.Join(homeConfig, ".env"), "PCCLI_CLIENT_ID=home-client\nPCCLI_CLIENT_SECRET=home-secret\nPCCLI_MOCK=true\n")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ClientID != "home-client" || cfg.ClientSecret != "home-secret" {
		t.Fatalf("expected home credentials, got %#v", cfg)
	}
}

func TestLoadUsesShellEnvBeforeDotenvFiles(t *testing.T) {
	work := t.TempDir()
	home := t.TempDir()
	t.Chdir(work)
	t.Setenv("HOME", home)
	clearConfigEnv(t)

	writeFile(t, filepath.Join(work, ".env"), "PCCLI_CLIENT_ID=local-client\nPCCLI_CLIENT_SECRET=local-secret\nPCCLI_MOCK=true\n")
	t.Setenv("PCCLI_CLIENT_ID", "env-client")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ClientID != "env-client" {
		t.Fatalf("expected env client ID, got %q", cfg.ClientID)
	}
	if cfg.ClientSecret != "local-secret" {
		t.Fatalf("expected local client secret, got %q", cfg.ClientSecret)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"PCCLI_CLIENT_ID",
		"PCCLI_CLIENT_SECRET",
		"PCCLI_BASE_URL",
		"PCCLI_TIMEOUT",
		"PCCLI_MOCK",
		"PCCLI_MOCK_FIXTURE",
		"PCCLI_COLOR",
		"PCCLI_USER_AGENT",
		"PCO_CLIENT_ID",
		"PCO_CLIENT_SECRET",
		"PCO_BASE_URL",
		"PCO_TIMEOUT",
		"PCO_MOCK",
		"PCO_MOCK_FIXTURE",
		"PCO_COLOR",
		"PCO_USER_AGENT",
		"PLANNING_CENTER_CLIENT_ID",
		"PLANNING_CENTER_SECRET",
	} {
		oldValue, hadValue := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if hadValue {
				_ = os.Setenv(key, oldValue)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
