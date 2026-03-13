package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	port := flag.Int("port", 0, "override server port")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	if *port != 0 {
		cfg.Server.Port = *port
	}

	mux := http.NewServeMux()

	// Register provider proxies
	for name, providerCfg := range cfg.Providers {
		proxy, err := NewProviderProxy(name, providerCfg)
		if err != nil {
			slog.Error("failed to create proxy", "provider", name, "error", err)
			os.Exit(1)
		}
		pattern := "/" + name + "/"
		mux.Handle(pattern, proxy)
		slog.Info("registered provider", "name", name, "target", providerCfg.BaseURL)
	}

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:              addr,
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	slog.Info("starting relay", "addr", addr)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).String(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
