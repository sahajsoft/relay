package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func NewProviderProxy(name string, cfg ProviderConfig) (http.Handler, error) {
	target, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base_url for %q: %w", name, err)
	}

	prefix := "/" + name

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host

			// Strip the provider prefix from the path
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.URL.RawPath = ""

			// Inject API key if configured and not already present
			if cfg.APIKey != "" && cfg.AuthHeader != "" {
				if req.Header.Get(cfg.AuthHeader) == "" {
					value := cfg.APIKey
					if cfg.AuthScheme != "" {
						value = cfg.AuthScheme + " " + cfg.APIKey
					}
					req.Header.Set(cfg.AuthHeader, value)
				}
			}

			slog.Info("proxying request",
				"provider", name,
				"method", req.Method,
				"path", req.URL.Path,
			)
		},
		FlushInterval: -1, // Flush immediately for SSE/streaming
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Error("proxy error",
				"provider", name,
				"path", r.URL.Path,
				"error", err,
			)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			fmt.Fprintf(w, `{"error":{"message":"proxy error: %s","type":"proxy_error"}}`, err.Error())
		},
	}

	return proxy, nil
}
