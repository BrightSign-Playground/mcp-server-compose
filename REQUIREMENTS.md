# Orchestration System Requirements

## Overview

This document specifies a container orchestration system for the MCP server stack. The system enables a developer to start and stop the entire suite of components — or any subset thereof — using a single master configuration file and a small CLI tool. It is designed for three use cases: local development, integration testing, and building RAG datasets.

The system must be agnostic between Docker and Podman and must not assume either is present exclusively.

---

## Components

The stack consists of the following components. Each maps to one subdirectory under `/mcp-servers/`.

### Always-on (core)

| Component | Directory | Purpose |
|---|---|---|
| `rag-mcp-server` | `rag-mcp-server/` | MCP server providing semantic search over the document corpus. Exposes port `15080`. Depends on PostgreSQL and llama-server. |
| `docs2vector` | `docs2vector/` | One-shot CLI tool that ingests documents into PostgreSQL via llama-server embeddings. Run on demand, not as a persistent service. |

### Optional (toggled by profile)

| Component | Directory | Profile key | Purpose |
|---|---|---|---|
| `keycloak` | `keycloak-testing/` | `keycloak` | OIDC/JWT provider using Keycloak + internal PostgreSQL. Provides automated realm/client setup via init container. |
| `logto` | `logto-testing/` | `logto` | Alternative OIDC/JWT provider using Logto + internal PostgreSQL. Requires one-time manual admin setup after first boot. |
| `llama-server` | `llama.cpp/` | `llama` | llama.cpp inference server providing OpenAI-compatible API for embeddings (and optionally generation). Use when the host does not run llama-server natively. |
| `postgres` | _(managed by orchestrator)_ | `postgres` | Shared PostgreSQL instance with pgvector extension. Use when the host does not provide a PostgreSQL server. |

**Constraints:**
- `keycloak` and `logto` are mutually exclusive; only one may be active at a time.
- If `postgres` profile is disabled, `DATABASE_URL` in the master config must point to a host-provided PostgreSQL instance.
- If `llama` profile is disabled, `embed.host` must point to an existing llama-server instance.

---

## Master Configuration File

The orchestration system is driven by a single file: `stack.toml` at the repository root (`/mcp-servers/stack.toml`). This file is the single source of truth for all component configuration. No individual component `.env` or `config.toml` file is edited by hand when using the orchestrator; they are generated from `stack.toml`.

A `stack.toml.example` file must be committed to the repository. The real `stack.toml` must be listed in `.gitignore` because it contains secrets.

### File Format

TOML is used throughout. Below is the full annotated schema.

```toml
# stack.toml — master configuration for the MCP server stack

# ── Runtime ────────────────────────────────────────────────────────────────────
[runtime]
# Which container runtime to use: "podman" or "docker".
# If omitted, the orchestrator auto-detects: prefer podman if available, else docker.
engine = "podman"

# ── Active profiles ─────────────────────────────────────────────────────────────
# List the optional components to enable. Valid values:
#   "keycloak"  — start Keycloak + its internal PostgreSQL
#   "logto"     — start Logto + its internal PostgreSQL  (mutually exclusive with keycloak)
#   "llama"     — start llama-server container
#   "postgres"  — start a shared PostgreSQL container with pgvector
# Order does not matter. Dependency ordering is handled by the orchestrator.
profiles = ["postgres", "keycloak", "llama"]

# ── PostgreSQL (shared) ─────────────────────────────────────────────────────────
# Used when "postgres" is in profiles. Also used to build DATABASE_URL for
# rag-mcp-server and docs2vector regardless of whether the postgres profile
# is active (in which case the values must match the host instance).
[postgres]
image   = "docker.io/pgvector/pgvector:pg17"
host    = "localhost"          # hostname reachable from the host machine
port    = 5432                 # host-side port
user    = "support"
password = "changeme"          # REQUIRED to change for non-dev environments
database = "support"
data_volume = "stack-pgdata"   # named volume; omit or set "" to use a bind mount
# data_dir = "/path/on/host"   # use this instead of data_volume for bind mount

# ── llama-server ────────────────────────────────────────────────────────────────
[llama]
# Used when "llama" is in profiles.
image = "ghcr.io/ggml-org/llama.cpp:server"   # official llama.cpp server image
host_port = 16000              # port exposed on the host
# Path to the directory containing GGUF model files on the host.
models_dir = "/path/to/models"
# Filename of the embedding model GGUF to load.
embed_model = "mxbai-embed-large-v1-f16.gguf"
# Optional: filename of a generation model. Leave empty to disable generation.
gen_model = ""
# Extra flags passed verbatim to llama-server (e.g. "-ngl 99 --cont-batching")
extra_flags = ""

# ── Keycloak ────────────────────────────────────────────────────────────────────
[keycloak]
# Used when "keycloak" is in profiles.
port    = 8080                 # Keycloak host port
db_port = 5434                 # Keycloak-internal postgres host port
admin_user     = "admin"
admin_password = "admin"       # REQUIRED to change for non-dev environments
realm          = "dev"
api_client_id  = "my-api"     # resource server client ID (becomes aud claim)
m2m_client_id  = "my-app"    # M2M caller client ID
m2m_client_secret = "changeme-dev-secret"   # REQUIRED to change
token_lifetime = 3600          # access token TTL in seconds
hostname = "localhost"         # value stamped in KC_HOSTNAME / iss claim

# ── Logto ────────────────────────────────────────────────────────────────────
[logto]
# Used when "logto" is in profiles.
port       = 3001              # Logto OIDC host port
admin_port = 3002              # Logto admin console host port
db_port    = 5435              # Logto-internal postgres host port
endpoint        = "http://localhost:3001"
admin_endpoint  = "http://localhost:3002"
# These are filled in after first-boot admin console setup:
app_id     = ""
app_secret = ""
audience   = "https://my-service"
mgmt_app_id     = ""          # optional management API app
mgmt_app_secret = ""

# ── RAG MCP Server ──────────────────────────────────────────────────────────────
[rag_mcp_server]
port = 15080
log_level = "info"

# Which OIDC provider to use for JWT validation: "keycloak" or "logto".
# Must match one of the active profiles (or point to an external provider).
auth_provider = "keycloak"

# If auth_provider is "keycloak", these are derived automatically from [keycloak].
# Override here only if using an external Keycloak instance.
# auth_jwks_url = ""
# auth_issuer   = ""
# auth_audience = ""

# If auth_provider is "logto", these are derived automatically from [logto].
# Override here only if using an external Logto instance.
# auth_jwks_url = ""
# auth_issuer   = ""
# auth_audience = ""

[rag_mcp_server.search]
probes = 4
retrieval_pool_size = 20
rrf_constant = 60

[rag_mcp_server.reranker]
enabled = false
host = "http://localhost:8081"

[rag_mcp_server.guardrails]
corpus_topic   = ""
min_topic_score = 0.25
min_match_score = 0.0

[rag_mcp_server.hyde]
enabled = false
model = "claude-haiku-4-5-20251001"
base_url = ""
system_prompt = ""

# ── docs2vector ─────────────────────────────────────────────────────────────────
[docs2vector]
# Directory on the host containing documents to ingest.
docs_dir = "/path/to/docs"
# chunk size in tokens
chunk_size = 512
# embedding model name (must match what llama-server has loaded)
embed_model = "mxbai-embed-large-v1"
# query and passage instruction prefixes (model-specific)
query_prefix   = ""
passage_prefix = ""

# ── Secrets ──────────────────────────────────────────────────────────────────────
[secrets]
# Anthropic API key — required only when rag_mcp_server.hyde.enabled = true
anthropic_api_key = ""

# ── bs-ai-support-model (optional chatbot) ────────────────────────────────────
# [bs_ai_support_model]
# enabled = false
# port = 18080
```

---

## CLI Tool: `stack`

A CLI tool named `stack` must be implemented. It lives at `/mcp-servers/stack` (compiled binary) with source in `/mcp-servers/cmd/stack/`. It is written in Go.

### Command Interface

```
stack [--config stack.toml] <command> [flags]
```

| Command | Description |
|---|---|
| `up` | Generate all component configs and start all enabled services. |
| `down` | Stop all running services. |
| `restart` | `down` then `up`. |
| `status` | Show running/stopped state for each component and their ports. |
| `ingest` | Run the `docs2vector` ingestion job against the configured `docs_dir`. |
| `logs [component]` | Tail logs from one or all components. |
| `generate` | Generate all component config/env files but do not start containers (dry-run). |
| `validate` | Parse and validate `stack.toml` without generating anything. |

Global flags:
- `--config <path>` — path to master config file (default: `./stack.toml`)
- `--engine <podman|docker>` — override container engine (overrides `runtime.engine`)
- `--dry-run` — print compose commands without executing them

### Engine Auto-Detection

If `runtime.engine` is not set in `stack.toml` and `--engine` is not passed:
1. Check if `podman` is on `PATH` and executable. If yes, use it.
2. Else check if `docker` is on `PATH` and executable. If yes, use it.
3. Else fail with a clear error message.

The resolved engine is printed to stderr at startup: `using container engine: podman`.

For Podman, the compose command is `podman compose`. For Docker, it is `docker compose` (plugin) with fallback to `docker-compose` (standalone).

### Configuration Generation (`generate` step)

The `up` and `generate` commands must write the following files before starting containers. All generated files must be treated as ephemeral and are listed in `.gitignore`. They must never be committed.

| Generated file | Description |
|---|---|
| `keycloak-testing/.env` | Env vars for keycloak compose, populated from `[keycloak]` |
| `logto-testing/.env` | Env vars for logto compose, populated from `[logto]` |
| `rag-mcp-server/.env` | Env vars for rag-mcp-server compose (DATABASE_URL, ANTHROPIC_API_KEY) |
| `rag-mcp-server/config.toml` | Full config.toml for rag-mcp-server, generated from `[rag_mcp_server]` |
| `docs2vector/.env` | Env vars for docs2vector (DATABASE_URL) |
| `docs2vector/config.toml` | Config for docs2vector, generated from `[docs2vector]` and `[llama]` |
| `.stack/postgres.env` | Env vars for the optional shared postgres service |
| `.stack/llama.env` | Env vars for the optional llama-server service |

Auth-related fields in `rag-mcp-server/config.toml` (`[auth]` section) must be derived automatically:
- If `auth_provider = "keycloak"` and keycloak profile is active, derive `jwks_url`, `issuer`, and `audience` from `[keycloak]` values.
- If `auth_provider = "logto"` and logto profile is active, derive from `[logto]` values.
- If the profile is not active (external provider), the override fields `auth_jwks_url`, `auth_issuer`, `auth_audience` in `[rag_mcp_server]` are used and must be non-empty or validation fails.

The `embed.host` in generated configs must account for the container network:
- When `llama` profile is active, use the service name as hostname within compose network.
- When `llama` profile is inactive (host-provided), use `host.containers.internal` (podman) or `host-gateway` (docker) to reach the host. The tool must substitute the correct value based on the engine.

### Compose Orchestration

The tool must produce a unified compose invocation, not simply shell out to each component's compose file independently. It does this by passing multiple `-f` flags (or `--file`) to the compose command, one per active component:

```
podman compose \
  -f .stack/compose.postgres.yml \
  -f keycloak-testing/compose.yml \
  -f rag-mcp-server/compose.yaml \
  -f .stack/compose.llama.yml \
  up -d
```

The tool synthesizes compose files for components that do not have their own (shared postgres, llama-server):
- `.stack/compose.postgres.yml` — shared PostgreSQL with pgvector
- `.stack/compose.llama.yml` — llama-server

Each component's existing compose file is used as-is where it exists. The tool must not modify them.

### Dependency Ordering

Service startup must respect these dependencies:

1. `postgres` (if profile active) → all other services
2. `keycloak` or `logto` → keycloak-init (already handled internally in keycloak compose) → `rag-mcp-server`
3. `llama-server` → `rag-mcp-server`, `docs2vector`
4. `rag-mcp-server` — last to start

These are expressed via compose `depends_on` in the generated/merged compose files.

### Shared Network

All services must be placed on a single named compose network (`stack-net`) so they can reach each other by service name. Each component compose file must be updated (or the synthesized files must define it) so that all services join `stack-net` as an external network.

The shared network is created by the orchestrator on `up` if it does not exist:
```
podman network create stack-net
```

---

## Synthesized Compose Files

The orchestrator writes these files into a `.stack/` directory at the repo root. This directory is in `.gitignore`.

### `.stack/compose.postgres.yml`

```yaml
name: stack-postgres

networks:
  stack-net:
    external: true

volumes:
  stack-pgdata:

services:
  postgres:
    image: ${POSTGRES_IMAGE}
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    ports:
      - "127.0.0.1:${POSTGRES_PORT}:5432"
    volumes:
      - stack-pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - stack-net
    restart: unless-stopped
```

### `.stack/compose.llama.yml`

```yaml
name: stack-llama

networks:
  stack-net:
    external: true

services:
  llama-server:
    image: ${LLAMA_IMAGE}
    ports:
      - "127.0.0.1:${LLAMA_HOST_PORT}:8080"
    volumes:
      - ${LLAMA_MODELS_DIR}:/models:ro
    command: >
      --model /models/${LLAMA_EMBED_MODEL}
      --host 0.0.0.0
      --port 8080
      --embeddings
      ${LLAMA_EXTRA_FLAGS}
    networks:
      - stack-net
    restart: unless-stopped
```

---

## Component Compose File Modifications

The existing component compose files (`keycloak-testing/compose.yml`, `logto-testing/compose.yml`, `rag-mcp-server/compose.yaml`) each need a one-time addition to join the shared network. These changes are committed to each component's repo.

Add to each compose file:

```yaml
networks:
  stack-net:
    external: true
```

And add `networks: [stack-net]` to each service definition.

These are the only changes the orchestrator requires in the component compose files. All other configuration comes via environment variables written to per-component `.env` files.

---

## Variable Derivation Rules

The following derivations must be implemented in the `stack` tool. They produce values injected into generated `.env` and `config.toml` files.

### DATABASE_URL

When `postgres` profile is active:
```
postgres://<postgres.user>:<postgres.password>@postgres:<internal_port>/<postgres.database>?sslmode=disable
```
The hostname is the compose service name `postgres` (for container-to-container). For tools run from the host (like `docs2vector` in non-container mode), the hostname is `localhost` with the configured host port.

When `postgres` profile is inactive, `DATABASE_URL` is derived directly from `[postgres]` host/port/user/password/database.

### Keycloak auth fields

```
jwks_url  = http://keycloak:<KC_PORT>/realms/<realm>/protocol/openid-connect/certs
issuer    = http://<hostname>:<KC_PORT>/realms/<realm>
audience  = <api_client_id>
```

Note: `jwks_url` uses the compose service name `keycloak` (container-to-container), but `issuer` uses the configured hostname (what the JWT iss claim will contain, which is validated against what clients see).

### Logto auth fields

```
jwks_url  = http://logto:<LOGTO_PORT>/oidc/jwks
issuer    = http://<endpoint_host>:<LOGTO_PORT>/oidc
audience  = <logto.audience>
```

### embed.host in rag-mcp-server config.toml

When `llama` profile is active:
```
http://llama-server:8080
```
When `llama` profile is inactive:
- Podman: `http://host.containers.internal:<llama.host_port>`
- Docker: `http://host-gateway:<llama.host_port>` (requires `extra_hosts: host-gateway:host-gateway` in compose)

---

## Ingest Command

`stack ingest` runs docs2vector as a one-shot container. It must:
1. Ensure the `postgres` service is running and healthy (or that `DATABASE_URL` is reachable).
2. Ensure the llama-server is running and reachable.
3. Build the docs2vector image if not already built.
4. Run the container with `--drop` flag (recreates tables) unless `--no-drop` is passed.
5. Mount the configured `docs_dir` into the container at `/docs`.
6. Pass the generated `docs2vector/config.toml` and `docs2vector/.env` to the container.
7. Stream container logs to stdout until the container exits.
8. Report success or failure with the exit code.

```
stack ingest [--no-drop] [--docs-dir /path/override]
```

---

## Validation Rules

`stack validate` (and the validation pass inside `up`) must enforce:

1. `profiles` contains at most one of `keycloak`, `logto`.
2. If `postgres` profile is inactive, `postgres.host`, `postgres.port`, `postgres.user`, `postgres.password`, `postgres.database` are all non-empty.
3. If `llama` profile is active, `llama.models_dir` exists on the host and `llama.embed_model` is non-empty.
4. If `llama` profile is inactive, `rag_mcp_server` embed host override or the llama host_port must be provided via config (or the orchestrator assumes the host provides llama-server on the default port).
5. If `rag_mcp_server.hyde.enabled = true`, `secrets.anthropic_api_key` is non-empty.
6. If `auth_provider` refers to a profile not in `profiles`, the override fields (`auth_jwks_url`, `auth_issuer`, `auth_audience`) must all be non-empty.
7. `keycloak.m2m_client_secret` must not equal `"changeme-dev-secret"` when a future `strict` mode flag is added (warn but do not fail in default mode).
8. `postgres.password` must not equal `"changeme"` in strict mode (warn in default mode).

---

## Makefile

A `Makefile` at `/mcp-servers/Makefile` must provide:

```makefile
.DEFAULT_GOAL := help

help:          ## Show this help
up:            ## Start all enabled services
down:          ## Stop all services
restart:       ## Restart all services
status:        ## Show service status
ingest:        ## Run docs2vector ingestion
logs:          ## Tail logs (COMPONENT= to filter)
generate:      ## Generate component configs without starting
validate:      ## Validate stack.toml
build:         ## Build the stack CLI tool
test:          ## Run unit tests for the stack tool
clean:         ## Remove generated files in .stack/ and component .env files
```

All targets delegate to the `stack` binary except `build` (which runs `go build`) and `test` (which runs `go test`).

---

## File Layout

```
/mcp-servers/
├── stack.toml.example         # committed template; real stack.toml is gitignored
├── stack.toml                 # gitignored — real config with secrets
├── Makefile                   # targets delegating to stack binary
├── REQUIREMENTS.md            # this document
├── cmd/
│   └── stack/
│       └── main.go            # CLI entrypoint
├── internal/
│   ├── config/                # stack.toml parsing and validation
│   ├── engine/                # podman/docker detection and command building
│   ├── generate/              # file generation (env files, config.toml, compose ymls)
│   ├── compose/               # compose up/down/status orchestration
│   └── ingest/                # docs2vector run logic
├── .stack/                    # gitignored — generated compose files and env files
├── bs-ai-support-model/
├── docs2vector/
├── keycloak-testing/
├── llama.cpp/
├── logto-testing/
└── rag-mcp-server/
```

---

## .gitignore Requirements

The following must be added to `/mcp-servers/.gitignore` (create if absent):

```
stack.toml
.stack/
keycloak-testing/.env
logto-testing/.env
rag-mcp-server/.env
rag-mcp-server/config.toml
docs2vector/.env
docs2vector/config.toml
bin/
*~
```

---

## Non-Requirements (Explicit Exclusions)

The following are out of scope for this system:

- TLS termination or HTTPS between containers (developer-only use; all services bind to `127.0.0.1`)
- Remote deployment or Kubernetes orchestration
- Automatic model download for llama-server (operator must supply GGUF files)
- UI for managing the stack
- Secret management beyond environment variables (no Vault, no secret stores)
- Hot-reload of configuration without a full `restart`
- Multi-host or distributed setups

---

## Acceptance Criteria

An implementation satisfies these requirements when:

1. `make build` produces a working `./bin/stack` binary.
2. `make test` passes all unit tests.
3. `stack validate` correctly accepts a valid `stack.toml` and rejects invalid ones with descriptive errors.
4. `stack generate` produces syntactically correct YAML and TOML files for all active profiles.
5. `stack up` starts all enabled services in dependency order using either podman or docker, verifiable with `stack status`.
6. `stack down` stops all services cleanly.
7. `stack ingest` runs docs2vector to completion and reports its exit code.
8. The system works with only `postgres` + `keycloak` + `rag-mcp-server` active.
9. The system works with only `postgres` + `logto` + `llama` + `rag-mcp-server` active.
10. The system works when `postgres` and `llama` profiles are inactive (host-provided services).
11. No component `.env` or generated `config.toml` files are committed to the repository.
