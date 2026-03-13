package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type ConversationEntry struct {
	ID        int64           `json:"id"`
	Timestamp string          `json:"timestamp"`
	Provider  string          `json:"provider"`
	Method    string          `json:"method"`
	Path      string          `json:"path"`
	Status    int             `json:"status"`
	Duration  string          `json:"duration"`
	Request   json.RawMessage `json:"request"`
	Response  json.RawMessage `json:"response"`
}

type ConversationLogger struct {
	dir     string
	counter atomic.Int64
}

func NewConversationLogger(dir string) (*ConversationLogger, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &ConversationLogger{dir: dir}, nil
}

func (cl *ConversationLogger) Wrap(provider string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestBody := captureRequestBody(r)
		recorder := newResponseRecorder(w)

		next.ServeHTTP(recorder, r)

		entry := ConversationEntry{
			ID:        cl.counter.Add(1),
			Timestamp: start.Format(time.RFC3339),
			Provider:  provider,
			Method:    r.Method,
			Path:      r.URL.Path,
			Status:    recorder.status,
			Duration:  time.Since(start).String(),
			Request:   normalizeJSON(requestBody),
			Response:  normalizeJSON(recorder.body.Bytes()),
		}

		cl.writeEntry(entry)
	})
}

func captureRequestBody(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body
}

func normalizeJSON(data []byte) json.RawMessage {
	if len(data) == 0 {
		return json.RawMessage("null")
	}
	if json.Valid(data) {
		return json.RawMessage(data)
	}
	return json.RawMessage("null")
}

func (cl *ConversationLogger) writeEntry(entry ConversationEntry) {
	filename := filepath.Join(cl.dir, time.Now().Format("2006-01-02")+".jsonl")

	data, err := json.Marshal(entry)
	if err != nil {
		slog.Error("failed to marshal conversation entry", "error", err)
		return
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file", "error", err)
		return
	}
	defer f.Close()

	f.Write(append(data, '\n'))
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
