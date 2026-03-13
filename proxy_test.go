package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestStripRoutePrefix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{"strips openai prefix", "/openai/v1/chat/completions", "/openai", "/v1/chat/completions"},
		{"strips anthropic prefix", "/anthropic/v1/messages", "/anthropic", "/v1/messages"},
		{"empty path after strip defaults to root", "/openai", "/openai", "/"},
		{"no match leaves path unchanged", "/other/path", "/openai", "/other/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: &url.URL{Path: tt.path}}
			stripRoutePrefix(req, tt.prefix)
			if req.URL.Path != tt.expected {
				t.Errorf("got %q, want %q", req.URL.Path, tt.expected)
			}
		})
	}
}

func TestRewriteTarget(t *testing.T) {
	target, _ := url.Parse("https://api.openai.com")
	req := httptest.NewRequest("POST", "http://localhost:8080/openai/v1/chat", nil)

	rewriteTarget(req, target)

	if req.URL.Scheme != "https" {
		t.Errorf("scheme: got %q, want %q", req.URL.Scheme, "https")
	}
	if req.URL.Host != "api.openai.com" {
		t.Errorf("host: got %q, want %q", req.URL.Host, "api.openai.com")
	}
	if req.Host != "api.openai.com" {
		t.Errorf("req.Host: got %q, want %q", req.Host, "api.openai.com")
	}
}

func TestInjectAuthIfMissing(t *testing.T) {
	t.Run("injects bearer token when missing", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/chat", nil)
		cfg := ProviderConfig{APIKey: "sk-test", AuthHeader: "Authorization", AuthScheme: "Bearer"}

		injectAuthIfMissing(req, cfg)

		got := req.Header.Get("Authorization")
		if got != "Bearer sk-test" {
			t.Errorf("got %q, want %q", got, "Bearer sk-test")
		}
	})

	t.Run("injects key without scheme for anthropic style", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/messages", nil)
		cfg := ProviderConfig{APIKey: "sk-ant-test", AuthHeader: "x-api-key", AuthScheme: ""}

		injectAuthIfMissing(req, cfg)

		got := req.Header.Get("x-api-key")
		if got != "sk-ant-test" {
			t.Errorf("got %q, want %q", got, "sk-ant-test")
		}
	})

	t.Run("preserves existing auth header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/chat", nil)
		req.Header.Set("Authorization", "Bearer client-key")
		cfg := ProviderConfig{APIKey: "sk-proxy", AuthHeader: "Authorization", AuthScheme: "Bearer"}

		injectAuthIfMissing(req, cfg)

		got := req.Header.Get("Authorization")
		if got != "Bearer client-key" {
			t.Errorf("got %q, want %q", got, "Bearer client-key")
		}
	})

	t.Run("skips injection when no api key configured", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/v1/chat", nil)
		cfg := ProviderConfig{APIKey: "", AuthHeader: "Authorization", AuthScheme: "Bearer"}

		injectAuthIfMissing(req, cfg)

		if got := req.Header.Get("Authorization"); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestNewProviderProxy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Received-Path", r.URL.Path)
		w.Header().Set("X-Received-Auth", r.Header.Get("x-api-key"))
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	cfg := ProviderConfig{
		BaseURL:    backend.URL,
		APIKey:     "test-key",
		AuthHeader: "x-api-key",
	}

	proxy, err := NewProviderProxy("testprovider", cfg)
	if err != nil {
		t.Fatalf("NewProviderProxy: %v", err)
	}

	srv := httptest.NewServer(proxy)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/testprovider/v1/messages")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if got := resp.Header.Get("X-Received-Path"); got != "/v1/messages" {
		t.Errorf("path: got %q, want %q", got, "/v1/messages")
	}
	if got := resp.Header.Get("X-Received-Auth"); got != "test-key" {
		t.Errorf("auth: got %q, want %q", got, "test-key")
	}
}
