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
			rewriteTarget(req, target)
			stripRoutePrefix(req, prefix)
			injectAuthIfMissing(req, cfg)
			logProxyRequest(name, req)
		},
		FlushInterval: -1,
		ErrorHandler:  proxyErrorHandler(name),
	}

	return proxy, nil
}

func rewriteTarget(req *http.Request, target *url.URL) {
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.Host = target.Host
}

func stripRoutePrefix(req *http.Request, prefix string) {
	req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
	if req.URL.Path == "" {
		req.URL.Path = "/"
	}
	req.URL.RawPath = ""
}

func injectAuthIfMissing(req *http.Request, cfg ProviderConfig) {
	if cfg.APIKey == "" || cfg.AuthHeader == "" {
		return
	}
	if req.Header.Get(cfg.AuthHeader) != "" {
		return
	}
	value := cfg.APIKey
	if cfg.AuthScheme != "" {
		value = cfg.AuthScheme + " " + cfg.APIKey
	}
	req.Header.Set(cfg.AuthHeader, value)
}

func logProxyRequest(provider string, req *http.Request) {
	slog.Info("proxying request",
		"provider", provider,
		"method", req.Method,
		"path", req.URL.Path,
	)
}

func proxyErrorHandler(provider string) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("proxy error",
			"provider", provider,
			"path", r.URL.Path,
			"error", err,
		)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		fmt.Fprintf(w, `{"error":{"message":"proxy error: %s","type":"proxy_error"}}`, err.Error())
	}
}
