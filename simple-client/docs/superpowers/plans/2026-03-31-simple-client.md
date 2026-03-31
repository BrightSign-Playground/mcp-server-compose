# simple-client Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a brandable web Q&A client that queries the RAG MCP server, streams answers from Claude, and lets users navigate history of question/answer pairs.

**Architecture:** Go backend behind Caddy serves a vanilla TypeScript SPA. The backend owns all secrets (OIDC tokens, Anthropic key) and exposes a single SSE endpoint `POST /api/ask`. The frontend handles UI state, history persistence in sessionStorage, and Markdown rendering. Caddy reverse-proxies `/api/*` to the Go backend and serves static assets directly.

**Tech Stack:** Go 1.24 (stdlib net/http + BurntSushi/toml), TypeScript 5 (vanilla DOM, no framework), Vite (bundler), Caddy (web server), marked (Markdown), DOMPurify (XSS sanitization).

---

## File Structure

```
simple-client/
├── REQUIREMENTS.md                  # spec (already exists)
├── docs/superpowers/plans/          # this plan
├── config.toml.example              # committed config template
├── Caddyfile                        # reverse proxy + static files
├── Makefile                         # build/test/run targets
├── .gitignore
│
├── backend/
│   ├── go.mod                       # module github.com/brightsign-playground/simple-client
│   ├── go.sum
│   ├── cmd/
│   │   └── server/
│   │       └── main.go              # entry point: load config, wire deps, start server
│   └── internal/
│       ├── config/
│       │   ├── config.go            # TOML loader + env override + validation
│       │   └── config_test.go
│       ├── auth/
│       │   ├── token.go             # OIDC client_credentials token fetch + cache
│       │   └── token_test.go
│       ├── mcpclient/
│       │   ├── client.go            # MCP Streamable HTTP client (init, notify, tool call)
│       │   └── client_test.go
│       ├── llm/
│       │   ├── claude.go            # Anthropic streaming API client
│       │   └── claude_test.go
│       ├── api/
│       │   ├── ask.go               # POST /api/ask handler (orchestrates auth+mcp+llm, writes SSE)
│       │   └── ask_test.go
│       └── server/
│           └── server.go            # HTTP mux assembly, branding routes, graceful shutdown
│
├── frontend/
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── index.html                   # SPA shell
│   └── src/
│       ├── main.ts                  # entry: wire UI, keyboard shortcuts, initial render
│       ├── types.ts                 # QAPair, ChunkResult, SSE event interfaces
│       ├── api.ts                   # POST /api/ask, SSE EventSource parsing
│       ├── history.ts               # QAPair store, sessionStorage persistence, navigation
│       ├── ui.ts                    # DOM manipulation: render pair, update slider, states
│       ├── markdown.ts              # marked + DOMPurify wrapper
│       ├── styles.css               # default theme with CSS custom properties
│       └── __tests__/
│           ├── history.test.ts      # history store + navigation logic
│           └── markdown.test.ts     # sanitization tests
│
└── dist/                            # gitignored -- vite build output
```

**Design decisions:**
- Backend is its own Go module (matches repo pattern: each subproject has its own go.mod).
- Frontend uses Vite for fast dev + production bundling. No framework -- vanilla TS is sufficient for this UI complexity.
- `marked` for Markdown rendering (small, fast, extensible). `DOMPurify` for XSS sanitization (industry standard).
- Caddy runs as a separate process via Caddyfile -- simplest approach, no Go embedding needed. The Makefile orchestrates both.

---

## Task 1: Project scaffolding and .gitignore

**Files:**
- Create: `simple-client/.gitignore`
- Create: `simple-client/config.toml.example`
- Create: `simple-client/Caddyfile`
- Create: `simple-client/Makefile`

- [ ] **Step 1: Create .gitignore**

```gitignore
# Secrets
config.toml
.env
.envrc

# Build artifacts
dist/
bin/
node_modules/

# Editor
*~
.vscode/
.idea/
```

- [ ] **Step 2: Create config.toml.example**

Copy the full annotated TOML schema from REQUIREMENTS.md section "Config file: config.toml". Add `backend_port = ":8091"` to the `[server]` section. Replace real secrets with placeholder values (`sk-ant-...`, `changeme-dev-secret`).

- [ ] **Step 3: Create Caddyfile**

```
{$LISTEN_ADDR::8090} {
    handle /api/* {
        reverse_proxy localhost:{$BACKEND_PORT:8091}
    }
    handle /logo {
        reverse_proxy localhost:{$BACKEND_PORT:8091}
    }
    handle /custom.css {
        reverse_proxy localhost:{$BACKEND_PORT:8091}
    }
    handle {
        root * {$STATIC_DIR:./dist}
        try_files {path} /index.html
        file_server
    }
}
```

- [ ] **Step 4: Create Makefile**

```makefile
.DEFAULT_GOAL := help

BACKEND_BIN := bin/server
FRONTEND_DIR := frontend
BACKEND_DIR := backend
DIST_DIR := dist

help: ## Print available targets
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-24s %s\n", $$1, $$2}'

build: build-frontend build-backend ## Build frontend and backend

build-frontend: ## Compile TypeScript and bundle into dist/
	cd $(FRONTEND_DIR) && npm run build

build-backend: ## Build Go backend binary
	cd $(BACKEND_DIR) && go build -o ../$(BACKEND_BIN) ./cmd/server

test: test-frontend test-backend ## Run all tests

test-frontend: ## Run frontend unit tests
	cd $(FRONTEND_DIR) && npm test

test-backend: ## Run backend unit tests
	cd $(BACKEND_DIR) && go test ./...

lint: ## Run linters
	cd $(BACKEND_DIR) && go vet ./...

dev: ## Start backend + frontend dev server (requires Caddy running separately)
	cd $(FRONTEND_DIR) && npm run dev &
	cd $(BACKEND_DIR) && go run ./cmd/server

run: build ## Build and start production server (Caddy + backend)
	$(BACKEND_BIN) &
	caddy run --config Caddyfile

clean: ## Remove build artifacts
	rm -rf $(BACKEND_BIN) $(DIST_DIR) bin/
	cd $(FRONTEND_DIR) && rm -rf node_modules

deps: ## Install all dependencies
	cd $(FRONTEND_DIR) && npm install
	cd $(BACKEND_DIR) && go mod download && go mod tidy
```

- [ ] **Step 5: Commit**

```bash
git add simple-client/.gitignore simple-client/config.toml.example simple-client/Caddyfile simple-client/Makefile
git commit -m "scaffold: add simple-client project skeleton with Makefile and Caddyfile."
```

---

## Task 2: Go module and config loader

**Files:**
- Create: `simple-client/backend/go.mod`
- Create: `simple-client/backend/internal/config/config.go`
- Create: `simple-client/backend/internal/config/config_test.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd simple-client/backend
go mod init github.com/brightsign-playground/simple-client
go get github.com/BurntSushi/toml
```

- [ ] **Step 2: Write failing config tests**

Create `internal/config/config_test.go`:

```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brightsign-playground/simple-client/internal/config"
)

func writeToml(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validToml = `
[server]
listen = ":9090"
static_dir = "./dist"

[mcp]
server_url = "http://localhost:15080"
limit = 5

[auth]
provider = "keycloak"
keycloak_issuer = "http://localhost:8080/realms/dev"
keycloak_client_id = "my-app"
keycloak_client_secret = "secret123"

[llm]
anthropic_api_key = "sk-ant-test"
model = "claude-sonnet-4-10"
max_tokens = 2048
system_prompt = "You are helpful."

[branding]
title = "Test App"
logo = ""
custom_css = ""
`

func TestLoadValidConfig(t *testing.T) {
	path := writeToml(t, validToml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Listen != ":9090" {
		t.Errorf("listen = %q, want %q", cfg.Server.Listen, ":9090")
	}
	if cfg.MCP.Limit != 5 {
		t.Errorf("limit = %d, want 5", cfg.MCP.Limit)
	}
	if cfg.Auth.Provider != "keycloak" {
		t.Errorf("provider = %q, want keycloak", cfg.Auth.Provider)
	}
	if cfg.LLM.Model != "claude-sonnet-4-10" {
		t.Errorf("model = %q, want claude-sonnet-4-10", cfg.LLM.Model)
	}
	if cfg.Branding.Title != "Test App" {
		t.Errorf("title = %q, want Test App", cfg.Branding.Title)
	}
}

func TestLoadEnvOverridesSecrets(t *testing.T) {
	path := writeToml(t, validToml)
	t.Setenv("ANTHROPIC_API_KEY", "sk-env-override")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLM.AnthropicAPIKey != "sk-env-override" {
		t.Errorf("api key = %q, want sk-env-override", cfg.LLM.AnthropicAPIKey)
	}
}

func TestLoadMissingRequiredFields(t *testing.T) {
	path := writeToml(t, `
[server]
listen = ":8090"
[mcp]
server_url = "http://localhost:15080"
[auth]
provider = "keycloak"
[llm]
[branding]
title = "Test"
`)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for missing required fields")
	}
}

func TestLoadInvalidProvider(t *testing.T) {
	toml := `
[server]
listen = ":8090"
[mcp]
server_url = "http://localhost:15080"
[auth]
provider = "auth0"
keycloak_issuer = "http://x"
keycloak_client_id = "x"
keycloak_client_secret = "x"
[llm]
anthropic_api_key = "sk-x"
model = "claude-sonnet-4-10"
max_tokens = 1024
system_prompt = "x"
[branding]
title = "X"
`
	path := writeToml(t, toml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
}

func TestLoadDefaults(t *testing.T) {
	toml := `
[server]
[mcp]
server_url = "http://localhost:15080"
[auth]
provider = "keycloak"
keycloak_issuer = "http://localhost:8080/realms/dev"
keycloak_client_id = "my-app"
keycloak_client_secret = "secret"
[llm]
anthropic_api_key = "sk-test"
model = "claude-sonnet-4-10"
max_tokens = 2048
system_prompt = "helper"
[branding]
title = "Default"
`
	path := writeToml(t, toml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Listen != ":8090" {
		t.Errorf("default listen = %q, want :8090", cfg.Server.Listen)
	}
	if cfg.MCP.Limit != 5 {
		t.Errorf("default limit = %d, want 5", cfg.MCP.Limit)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
cd simple-client/backend && go test ./internal/config/...
```

Expected: compilation error -- `config` package does not exist yet.

- [ ] **Step 4: Implement config.go**

Create `internal/config/config.go`:

```go
package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	Listen      string `toml:"listen"`
	BackendPort string `toml:"backend_port"`
	StaticDir   string `toml:"static_dir"`
}

type MCPConfig struct {
	ServerURL string `toml:"server_url"`
	Limit     int    `toml:"limit"`
}

type AuthConfig struct {
	Provider             string `toml:"provider"`
	KeycloakIssuer       string `toml:"keycloak_issuer"`
	KeycloakClientID     string `toml:"keycloak_client_id"`
	KeycloakClientSecret string `toml:"keycloak_client_secret"`
	LogtoIssuer          string `toml:"logto_issuer"`
	LogtoAppID           string `toml:"logto_app_id"`
	LogtoAppSecret       string `toml:"logto_app_secret"`
	LogtoAudience        string `toml:"logto_audience"`
}

type LLMConfig struct {
	AnthropicAPIKey string `toml:"anthropic_api_key"`
	Model           string `toml:"model"`
	MaxTokens       int    `toml:"max_tokens"`
	SystemPrompt    string `toml:"system_prompt"`
}

type BrandingConfig struct {
	Title     string `toml:"title"`
	Logo      string `toml:"logo"`
	CustomCSS string `toml:"custom_css"`
}

type Config struct {
	Server   ServerConfig   `toml:"server"`
	MCP      MCPConfig      `toml:"mcp"`
	Auth     AuthConfig     `toml:"auth"`
	LLM      LLMConfig      `toml:"llm"`
	Branding BrandingConfig `toml:"branding"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Listen:      ":8090",
			BackendPort: ":8091",
			StaticDir: "./dist",
		},
		MCP: MCPConfig{
			Limit: 5,
		},
		Branding: BrandingConfig{
			Title: "Support Search",
		},
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// Environment variable overrides for secrets.
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.LLM.AnthropicAPIKey = v
	}
	if v := os.Getenv("KEYCLOAK_CLIENT_SECRET"); v != "" {
		cfg.Auth.KeycloakClientSecret = v
	}
	if v := os.Getenv("LOGTO_APP_SECRET"); v != "" {
		cfg.Auth.LogtoAppSecret = v
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if cfg.MCP.ServerURL == "" {
		return fmt.Errorf("config: mcp.server_url is required")
	}
	switch cfg.Auth.Provider {
	case "keycloak":
		if cfg.Auth.KeycloakIssuer == "" || cfg.Auth.KeycloakClientID == "" || cfg.Auth.KeycloakClientSecret == "" {
			return fmt.Errorf("config: keycloak provider requires keycloak_issuer, keycloak_client_id, keycloak_client_secret")
		}
	case "logto":
		if cfg.Auth.LogtoIssuer == "" || cfg.Auth.LogtoAppID == "" || cfg.Auth.LogtoAppSecret == "" {
			return fmt.Errorf("config: logto provider requires logto_issuer, logto_app_id, logto_app_secret")
		}
	default:
		return fmt.Errorf("config: auth.provider must be keycloak or logto, got %q", cfg.Auth.Provider)
	}
	if cfg.LLM.AnthropicAPIKey == "" {
		return fmt.Errorf("config: llm.anthropic_api_key is required (set in config.toml or ANTHROPIC_API_KEY env)")
	}
	if cfg.LLM.Model == "" {
		return fmt.Errorf("config: llm.model is required")
	}
	if cfg.LLM.MaxTokens < 1 {
		return fmt.Errorf("config: llm.max_tokens must be >= 1")
	}
	if cfg.LLM.SystemPrompt == "" {
		return fmt.Errorf("config: llm.system_prompt is required")
	}
	if cfg.Branding.Title == "" {
		return fmt.Errorf("config: branding.title is required")
	}
	if cfg.MCP.Limit < 1 || cfg.MCP.Limit > 20 {
		cfg.MCP.Limit = 5
	}
	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
cd simple-client/backend && go test ./internal/config/... -v
```

Expected: all 5 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add simple-client/backend/go.mod simple-client/backend/go.sum simple-client/backend/internal/config/
git commit -m "feat: add config loader with TOML parsing, env overrides, and validation."
```

---

## Task 3: OIDC token fetcher with caching

**Files:**
- Create: `simple-client/backend/internal/auth/token.go`
- Create: `simple-client/backend/internal/auth/token_test.go`

- [ ] **Step 1: Write failing token tests**

Create `internal/auth/token_test.go`:

```go
package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brightsign-playground/simple-client/internal/auth"
)

func TestFetchToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("grant_type") != "client_credentials" {
			t.Errorf("grant_type = %q, want client_credentials", r.FormValue("grant_type"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-jwt-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	cache := auth.NewTokenCache(srv.URL+"/token", "client-id", "client-secret", "")
	token, err := cache.Token()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-jwt-token" {
		t.Errorf("token = %q, want test-jwt-token", token)
	}
}

func TestTokenCaching(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	cache := auth.NewTokenCache(srv.URL+"/token", "id", "secret", "")
	_, _ = cache.Token()
	_, _ = cache.Token()
	_, _ = cache.Token()

	if n := calls.Load(); n != 1 {
		t.Errorf("token endpoint called %d times, want 1 (should be cached)", n)
	}
}

func TestTokenRefreshBeforeExpiry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "token-" + string(rune('0'+n)),
			"expires_in":   1, // 1 second -- will expire immediately
			"token_type":   "Bearer",
		})
	}))
	defer srv.Close()

	cache := auth.NewTokenCache(srv.URL+"/token", "id", "secret", "")
	_, _ = cache.Token()
	time.Sleep(2 * time.Second)
	_, _ = cache.Token()

	if n := calls.Load(); n < 2 {
		t.Errorf("token endpoint called %d times, want >= 2 (expired token should trigger refresh)", n)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/backend && go test ./internal/auth/... -v
```

Expected: compilation error -- `auth` package does not exist.

- [ ] **Step 3: Implement token.go**

Create `internal/auth/token.go`:

```go
package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// TokenCache fetches and caches OIDC client_credentials tokens.
// It is safe for concurrent use.
type TokenCache struct {
	tokenURL     string
	clientID     string
	clientSecret string
	audience     string // optional; sent as "resource" for Logto

	mu        sync.Mutex
	token     string
	expiresAt time.Time
	client    *http.Client
}

func NewTokenCache(tokenURL, clientID, clientSecret, audience string) *TokenCache {
	return &TokenCache{
		tokenURL:     tokenURL,
		clientID:     clientID,
		clientSecret: clientSecret,
		audience:     audience,
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

// Token returns a valid access token, fetching or refreshing as needed.
func (tc *TokenCache) Token() (string, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Return cached token if it has at least 30 seconds of validity remaining.
	if tc.token != "" && time.Now().Add(30*time.Second).Before(tc.expiresAt) {
		return tc.token, nil
	}

	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {tc.clientID},
		"client_secret": {tc.clientSecret},
	}
	if tc.audience != "" {
		form.Set("resource", tc.audience)
		form.Set("scope", "openid")
	}

	resp, err := tc.client.Post(tc.tokenURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("auth: token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("auth: reading token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth: token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("auth: parsing token response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("auth: no access_token in response")
	}

	tc.token = tokenResp.AccessToken
	tc.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	return tc.token, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/backend && go test ./internal/auth/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add simple-client/backend/internal/auth/
git commit -m "feat: add OIDC client_credentials token cache with auto-refresh."
```

---

## Task 4: MCP protocol client

**Files:**
- Create: `simple-client/backend/internal/mcpclient/client.go`
- Create: `simple-client/backend/internal/mcpclient/client_test.go`

- [ ] **Step 1: Write failing MCP client tests**

Create `internal/mcpclient/client_test.go`:

```go
package mcpclient_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brightsign-playground/simple-client/internal/mcpclient"
)

func TestSearchDocuments(t *testing.T) {
	step := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		method, _ := req["method"].(string)

		switch step {
		case 0: // initialize
			if method != "initialize" {
				t.Errorf("step 0: method = %q, want initialize", method)
			}
			w.Header().Set("Mcp-Session-Id", "test-session")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"protocolVersion": "2025-06-18",
					"capabilities":    map[string]interface{}{},
					"serverInfo":      map[string]interface{}{"name": "test", "version": "0.1"},
				},
			})
		case 1: // notifications/initialized
			if method != "notifications/initialized" {
				t.Errorf("step 1: method = %q, want notifications/initialized", method)
			}
			w.WriteHeader(http.StatusAccepted)
		case 2: // tools/call
			if method != "tools/call" {
				t.Errorf("step 2: method = %q, want tools/call", method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"result": map[string]interface{}{
					"structuredContent": map[string]interface{}{
						"results": []map[string]interface{}{
							{"id": 1, "content": "test chunk", "score": 0.9, "source_path": "test.md", "title": "Test"},
						},
						"total_chunks_in_db": 100,
					},
				},
			})
		}
		step++
	}))
	defer srv.Close()

	client := mcpclient.New(srv.URL)
	results, err := client.SearchDocuments("test query", 5, "test-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Content != "test chunk" {
		t.Errorf("content = %q, want test chunk", results[0].Content)
	}
}

func TestSearchDocumentsMCPError(t *testing.T) {
	step := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch step {
		case 0:
			w.Header().Set("Mcp-Session-Id", "s")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 0,
				"result": map[string]interface{}{"protocolVersion": "2025-06-18", "capabilities": map[string]interface{}{}, "serverInfo": map[string]interface{}{"name": "t", "version": "0.1"}},
			})
		case 1:
			w.WriteHeader(http.StatusAccepted)
		case 2:
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0", "id": 1,
				"result": map[string]interface{}{
					"isError": true,
					"content": []map[string]interface{}{
						{"type": "text", "text": "off_topic: query not related"},
					},
				},
			})
		}
		step++
	}))
	defer srv.Close()

	client := mcpclient.New(srv.URL)
	_, err := client.SearchDocuments("off topic query", 5, "tok")
	if err == nil {
		t.Fatal("expected error for MCP tool error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/backend && go test ./internal/mcpclient/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement client.go**

Create `internal/mcpclient/client.go`:

```go
package mcpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChunkResult struct {
	ID             int     `json:"id"`
	SourcePath     string  `json:"source_path"`
	Title          string  `json:"title"`
	ChunkIndex     int     `json:"chunk_index"`
	HeadingContext string  `json:"heading_context"`
	ChunkType      string  `json:"chunk_type"`
	Content        string  `json:"content"`
	Score          float64 `json:"score"`
}

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// SearchDocuments runs the full MCP handshake (initialize, notify, tools/call)
// and returns the matching chunks.
func (c *Client) SearchDocuments(query string, limit int, bearerToken string) ([]ChunkResult, error) {
	endpoint := c.baseURL + "/mcp"

	// Step 1: initialize.
	initBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      0,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]interface{}{"name": "simple-client", "version": "1.0"},
		},
	}
	initResp, sessionID, err := c.postMCP(endpoint, initBody, bearerToken, "")
	if err != nil {
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}
	_ = initResp

	// Step 2: notifications/initialized.
	notifBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	_, _, err = c.postMCP(endpoint, notifBody, bearerToken, sessionID)
	if err != nil {
		return nil, fmt.Errorf("mcp notification: %w", err)
	}

	// Step 3: tools/call search_documents.
	toolBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "search_documents",
			"arguments": map[string]interface{}{
				"query": query,
				"limit": limit,
			},
		},
	}
	toolResp, _, err := c.postMCP(endpoint, toolBody, bearerToken, sessionID)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/call: %w", err)
	}

	return parseToolResponse(toolResp)
}

func (c *Client) postMCP(url string, body interface{}, token, sessionID string) (json.RawMessage, string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	sid := resp.Header.Get("Mcp-Session-Id")
	if sid == "" {
		sid = sessionID
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, sid, err
	}

	if resp.StatusCode == http.StatusAccepted {
		return nil, sid, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, sid, fmt.Errorf("mcp returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Handle SSE responses (data: prefix).
	raw := string(respBody)
	if strings.HasPrefix(raw, "data:") {
		raw = strings.TrimPrefix(strings.SplitN(raw, "\n", 2)[0], "data: ")
		respBody = []byte(raw)
	}

	return json.RawMessage(respBody), sid, nil
}

func parseToolResponse(raw json.RawMessage) ([]ChunkResult, error) {
	var envelope struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				Results []ChunkResult `json:"results"`
			} `json:"structuredContent"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("parsing MCP response: %w", err)
	}
	if envelope.Result.IsError {
		msg := "unknown MCP error"
		if len(envelope.Result.Content) > 0 {
			msg = envelope.Result.Content[0].Text
		}
		return nil, fmt.Errorf("mcp_error: %s", msg)
	}
	return envelope.Result.StructuredContent.Results, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/backend && go test ./internal/mcpclient/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add simple-client/backend/internal/mcpclient/
git commit -m "feat: add MCP Streamable HTTP client with session handling."
```

---

## Task 5: Claude streaming API client

**Files:**
- Create: `simple-client/backend/internal/llm/claude.go`
- Create: `simple-client/backend/internal/llm/claude_test.go`

- [ ] **Step 1: Write failing Claude client tests**

Create `internal/llm/claude_test.go`:

```go
package llm_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brightsign-playground/simple-client/internal/llm"
	"github.com/brightsign-playground/simple-client/internal/mcpclient"
)

func TestStreamAnswer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-test" {
			t.Errorf("missing api key header")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("not a flusher")
		}
		fmt.Fprintln(w, `event: message_start`)
		fmt.Fprintln(w, `data: {"type":"message_start","message":{"id":"msg_1","role":"assistant"}}`)
		fmt.Fprintln(w)
		flusher.Flush()

		fmt.Fprintln(w, `event: content_block_start`)
		fmt.Fprintln(w, `data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		fmt.Fprintln(w)
		flusher.Flush()

		fmt.Fprintln(w, `event: content_block_delta`)
		fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello "}}`)
		fmt.Fprintln(w)
		flusher.Flush()

		fmt.Fprintln(w, `event: content_block_delta`)
		fmt.Fprintln(w, `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"world"}}`)
		fmt.Fprintln(w)
		flusher.Flush()

		fmt.Fprintln(w, `event: message_stop`)
		fmt.Fprintln(w, `data: {"type":"message_stop"}`)
		fmt.Fprintln(w)
		flusher.Flush()
	}))
	defer srv.Close()

	client := llm.NewClient(srv.URL, "sk-test", "claude-test", 1024, "You are helpful.")
	chunks := []mcpclient.ChunkResult{
		{Content: "test content", SourcePath: "test.md", Score: 0.9},
	}

	var collected string
	err := client.StreamAnswer("what is this?", chunks, func(delta string) {
		collected += delta
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if collected != "Hello world" {
		t.Errorf("collected = %q, want %q", collected, "Hello world")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/backend && go test ./internal/llm/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement claude.go**

Create `internal/llm/claude.go`:

```go
package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/brightsign-playground/simple-client/internal/mcpclient"
)

type Client struct {
	baseURL      string
	apiKey       string
	model        string
	maxTokens    int
	systemPrompt string
	http         *http.Client
}

func NewClient(baseURL, apiKey, model string, maxTokens int, systemPrompt string) *Client {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		apiKey:       apiKey,
		model:        model,
		maxTokens:    maxTokens,
		systemPrompt: systemPrompt,
		http:         &http.Client{Timeout: 120 * time.Second},
	}
}

func (c *Client) StreamAnswer(question string, chunks []mcpclient.ChunkResult, onDelta func(string)) error {
	context := buildContext(chunks)

	reqBody := map[string]interface{}{
		"model":      c.model,
		"max_tokens": c.maxTokens,
		"stream":     true,
		"system":     c.systemPrompt,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "Document excerpts:\n\n" + context + "\nQuestion: " + question,
			},
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("llm: marshaling request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("llm: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("llm: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		return fmt.Errorf("llm: API returned %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			onDelta(event.Delta.Text)
		}
	}
	return scanner.Err()
}

func buildContext(chunks []mcpclient.ChunkResult) string {
	var b strings.Builder
	for i, ch := range chunks {
		fmt.Fprintf(&b, "[%d] %s", i+1, ch.SourcePath)
		if ch.HeadingContext != "" {
			fmt.Fprintf(&b, "\nSection: %s", ch.HeadingContext)
		}
		fmt.Fprintf(&b, "\nType: %s\nScore: %.2f\n\n%s\n\n---\n\n", ch.ChunkType, ch.Score, ch.Content)
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/backend && go test ./internal/llm/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add simple-client/backend/internal/llm/
git commit -m "feat: add Claude streaming API client with SSE parsing."
```

---

## Task 6: POST /api/ask handler with SSE output

**Files:**
- Create: `simple-client/backend/internal/api/ask.go`
- Create: `simple-client/backend/internal/api/ask_test.go`

- [ ] **Step 1: Write failing ask handler tests**

Create `internal/api/ask_test.go`:

```go
package api_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brightsign-playground/simple-client/internal/api"
	"github.com/brightsign-playground/simple-client/internal/mcpclient"
)

type fakeTokenCache struct{}

func (f *fakeTokenCache) Token() (string, error) { return "fake-jwt", nil }

type fakeMCPClient struct {
	results []mcpclient.ChunkResult
	err     error
}

func (f *fakeMCPClient) SearchDocuments(query string, limit int, token string) ([]mcpclient.ChunkResult, error) {
	return f.results, f.err
}

type fakeLLMClient struct {
	answer string
	err    error
}

func (f *fakeLLMClient) StreamAnswer(question string, chunks []mcpclient.ChunkResult, onDelta func(string)) error {
	if f.err != nil {
		return f.err
	}
	for _, word := range strings.Fields(f.answer) {
		onDelta(word + " ")
	}
	return nil
}

func TestAskHandlerSuccess(t *testing.T) {
	handler := api.NewAskHandler(
		&fakeTokenCache{},
		&fakeMCPClient{
			results: []mcpclient.ChunkResult{{ID: 1, Content: "chunk text", Score: 0.8, SourcePath: "a.md"}},
		},
		&fakeLLMClient{answer: "The answer is 42."},
		5,
	)

	body := strings.NewReader(`{"question":"what is the answer?"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}

	// Parse SSE events.
	scanner := bufio.NewScanner(rec.Body)
	var events []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event:") {
			events = append(events, strings.TrimSpace(strings.TrimPrefix(line, "event:")))
		}
	}
	if len(events) < 3 {
		t.Fatalf("got %d events, want >= 3 (chunks, delta(s), done)", len(events))
	}
	if events[0] != "chunks" {
		t.Errorf("first event = %q, want chunks", events[0])
	}
	if events[len(events)-1] != "done" {
		t.Errorf("last event = %q, want done", events[len(events)-1])
	}
}

func TestAskHandlerEmptyQuestion(t *testing.T) {
	handler := api.NewAskHandler(&fakeTokenCache{}, &fakeMCPClient{}, &fakeLLMClient{}, 5)
	body := strings.NewReader(`{"question":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Should get an error SSE event.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (errors come via SSE)", rec.Code)
	}

	var foundError bool
	scanner := bufio.NewScanner(rec.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "invalid_request") {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected invalid_request error event for empty question")
	}
}

func TestAskHandlerQuestionTooLong(t *testing.T) {
	handler := api.NewAskHandler(&fakeTokenCache{}, &fakeMCPClient{}, &fakeLLMClient{}, 5)
	longQ := strings.Repeat("a", 1001)
	body := strings.NewReader(`{"question":"` + longQ + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var foundError bool
	scanner := bufio.NewScanner(rec.Body)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "invalid_request") {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected invalid_request error for question > 1000 chars")
	}
}

func TestAskHandlerNoResults(t *testing.T) {
	handler := api.NewAskHandler(
		&fakeTokenCache{},
		&fakeMCPClient{results: []mcpclient.ChunkResult{}},
		&fakeLLMClient{},
		5,
	)
	body := strings.NewReader(`{"question":"anything"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var found bool
	scanner := bufio.NewScanner(rec.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "no_results") {
			found = true
		}
	}
	if !found {
		t.Error("expected no_results error when MCP returns empty")
	}
}

// Verify that rate data fields appear in the SSE "done" event.
func TestAskHandlerDoneEventContainsFullAnswer(t *testing.T) {
	handler := api.NewAskHandler(
		&fakeTokenCache{},
		&fakeMCPClient{results: []mcpclient.ChunkResult{{ID: 1, Content: "c", Score: 0.5}}},
		&fakeLLMClient{answer: "full answer here"},
		5,
	)
	body := strings.NewReader(`{"question":"q"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/ask", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var doneData string
	scanner := bufio.NewScanner(rec.Body)
	nextIsDone := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "event: done" {
			nextIsDone = true
			continue
		}
		if nextIsDone && strings.HasPrefix(line, "data:") {
			doneData = strings.TrimPrefix(line, "data: ")
			break
		}
	}
	var done struct {
		Answer string `json:"answer"`
	}
	json.Unmarshal([]byte(doneData), &done)
	if !strings.Contains(done.Answer, "full") {
		t.Errorf("done.answer = %q, expected it to contain full answer", done.Answer)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/backend && go test ./internal/api/... -v
```

Expected: compilation error.

- [ ] **Step 3: Implement ask.go**

Create `internal/api/ask.go`. This handler:
- Accepts JSON `{question, limit}`.
- Sets `Content-Type: text/event-stream`.
- Calls token cache, MCP client, LLM client in sequence.
- Writes SSE events: `chunks`, `delta` (per LLM fragment), `done`, or `error`.

```go
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brightsign-playground/simple-client/internal/mcpclient"
)

// TokenProvider abstracts token acquisition.
type TokenProvider interface {
	Token() (string, error)
}

// MCPSearcher abstracts MCP document search.
type MCPSearcher interface {
	SearchDocuments(query string, limit int, token string) ([]mcpclient.ChunkResult, error)
}

// LLMStreamer abstracts streaming LLM answer generation.
type LLMStreamer interface {
	StreamAnswer(question string, chunks []mcpclient.ChunkResult, onDelta func(string)) error
}

type askRequest struct {
	Question string `json:"question"`
	Limit    int    `json:"limit"`
}

type AskHandler struct {
	tokens       TokenProvider
	mcp          MCPSearcher
	llm          LLMStreamer
	defaultLimit int
}

func NewAskHandler(tokens TokenProvider, mcp MCPSearcher, llm LLMStreamer, defaultLimit int) *AskHandler {
	return &AskHandler{tokens: tokens, mcp: mcp, llm: llm, defaultLimit: defaultLimit}
}

func (h *AskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeSSEError(w, nil, "invalid_request", "streaming not supported")
		return
	}

	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeSSEError(w, flusher, "invalid_request", "invalid JSON body")
		return
	}

	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		writeSSEError(w, flusher, "invalid_request", "question is required")
		return
	}
	if len(req.Question) > 1000 {
		writeSSEError(w, flusher, "invalid_request", "question exceeds 1000 characters")
		return
	}
	if req.Limit < 1 || req.Limit > 20 {
		req.Limit = h.defaultLimit
	}

	// Get OIDC token.
	token, err := h.tokens.Token()
	if err != nil {
		slog.Error("token acquisition failed", "error", err)
		writeSSEError(w, flusher, "auth_error", "failed to obtain authentication token")
		return
	}

	// Search MCP.
	chunks, err := h.mcp.SearchDocuments(req.Question, req.Limit, token)
	if err != nil {
		slog.Error("MCP search failed", "error", err)
		writeSSEError(w, flusher, "mcp_error", err.Error())
		return
	}
	if len(chunks) == 0 {
		writeSSEError(w, flusher, "no_results", "no matching documents found")
		return
	}

	// Send chunks event.
	chunksJSON, _ := json.Marshal(chunks)
	writeSSEEvent(w, flusher, "chunks", string(chunksJSON))

	// Stream LLM answer.
	var fullAnswer strings.Builder
	err = h.llm.StreamAnswer(req.Question, chunks, func(delta string) {
		fullAnswer.WriteString(delta)
		deltaJSON, _ := json.Marshal(map[string]string{"text": delta})
		writeSSEEvent(w, flusher, "delta", string(deltaJSON))
	})
	if err != nil {
		slog.Error("LLM streaming failed", "error", err)
		writeSSEError(w, flusher, "llm_error", err.Error())
		return
	}

	// Send done event.
	doneJSON, _ := json.Marshal(map[string]string{"answer": fullAnswer.String()})
	writeSSEEvent(w, flusher, "done", string(doneJSON))
}

func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	if flusher != nil {
		flusher.Flush()
	}
}

func writeSSEError(w http.ResponseWriter, flusher http.Flusher, code, message string) {
	data, _ := json.Marshal(map[string]string{"code": code, "message": message})
	writeSSEEvent(w, flusher, "error", string(data))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/backend && go test ./internal/api/... -v
```

Expected: all 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add simple-client/backend/internal/api/
git commit -m "feat: add POST /api/ask SSE handler orchestrating auth, MCP, and LLM."
```

---

## Task 7: HTTP server assembly and main.go

**Files:**
- Create: `simple-client/backend/internal/server/server.go`
- Create: `simple-client/backend/cmd/server/main.go`

- [ ] **Step 1: Create server.go**

```go
package server

import (
	"context"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

type Config struct {
	Addr      string
	StaticDir string
	LogoPath  string
	CSSPath   string
}

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(cfg Config, askHandler http.Handler, logger *slog.Logger) *Server {
	mux := http.NewServeMux()

	// API routes.
	mux.Handle("POST /api/ask", askHandler)

	// Branding routes.
	if cfg.LogoPath != "" {
		mux.HandleFunc("GET /logo", func(w http.ResponseWriter, r *http.Request) {
			ext := filepath.Ext(cfg.LogoPath)
			ct := mime.TypeByExtension(ext)
			if ct != "" {
				w.Header().Set("Content-Type", ct)
			}
			http.ServeFile(w, r, cfg.LogoPath)
		})
	}
	if cfg.CSSPath != "" {
		mux.HandleFunc("GET /custom.css", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/css")
			http.ServeFile(w, r, cfg.CSSPath)
		})
	}

	// Health check.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Static files (fallback).
	if cfg.StaticDir != "" {
		fs := http.FileServer(http.Dir(cfg.StaticDir))
		mux.Handle("/", fs)
	}

	return &Server{
		httpServer: &http.Server{Addr: cfg.Addr, Handler: mux},
		logger:     logger,
	}
}

func (s *Server) Start() error {
	s.logger.Info("backend listening", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
```

- [ ] **Step 2: Create main.go**

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brightsign-playground/simple-client/internal/api"
	"github.com/brightsign-playground/simple-client/internal/auth"
	"github.com/brightsign-playground/simple-client/internal/config"
	"github.com/brightsign-playground/simple-client/internal/llm"
	"github.com/brightsign-playground/simple-client/internal/mcpclient"
	"github.com/brightsign-playground/simple-client/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	configPath := os.Getenv("SIMPLE_CLIENT_CONFIG")
	if configPath == "" {
		configPath = "config.toml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Build OIDC token URL from provider config.
	var tokenURL, clientID, clientSecret, audience string
	switch cfg.Auth.Provider {
	case "keycloak":
		tokenURL = cfg.Auth.KeycloakIssuer + "/protocol/openid-connect/token"
		clientID = cfg.Auth.KeycloakClientID
		clientSecret = cfg.Auth.KeycloakClientSecret
	case "logto":
		tokenURL = cfg.Auth.LogtoIssuer + "/oidc/token"
		clientID = cfg.Auth.LogtoAppID
		clientSecret = cfg.Auth.LogtoAppSecret
		audience = cfg.Auth.LogtoAudience
	}

	tokens := auth.NewTokenCache(tokenURL, clientID, clientSecret, audience)
	mcp := mcpclient.New(cfg.MCP.ServerURL)
	llmClient := llm.NewClient("", cfg.LLM.AnthropicAPIKey, cfg.LLM.Model, cfg.LLM.MaxTokens, cfg.LLM.SystemPrompt)

	askHandler := api.NewAskHandler(tokens, mcp, llmClient, cfg.MCP.Limit)

	// Backend listens on backend_port (Caddy proxies from the public listen port).
	srv := server.New(server.Config{
		Addr:      cfg.Server.BackendPort,
		StaticDir: "", // Caddy serves static files; backend does not.
		LogoPath:  cfg.Branding.Logo,
		CSSPath:   cfg.Branding.CustomCSS,
	}, askHandler, logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
}
```

- [ ] **Step 3: Verify it compiles**

```bash
cd simple-client/backend && go build ./cmd/server
```

Expected: compiles without error.

- [ ] **Step 4: Commit**

```bash
git add simple-client/backend/internal/server/ simple-client/backend/cmd/server/
git commit -m "feat: add HTTP server assembly and main.go entry point."
```

---

## Task 8: Frontend scaffold and TypeScript types

**Files:**
- Create: `simple-client/frontend/package.json`
- Create: `simple-client/frontend/tsconfig.json`
- Create: `simple-client/frontend/vite.config.ts`
- Create: `simple-client/frontend/index.html`
- Create: `simple-client/frontend/src/types.ts`

- [ ] **Step 1: Create package.json**

```json
{
  "name": "simple-client-frontend",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "test": "vitest run"
  },
  "devDependencies": {
    "typescript": "^5.7.0",
    "vite": "^6.0.0",
    "vitest": "^3.0.0",
    "jsdom": "^25.0.0",
    "@types/dompurify": "^3.2.0"
  },
  "dependencies": {
    "marked": "^15.0.0",
    "dompurify": "^3.2.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "outDir": "./dist",
    "rootDir": "./src",
    "types": ["vitest/globals"]
  },
  "include": ["src"]
}
```

- [ ] **Step 3: Create vite.config.ts**

```typescript
import { defineConfig } from "vite";

export default defineConfig({
  root: ".",
  build: {
    outDir: "../dist",
    emptyOutDir: true,
  },
  server: {
    proxy: {
      "/api": "http://localhost:8091",
      "/logo": "http://localhost:8091",
      "/custom.css": "http://localhost:8091",
    },
  },
  test: {
    environment: "jsdom",
  },
});
```

- [ ] **Step 4: Create index.html**

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Support Search</title>
  <link rel="stylesheet" href="/src/styles.css" />
  <link rel="stylesheet" href="/custom.css" onerror="this.remove()" />
</head>
<body>
  <header class="header">
    <img id="logo" src="/logo" alt="" class="header-logo" onerror="this.style.display='none'" />
    <h1 id="app-title" class="header-title">Support Search</h1>
  </header>

  <div class="layout">
    <aside class="history-panel" id="history-panel">
      <button class="history-btn" id="btn-prev" title="Previous (Left Arrow)" disabled>&lt;</button>
      <span class="history-counter" id="history-counter">0 / 0</span>
      <button class="history-btn" id="btn-next" title="Next (Right Arrow)" disabled>&gt;</button>
      <input type="range" id="history-slider" class="history-slider" min="0" max="0" value="0" disabled />
      <button class="history-btn history-btn-new" id="btn-new" title="New question (Ctrl+N)">+</button>
    </aside>

    <main class="main-content">
      <section class="question-section">
        <label for="question-input" class="section-label">Question</label>
        <textarea
          id="question-input"
          class="text-box question-box"
          placeholder="Ask a question..."
          maxlength="1000"
          rows="3"
        ></textarea>
        <span class="char-counter" id="char-counter">0 / 1000</span>
      </section>

      <section class="answer-section">
        <label class="section-label">Answer</label>
        <div id="answer-output" class="text-box answer-box">
          <p class="placeholder-text">Ask a question to see the answer here.</p>
        </div>
        <div id="streaming-indicator" class="streaming-indicator" hidden>
          <span class="dot"></span> Generating answer...
        </div>
      </section>

      <details id="sources-section" class="sources-section" hidden>
        <summary>Sources (<span id="source-count">0</span>)</summary>
        <div id="sources-list" class="sources-list"></div>
      </details>

      <button id="btn-ask" class="ask-btn" disabled>Ask</button>
    </main>
  </div>

  <script type="module" src="/src/main.ts"></script>
</body>
</html>
```

- [ ] **Step 5: Create types.ts**

```typescript
export interface ChunkResult {
  id: number;
  source_path: string;
  title: string;
  chunk_index: number;
  heading_context: string;
  chunk_type: string;
  content: string;
  score: number;
}

export interface QAPair {
  id: number;
  question: string;
  answer: string;
  chunks: ChunkResult[];
  timestamp: string; // ISO 8601
  status: "streaming" | "complete" | "error";
  error?: string;
}

export interface AskRequest {
  question: string;
  limit?: number;
}

export interface SSEDeltaEvent {
  text: string;
}

export interface SSEDoneEvent {
  answer: string;
}

export interface SSEErrorEvent {
  code: string;
  message: string;
}
```

- [ ] **Step 6: Install dependencies**

```bash
cd simple-client/frontend && npm install
```

- [ ] **Step 7: Commit**

```bash
git add simple-client/frontend/package.json simple-client/frontend/package-lock.json simple-client/frontend/tsconfig.json simple-client/frontend/vite.config.ts simple-client/frontend/index.html simple-client/frontend/src/types.ts
git commit -m "scaffold: add frontend project with Vite, TypeScript, HTML shell, and type definitions."
```

---

## Task 9: History store with sessionStorage persistence

**Files:**
- Create: `simple-client/frontend/src/history.ts`
- Create: `simple-client/frontend/src/__tests__/history.test.ts`

- [ ] **Step 1: Write failing history tests**

Create `src/__tests__/history.test.ts`:

```typescript
import { describe, it, expect, beforeEach } from "vitest";
import { HistoryStore } from "../history";

beforeEach(() => {
  sessionStorage.clear();
});

describe("HistoryStore", () => {
  it("starts empty", () => {
    const store = new HistoryStore();
    expect(store.count()).toBe(0);
    expect(store.current()).toBeNull();
  });

  it("adds a pair and navigates to it", () => {
    const store = new HistoryStore();
    store.startNew("test question");
    expect(store.count()).toBe(1);
    const pair = store.current();
    expect(pair?.question).toBe("test question");
    expect(pair?.status).toBe("streaming");
  });

  it("completes a pair", () => {
    const store = new HistoryStore();
    store.startNew("q");
    store.updateAnswer("partial ");
    store.updateAnswer("answer");
    store.complete();
    expect(store.current()?.answer).toBe("partial answer");
    expect(store.current()?.status).toBe("complete");
  });

  it("navigates forward and back", () => {
    const store = new HistoryStore();
    store.startNew("q1");
    store.complete();
    store.startNew("q2");
    store.complete();
    expect(store.currentIndex()).toBe(1);
    store.goTo(0);
    expect(store.current()?.question).toBe("q1");
    store.goTo(1);
    expect(store.current()?.question).toBe("q2");
  });

  it("persists to sessionStorage", () => {
    const store = new HistoryStore();
    store.startNew("persisted");
    store.complete();

    const store2 = new HistoryStore();
    expect(store2.count()).toBe(1);
    expect(store2.current()?.question).toBe("persisted");
  });

  it("limits to 100 pairs", () => {
    const store = new HistoryStore();
    for (let i = 0; i < 105; i++) {
      store.startNew(`q${i}`);
      store.complete();
    }
    expect(store.count()).toBe(100);
    expect(store.pairs()[0].question).toBe("q5");
  });

  it("marks error state", () => {
    const store = new HistoryStore();
    store.startNew("q");
    store.setError("mcp_error: timeout");
    expect(store.current()?.status).toBe("error");
    expect(store.current()?.error).toBe("mcp_error: timeout");
  });

  it("goToNew moves past last pair", () => {
    const store = new HistoryStore();
    store.startNew("q1");
    store.complete();
    store.goToNew();
    expect(store.current()).toBeNull();
    expect(store.isAtNew()).toBe(true);
  });

  it("sets chunks on current pair", () => {
    const store = new HistoryStore();
    store.startNew("q");
    store.setChunks([{ id: 1, content: "c", score: 0.8, source_path: "a.md", title: "A", chunk_index: 0, heading_context: "", chunk_type: "paragraph" }]);
    expect(store.current()?.chunks.length).toBe(1);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/frontend && npx vitest run
```

Expected: error -- `history.ts` does not exist.

- [ ] **Step 3: Implement history.ts**

```typescript
import type { QAPair, ChunkResult } from "./types";

const STORAGE_KEY = "simple-client-history";
const MAX_PAIRS = 100;

export class HistoryStore {
  private _pairs: QAPair[];
  private _index: number;

  constructor() {
    const saved = sessionStorage.getItem(STORAGE_KEY);
    if (saved) {
      try {
        this._pairs = JSON.parse(saved);
      } catch {
        this._pairs = [];
      }
    } else {
      this._pairs = [];
    }
    this._index = this._pairs.length > 0 ? this._pairs.length - 1 : -1;
  }

  count(): number {
    return this._pairs.length;
  }

  pairs(): QAPair[] {
    return this._pairs;
  }

  currentIndex(): number {
    return this._index;
  }

  current(): QAPair | null {
    if (this._index < 0 || this._index >= this._pairs.length) return null;
    return this._pairs[this._index];
  }

  startNew(question: string): QAPair {
    const pair: QAPair = {
      id: this._pairs.length + 1,
      question,
      answer: "",
      chunks: [],
      timestamp: new Date().toISOString(),
      status: "streaming",
    };
    this._pairs.push(pair);
    if (this._pairs.length > MAX_PAIRS) {
      this._pairs.splice(0, this._pairs.length - MAX_PAIRS);
    }
    this._index = this._pairs.length - 1;
    this.persist();
    return pair;
  }

  updateAnswer(delta: string): void {
    const pair = this.current();
    if (pair && pair.status === "streaming") {
      pair.answer += delta;
      this.persist();
    }
  }

  setChunks(chunks: ChunkResult[]): void {
    const pair = this.current();
    if (pair) {
      pair.chunks = chunks;
      this.persist();
    }
  }

  complete(): void {
    const pair = this.current();
    if (pair) {
      pair.status = "complete";
      this.persist();
    }
  }

  setError(message: string): void {
    const pair = this.current();
    if (pair) {
      pair.status = "error";
      pair.error = message;
      this.persist();
    }
  }

  goTo(index: number): void {
    if (index >= 0 && index < this._pairs.length) {
      this._index = index;
    }
  }

  // Move past the last pair to represent "new empty question" state.
  // current() returns null in this position.
  goToNew(): void {
    this._index = this._pairs.length;
  }

  isAtLatest(): boolean {
    return this._index >= this._pairs.length - 1;
  }

  isAtNew(): boolean {
    return this._index >= this._pairs.length;
  }

  private persist(): void {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(this._pairs));
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/frontend && npx vitest run
```

Expected: all 8 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add simple-client/frontend/src/history.ts simple-client/frontend/src/__tests__/history.test.ts
git commit -m "feat: add QAPair history store with sessionStorage persistence and navigation."
```

---

## Task 10: Markdown renderer with XSS sanitization

**Files:**
- Create: `simple-client/frontend/src/markdown.ts`
- Create: `simple-client/frontend/src/__tests__/markdown.test.ts`

- [ ] **Step 1: Write failing markdown tests**

Create `src/__tests__/markdown.test.ts`:

```typescript
import { describe, it, expect } from "vitest";
import { renderMarkdown } from "../markdown";

describe("renderMarkdown", () => {
  it("renders basic markdown", () => {
    const html = renderMarkdown("**bold** and *italic*");
    expect(html).toContain("<strong>bold</strong>");
    expect(html).toContain("<em>italic</em>");
  });

  it("renders code blocks", () => {
    const html = renderMarkdown("```js\nconsole.log('hi')\n```");
    expect(html).toContain("<code>");
    expect(html).toContain("console.log");
  });

  it("strips script tags (XSS)", () => {
    const html = renderMarkdown('<script>alert("xss")</script>');
    expect(html).not.toContain("<script>");
  });

  it("strips event handlers (XSS)", () => {
    const html = renderMarkdown('<img src=x onerror="alert(1)">');
    expect(html).not.toContain("onerror");
  });

  it("renders links", () => {
    const html = renderMarkdown("[click](https://example.com)");
    expect(html).toContain('href="https://example.com"');
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd simple-client/frontend && npx vitest run
```

Expected: error -- `markdown.ts` does not exist.

- [ ] **Step 3: Implement markdown.ts**

```typescript
import { marked } from "marked";
import DOMPurify from "dompurify";

// Configure marked for safe defaults.
marked.setOptions({
  breaks: true,
  gfm: true,
});

export function renderMarkdown(text: string): string {
  const raw = marked.parse(text) as string;
  return DOMPurify.sanitize(raw, {
    ALLOWED_TAGS: [
      "p", "br", "strong", "em", "code", "pre", "blockquote",
      "ul", "ol", "li", "a", "h1", "h2", "h3", "h4", "h5", "h6",
      "table", "thead", "tbody", "tr", "th", "td", "hr", "del",
    ],
    ALLOWED_ATTR: ["href", "title", "target", "rel"],
  });
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd simple-client/frontend && npx vitest run
```

Expected: all 5 markdown tests PASS (plus the 8 history tests).

- [ ] **Step 5: Commit**

```bash
git add simple-client/frontend/src/markdown.ts simple-client/frontend/src/__tests__/markdown.test.ts
git commit -m "feat: add Markdown renderer with DOMPurify XSS sanitization."
```

---

## Task 11: SSE API client

**Files:**
- Create: `simple-client/frontend/src/api.ts`

- [ ] **Step 1: Implement api.ts**

```typescript
import type { ChunkResult, SSEDeltaEvent, SSEDoneEvent, SSEErrorEvent } from "./types";

export interface AskCallbacks {
  onChunks: (chunks: ChunkResult[]) => void;
  onDelta: (text: string) => void;
  onDone: (answer: string) => void;
  onError: (code: string, message: string) => void;
}

export async function ask(question: string, limit: number | undefined, callbacks: AskCallbacks): Promise<void> {
  const body = JSON.stringify({ question, ...(limit !== undefined && { limit }) });

  let response: Response;
  try {
    response = await fetch("/api/ask", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
    });
  } catch {
    callbacks.onError("network_error", "Connection error: unable to reach the server.");
    return;
  }

  if (!response.ok || !response.body) {
    callbacks.onError("network_error", `Server returned ${response.status}`);
    return;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      let currentEvent = "";
      for (const line of lines) {
        if (line.startsWith("event: ")) {
          currentEvent = line.slice(7).trim();
        } else if (line.startsWith("data: ") && currentEvent) {
          const data = line.slice(6);
          try {
            switch (currentEvent) {
              case "chunks":
                callbacks.onChunks(JSON.parse(data) as ChunkResult[]);
                break;
              case "delta": {
                const evt = JSON.parse(data) as SSEDeltaEvent;
                callbacks.onDelta(evt.text);
                break;
              }
              case "done": {
                const evt = JSON.parse(data) as SSEDoneEvent;
                callbacks.onDone(evt.answer);
                break;
              }
              case "error": {
                const evt = JSON.parse(data) as SSEErrorEvent;
                callbacks.onError(evt.code, evt.message);
                return;
              }
            }
          } catch {
            // Skip malformed JSON lines.
          }
          currentEvent = "";
        } else if (line === "") {
          currentEvent = "";
        }
      }
    }
  } catch {
    callbacks.onError("network_error", "Stream interrupted. The answer may be incomplete.");
  }
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd simple-client/frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add simple-client/frontend/src/api.ts
git commit -m "feat: add SSE API client for POST /api/ask with streaming event parsing."
```

---

## Task 12: Default CSS theme

**Files:**
- Create: `simple-client/frontend/src/styles.css`

- [ ] **Step 1: Create styles.css**

Write the full CSS file implementing the layout from REQUIREMENTS.md:

- CSS custom properties (`:root` variables) for all colors, fonts, spacing.
- `.header` with logo + title.
- `.layout` as a CSS grid: `aside.history-panel` (60px) + `main.main-content`.
- `.history-panel` with vertical centering, buttons, range slider, counter.
- `.text-box` base class with `overflow-y: auto`, border, border-radius.
- `.question-box` -- white background, editable textarea.
- `.answer-box` -- surface background, read-only div.
- `.char-counter` positioned below the question box.
- `.ask-btn` with primary color, disabled state.
- `.streaming-indicator` with pulsing dot animation.
- `.sources-section` as a collapsible `<details>`.
- `@media (max-width: 768px)` -- collapse history panel to an icon bar.
- `.placeholder-text` -- muted italic text for empty answer box.

Refer to REQUIREMENTS.md "Branding > Default theme" and "Layout" sections for exact values. Use the CSS custom properties defined in the spec:

```css
:root {
    --color-primary: #2563eb;
    --color-background: #ffffff;
    --color-surface: #f9fafb;
    --color-text: #111827;
    --color-text-muted: #6b7280;
    --color-border: #e5e7eb;
    --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    --font-family-mono: "SF Mono", "Fira Code", "Fira Mono", monospace;
    --border-radius: 8px;
    --spacing-unit: 8px;
}
```

Full CSS is too long for inline listing; the implementer should write it following the layout wireframe and these variables. Keep the file under 300 lines.

- [ ] **Step 2: Verify the page renders in a browser**

```bash
cd simple-client/frontend && npx vite --open
```

Open http://localhost:5173 and verify the layout matches the wireframe from REQUIREMENTS.md. The page will show the HTML shell with styled empty boxes. No functionality yet.

- [ ] **Step 3: Commit**

```bash
git add simple-client/frontend/src/styles.css
git commit -m "feat: add default CSS theme with custom properties for branding."
```

---

## Task 13: UI wiring and main.ts

**Files:**
- Create: `simple-client/frontend/src/ui.ts`
- Create: `simple-client/frontend/src/main.ts`

- [ ] **Step 1: Create ui.ts**

This module owns all DOM manipulation. It exports functions (not classes) that read/write DOM elements by ID. Key functions:

```typescript
// ui.ts — DOM manipulation functions

import { renderMarkdown } from "./markdown";
import type { QAPair, ChunkResult } from "./types";

// Get all DOM elements by ID once.
const $ = (id: string) => document.getElementById(id)!;

export function renderPair(pair: QAPair | null): void {
  const questionInput = $("question-input") as HTMLTextAreaElement;
  const answerOutput = $("answer-output");
  const sourcesSection = $("sources-section") as HTMLDetailsElement;
  const sourcesList = $("sources-list");
  const sourceCount = $("source-count");
  const streamingIndicator = $("streaming-indicator");

  if (!pair) {
    questionInput.value = "";
    questionInput.readOnly = false;
    answerOutput.innerHTML = '<p class="placeholder-text">Ask a question to see the answer here.</p>';
    sourcesSection.hidden = true;
    streamingIndicator.hidden = true;
    updateCharCounter(0);
    return;
  }

  questionInput.value = pair.question;
  questionInput.readOnly = pair.status !== "streaming" || true; // historical pairs always read-only; handled by caller

  if (pair.status === "error") {
    answerOutput.innerHTML = `<p class="error-text">${escapeHtml(pair.error || "Unknown error")}</p>`;
  } else if (pair.answer) {
    answerOutput.innerHTML = renderMarkdown(pair.answer);
  } else {
    answerOutput.innerHTML = '<p class="placeholder-text">Waiting for answer...</p>';
  }

  streamingIndicator.hidden = pair.status !== "streaming";

  if (pair.chunks.length > 0) {
    sourcesSection.hidden = false;
    sourceCount.textContent = String(pair.chunks.length);
    sourcesList.innerHTML = pair.chunks.map((ch, i) => renderSourceItem(ch, i)).join("");
  } else {
    sourcesSection.hidden = true;
  }
}

function renderSourceItem(chunk: ChunkResult, index: number): string {
  const heading = chunk.heading_context ? ` > ${escapeHtml(chunk.heading_context)}` : "";
  const preview = escapeHtml(chunk.content.slice(0, 150)) + (chunk.content.length > 150 ? "..." : "");
  return `
    <details class="source-item">
      <summary>[${index + 1}] ${escapeHtml(chunk.source_path)}${heading} <span class="score">(${chunk.score.toFixed(2)})</span></summary>
      <pre class="source-content">${escapeHtml(chunk.content)}</pre>
    </details>`;
}

export function appendDelta(text: string): void {
  const answerOutput = $("answer-output");
  // If there is a placeholder, clear it first.
  if (answerOutput.querySelector(".placeholder-text")) {
    answerOutput.innerHTML = "";
  }
  // Append raw text; final render happens on done.
  const current = answerOutput.getAttribute("data-raw") || "";
  const updated = current + text;
  answerOutput.setAttribute("data-raw", updated);
  answerOutput.innerHTML = renderMarkdown(updated);
  answerOutput.scrollTop = answerOutput.scrollHeight;
}

export function updateHistoryControls(index: number, total: number, isStreaming: boolean): void {
  const counter = $("history-counter");
  const slider = $("history-slider") as HTMLInputElement;
  const prevBtn = $("btn-prev") as HTMLButtonElement;
  const nextBtn = $("btn-next") as HTMLButtonElement;
  const newBtn = $("btn-new") as HTMLButtonElement;

  counter.textContent = total === 0 ? "0 / 0" : `${index + 1} / ${total}`;
  slider.max = String(Math.max(0, total - 1));
  slider.value = String(index);
  slider.disabled = total <= 1 || isStreaming;
  prevBtn.disabled = index <= 0 || isStreaming;
  nextBtn.disabled = index >= total - 1 || isStreaming;
  newBtn.disabled = isStreaming;
}

export function setAskEnabled(enabled: boolean): void {
  ($("btn-ask") as HTMLButtonElement).disabled = !enabled;
}

export function setQuestionEditable(editable: boolean): void {
  ($("question-input") as HTMLTextAreaElement).readOnly = !editable;
}

export function updateCharCounter(length: number): void {
  const counter = $("char-counter");
  counter.textContent = `${length} / 1000`;
  counter.classList.toggle("over-limit", length > 1000);
}

export function getQuestion(): string {
  return ($("question-input") as HTMLTextAreaElement).value.trim();
}

function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}
```

- [ ] **Step 2: Create main.ts**

```typescript
// main.ts — entry point: wire event handlers, keyboard shortcuts, initial render

import { HistoryStore } from "./history";
import { ask } from "./api";
import * as ui from "./ui";

const store = new HistoryStore();
let isStreaming = false;

function render(): void {
  const pair = store.current();
  const isLatest = store.isAtLatest();
  ui.renderPair(pair);
  ui.updateHistoryControls(store.currentIndex(), store.count(), isStreaming);
  ui.setQuestionEditable(!isStreaming && (store.count() === 0 || store.isAtNew() || isLatest));
  ui.setAskEnabled(!isStreaming && ui.getQuestion().length > 0 && ui.getQuestion().length <= 1000);
}

async function submitQuestion(): Promise<void> {
  const question = ui.getQuestion();
  if (!question || question.length > 1000 || isStreaming) return;

  isStreaming = true;
  store.startNew(question);
  render();

  // Clear the raw markdown accumulator.
  document.getElementById("answer-output")!.setAttribute("data-raw", "");

  await ask(question, undefined, {
    onChunks(chunks) {
      store.setChunks(chunks);
      // Do not re-render full pair; sources will show on completion.
    },
    onDelta(text) {
      store.updateAnswer(text);
      ui.appendDelta(text);
    },
    onDone() {
      store.complete();
      isStreaming = false;
      render();
    },
    onError(code, message) {
      store.setError(`${code}: ${message}`);
      isStreaming = false;
      render();
    },
  });
}

// Event listeners.
document.getElementById("btn-ask")!.addEventListener("click", submitQuestion);

document.getElementById("question-input")!.addEventListener("input", () => {
  const len = ui.getQuestion().length;
  ui.updateCharCounter(len);
  ui.setAskEnabled(!isStreaming && len > 0 && len <= 1000);
});

document.getElementById("question-input")!.addEventListener("keydown", (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
    e.preventDefault();
    submitQuestion();
  }
});

document.getElementById("btn-prev")!.addEventListener("click", () => {
  if (isStreaming) return;
  store.goTo(store.currentIndex() - 1);
  render();
});

document.getElementById("btn-next")!.addEventListener("click", () => {
  if (isStreaming) return;
  store.goTo(store.currentIndex() + 1);
  render();
});

document.getElementById("btn-new")!.addEventListener("click", () => {
  if (isStreaming) return;
  // Move past the last pair to show empty editable state.
  store.goToNew();
  render();
  (document.getElementById("question-input") as HTMLTextAreaElement).focus();
});

(document.getElementById("history-slider") as HTMLInputElement).addEventListener("input", (e) => {
  if (isStreaming) return;
  store.goTo(Number((e.target as HTMLInputElement).value));
  render();
});

document.addEventListener("keydown", (e) => {
  if (isStreaming) return;
  const active = document.activeElement;
  const isInTextarea = active?.tagName === "TEXTAREA";
  if (!isInTextarea && e.key === "ArrowLeft") {
    store.goTo(store.currentIndex() - 1);
    render();
  }
  if (!isInTextarea && e.key === "ArrowRight") {
    store.goTo(store.currentIndex() + 1);
    render();
  }
  if ((e.ctrlKey || e.metaKey) && e.key === "n") {
    e.preventDefault();
    document.getElementById("btn-new")!.click();
  }
});

// Initial render.
render();
```

- [ ] **Step 3: Verify it compiles**

```bash
cd simple-client/frontend && npx tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add simple-client/frontend/src/ui.ts simple-client/frontend/src/main.ts
git commit -m "feat: add UI module and main.ts wiring event handlers and keyboard shortcuts."
```

---

## Task 14: End-to-end build and smoke test

**Files:**
- Modify: `simple-client/Makefile` (already created in Task 1; verify it works)

- [ ] **Step 1: Build frontend**

```bash
cd simple-client && make build-frontend
```

Expected: `dist/` directory created with bundled HTML, JS, CSS.

- [ ] **Step 2: Build backend**

```bash
cd simple-client && make build-backend
```

Expected: `bin/server` binary created.

- [ ] **Step 3: Run all tests**

```bash
cd simple-client && make test
```

Expected: all frontend tests (history + markdown) and backend tests (config, auth, mcpclient, llm, api) PASS.

- [ ] **Step 4: Start the stack and smoke test**

This requires the MCP stack running (`make up` from repo root). If available:

```bash
# Terminal 1: start backend
cd simple-client && cp config.toml.example config.toml
# Edit config.toml with real credentials
cd simple-client/backend && go run ./cmd/server

# Terminal 2: start Caddy
cd simple-client && caddy run --config Caddyfile

# Terminal 3: open browser
open http://localhost:8090
```

Verify:
- Page loads with header, question box, answer box, history panel.
- Typing a question enables the Ask button.
- Character counter updates.
- Pressing Ask streams an answer (requires running MCP stack + Claude API key).

- [ ] **Step 5: Commit any fixes from smoke test**

```bash
git add -u simple-client/
git commit -m "fix: address smoke test findings."
```

---

## Task 15: Backend go.sum tidy and final cleanup

**Files:**
- Modify: `simple-client/backend/go.sum`

- [ ] **Step 1: Tidy Go dependencies**

```bash
cd simple-client/backend && go mod tidy
```

- [ ] **Step 2: Run full test suite one more time**

```bash
cd simple-client && make test
```

Expected: all tests PASS.

- [ ] **Step 3: Verify .gitignore is correct**

```bash
cd simple-client && cat .gitignore
```

Verify `config.toml`, `.env`, `.envrc`, `dist/`, `bin/`, `node_modules/` are all listed.

- [ ] **Step 4: Final commit**

```bash
git add simple-client/
git commit -m "chore: tidy dependencies and finalize simple-client project."
```
