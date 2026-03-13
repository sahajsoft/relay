# Relay

## Project Overview

Relay is a reverse proxy for LLM APIs written in Go. It routes requests to different LLM providers (OpenAI, Anthropic) based on URL path prefix. Module path: `github.com/mallikarjunabr/relay`.

## Architecture

- `main.go` — Entry point, flag parsing, mux setup, logging middleware, HTTP server
- `config.go` — YAML config types and loader with env var expansion
- `proxy.go` — Reverse proxy factory using `net/http/httputil.ReverseProxy`
- `config.yaml.example` — Reference config (real `config.yaml` is gitignored)

## Development

- Go version: 1.24+
- Build: `go build -o relay .`
- Run: `go run . -config config.yaml`
- Test: `go test ./...`
- Dependencies: `gopkg.in/yaml.v3`

## Key Design Decisions

- Single `package main`, no sub-packages
- `httputil.ReverseProxy` with `FlushInterval: -1` handles streaming/SSE natively
- API keys injected from config only if the client doesn't send their own auth header
- `os.ExpandEnv` on raw YAML enables `${ENV_VAR}` syntax in config
