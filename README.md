# Relay

A configurable reverse proxy for LLM APIs. Route requests to OpenAI, Anthropic, and other providers through a single endpoint.

## Features

- Proxy requests to multiple LLM providers (OpenAI, Anthropic)
- Configurable via YAML with environment variable expansion
- API key injection — centralize keys or let clients pass their own
- Streaming support (SSE) out of the box
- Request logging

## Getting Started

```bash
# Copy and edit config
cp config.yaml.example config.yaml
# Set your API keys
export OPENAI_API_KEY=sk-...
export ANTHROPIC_API_KEY=sk-ant-...

# Run
go run . -config config.yaml
```

## Usage

Requests are routed by path prefix matching the provider name:

```bash
# OpenAI
curl http://localhost:8080/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}'

# Anthropic
curl http://localhost:8080/anthropic/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-20250514","max_tokens":100,"messages":[{"role":"user","content":"hello"}]}'
```

## Configuration

See [config.yaml.example](config.yaml.example) for the full reference.

```yaml
server:
  port: 8080

providers:
  openai:
    base_url: https://api.openai.com
    api_key: ${OPENAI_API_KEY}
    auth_header: Authorization
    auth_scheme: Bearer

  anthropic:
    base_url: https://api.anthropic.com
    api_key: ${ANTHROPIC_API_KEY}
    auth_header: x-api-key
    auth_scheme: ""
```

## Building

```bash
go build -o relay .
./relay -config config.yaml
```
