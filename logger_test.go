package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewConversationLogger(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	cl, err := NewConversationLogger(dir)
	if err != nil {
		t.Fatalf("NewConversationLogger: %v", err)
	}
	if cl.dir != dir {
		t.Errorf("dir: got %q, want %q", cl.dir, dir)
	}
}

func TestConversationLoggerWrap(t *testing.T) {
	dir := t.TempDir()
	cl, _ := NewConversationLogger(dir)

	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"reply":"hello"}`))
	})

	handler := cl.Wrap("anthropic", backend)
	body := bytes.NewBufferString(`{"message":"hi"}`)
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", body)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}

	logFile := filepath.Join(dir, time.Now().Format("2006-01-02")+".jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}

	var entry ConversationEntry
	if err := json.Unmarshal(bytes.TrimSpace(data), &entry); err != nil {
		t.Fatalf("parsing log entry: %v", err)
	}

	if entry.Provider != "anthropic" {
		t.Errorf("provider: got %q, want %q", entry.Provider, "anthropic")
	}
	if entry.Status != 200 {
		t.Errorf("status: got %d, want %d", entry.Status, 200)
	}
	if !strings.Contains(string(entry.Request), "hi") {
		t.Errorf("request body not captured: %s", entry.Request)
	}
	if !strings.Contains(string(entry.Response), "hello") {
		t.Errorf("response body not captured: %s", entry.Response)
	}
}

func TestCaptureRequestBody(t *testing.T) {
	body := strings.NewReader(`{"test":true}`)
	req := httptest.NewRequest("POST", "/test", body)

	captured := captureRequestBody(req)

	if string(captured) != `{"test":true}` {
		t.Errorf("got %q, want %q", captured, `{"test":true}`)
	}

	secondRead := captureRequestBody(req)
	if string(secondRead) != `{"test":true}` {
		t.Errorf("body not replayable: got %q", secondRead)
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{"valid json", []byte(`{"a":1}`), `{"a":1}`},
		{"empty input", nil, "null"},
		{"invalid json", []byte("not json"), "null"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(normalizeJSON(tt.input))
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
