package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads valid config with env expansion", func(t *testing.T) {
		t.Setenv("TEST_API_KEY", "expanded-key")

		content := `
server:
  port: 9090
providers:
  openai:
    base_url: https://api.openai.com
    api_key: ${TEST_API_KEY}
    auth_header: Authorization
    auth_scheme: Bearer
`
		path := writeTestConfig(t, content)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}

		if cfg.Server.Port != 9090 {
			t.Errorf("port: got %d, want %d", cfg.Server.Port, 9090)
		}
		if cfg.Providers["openai"].APIKey != "expanded-key" {
			t.Errorf("api_key: got %q, want %q", cfg.Providers["openai"].APIKey, "expanded-key")
		}
	})

	t.Run("defaults port to 8080", func(t *testing.T) {
		content := `
providers:
  test:
    base_url: https://example.com
`
		path := writeTestConfig(t, content)
		cfg, err := LoadConfig(path)
		if err != nil {
			t.Fatalf("LoadConfig: %v", err)
		}
		if cfg.Server.Port != 8080 {
			t.Errorf("port: got %d, want %d", cfg.Server.Port, 8080)
		}
	})

	t.Run("errors on no providers", func(t *testing.T) {
		content := `server:
  port: 8080
`
		path := writeTestConfig(t, content)
		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("expected error for empty providers")
		}
	})

	t.Run("errors on missing base_url", func(t *testing.T) {
		content := `
providers:
  broken:
    api_key: test
`
		path := writeTestConfig(t, content)
		_, err := LoadConfig(path)
		if err == nil {
			t.Fatal("expected error for missing base_url")
		}
	})

	t.Run("errors on missing file", func(t *testing.T) {
		_, err := LoadConfig("/nonexistent/config.yaml")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing test config: %v", err)
	}
	return path
}
