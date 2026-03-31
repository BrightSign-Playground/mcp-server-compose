# stack

Single-binary CLI tool that orchestrates the MCP server stack for local development,
integration testing, and RAG dataset building. Reads `stack.toml`, derives all component
configuration from it, and drives podman/docker compose to start or stop the full stack.

## Quick Start

### 1. Prerequisites

Install [Go](https://go.dev/), [Podman](https://podman.io/) (or Docker), and
[uv](https://docs.astral.sh/uv/getting-started/installation/), then:

```sh
make prereqs          # installs huggingface_hub CLI + podman-compose
make build            # produces ./bin/stack
```

### 2. Configure and start the stack

```sh
cp stack.toml.example stack.toml
$EDITOR stack.toml    # set llama.models_dir, adjust passwords

make up               # generates configs, creates network, starts all services
make ingest           # ingest documents into the vector database
```

Wait for `make up` to report all services healthy. The MCP endpoint is now
listening at `http://localhost:15080/mcp`.

### 3. Get a JWT token

Every request to the MCP server requires a valid JWT. The stack includes
Keycloak (default) or Logto as an OIDC provider. Keycloak is fully automated --
realm, clients, and audience are provisioned on first start.

```sh
# With the default Keycloak config from stack.toml.example:
export OIDC_PROVIDER=keycloak
export KEYCLOAK_ISSUER=http://localhost:8080/realms/dev
export KEYCLOAK_CLIENT_ID=my-app
export KEYCLOAK_CLIENT_SECRET=changeme-dev-secret

TOKEN=$(./rag-mcp-server/scripts/get-token.sh)
echo "$TOKEN"
```

### 4. Test with curl

```sh
# Initialize MCP session
SESSION=$(curl -s -X POST http://localhost:15080/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0.1"}}}' \
  -D - -o /dev/null 2>&1 | grep -i mcp-session-id | tr -d '\r' | awk '{print $2}')

# Send initialized notification
curl -s -X POST http://localhost:15080/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: $SESSION" \
  -d '{"jsonrpc":"2.0","method":"notifications/initialized"}'

# Search documents
curl -s -X POST http://localhost:15080/mcp \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: $SESSION" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"search_documents","arguments":{"query":"how do I reset my password","limit":5}}}' | jq .
```

### 5. Connect from Claude Code

Claude Code supports MCP servers over Streamable HTTP. Because this server
requires JWT authentication, you need to supply a token via headers.

**Option A: Dynamic token refresh with `headersHelper` (recommended)**

Create a script that outputs auth headers as JSON. Claude Code calls it
automatically before each MCP request, so tokens are always fresh:

```sh
#!/usr/bin/env bash
# get-mcp-headers.sh — outputs JSON headers for Claude Code headersHelper
set -euo pipefail

export OIDC_PROVIDER=keycloak
export KEYCLOAK_ISSUER=http://localhost:8080/realms/dev
export KEYCLOAK_CLIENT_ID=my-app
export KEYCLOAK_CLIENT_SECRET=changeme-dev-secret

TOKEN=$(./rag-mcp-server/scripts/get-token.sh)
echo "{\"Authorization\": \"Bearer ${TOKEN}\"}"
```

```sh
chmod +x get-mcp-headers.sh
```

Then add to `.mcp.json` in your project root (or `~/.claude/mcp.json`
for global):

```json
{
  "mcpServers": {
    "rag-search": {
      "type": "http",
      "url": "http://localhost:15080/mcp",
      "headersHelper": "./get-mcp-headers.sh"
    }
  }
}
```

This is the best approach -- tokens are fetched on demand and never go stale.

**Option B: Static token via environment variable**

If you prefer a simpler setup and can tolerate manual token refresh:

```json
{
  "mcpServers": {
    "rag-search": {
      "type": "http",
      "url": "http://localhost:15080/mcp",
      "headers": {
        "Authorization": "Bearer ${RAG_MCP_TOKEN}"
      }
    }
  }
}
```

Claude Code expands `${RAG_MCP_TOKEN}` from your environment. Set it
before launching:

```sh
export RAG_MCP_TOKEN=$(./rag-mcp-server/scripts/get-token.sh)
claude
```

Or automate it in `.envrc` (direnv refreshes on every shell entry):

```sh
# .envrc — auto-refresh token on every shell entry
export OIDC_PROVIDER=keycloak
export KEYCLOAK_ISSUER=http://localhost:8080/realms/dev
export KEYCLOAK_CLIENT_ID=my-app
export KEYCLOAK_CLIENT_SECRET=changeme-dev-secret
export RAG_MCP_TOKEN=$(./rag-mcp-server/scripts/get-token.sh)
```

Tokens expire after 1 hour (default `keycloak.token_lifetime` in
`stack.toml`). With Option B you must re-export `RAG_MCP_TOKEN` and
restart Claude Code when the token expires.

**Option C: CLI one-liner**

```sh
claude mcp add --transport http rag-search http://localhost:15080/mcp \
  --header "Authorization: Bearer $(./rag-mcp-server/scripts/get-token.sh)"
```

### 6. Connect from VS Code (Copilot / Continue / other MCP clients)

VS Code MCP clients that support Streamable HTTP can connect the same way.
Add to your VS Code `settings.json`:

```json
{
  "mcp": {
    "servers": {
      "rag-search": {
        "type": "http",
        "url": "http://localhost:15080/mcp",
        "headers": {
          "Authorization": "Bearer ${RAG_MCP_TOKEN}"
        }
      }
    }
  }
}
```

Set `RAG_MCP_TOKEN` in your environment before launching VS Code, or use
a `.env` file if your MCP client supports it.

For MCP clients that only support stdio transport, use
[mcp-remote](https://github.com/anthropics/mcp-remote) as a bridge:

```sh
TOKEN=$(./rag-mcp-server/scripts/get-token.sh)

npx mcp-remote http://localhost:15080/mcp \
  --header "Authorization: Bearer $TOKEN"
```

Then configure the stdio client to run the `npx mcp-remote` command.

### Authentication summary

```mermaid
sequenceDiagram
    participant C as Claude Code / VS Code
    participant P as mcp-auth-proxy.sh
    participant KC as Keycloak (:8080)
    participant R as rag-mcp-server (:15080)

    C->>P: Need token (or .envrc auto-runs)
    P->>KC: POST /token (client_credentials)
    KC-->>P: JWT access_token
    P-->>C: RAG_MCP_TOKEN set

    C->>R: POST /mcp + Bearer JWT + query
    R->>R: Validate JWT (JWKS cached)
    R-->>C: search results (MCP JSON-RPC)
```

---

## Architecture

See [docs/DESIGN.md](docs/DESIGN.md) for full architecture.

### System Block Diagram

```mermaid
graph TB
    subgraph cli["stack CLI (Go binary)"]
        ST["stack.toml"] --> GEN["Config Generator"]
    end

    subgraph net["stack-net (shared container network)"]
        PG[("PostgreSQL\n+ pgvector\n:5432")]
        LL["llama-server\n(embeddings)\n:8080"]
        KC["Keycloak / Logto\n(OIDC provider)"]
        RAG["rag-mcp-server\n(MCP endpoint)\n:15080"]
        D2V["docs2vector\n(one-shot ingest)"]
    end

    subgraph agent["AI Agent Host"]
        A["AI Agent\n(Claude, GPT, etc.)"]
    end

    GEN -->|"generates .env,\ncompose YAML,\nconfig.toml"| net
    A -->|"0. GET JWT\n(client_credentials)"| KC
    A -->|"1. POST /mcp\n+ Bearer JWT\n+ query"| RAG
    RAG -->|"2. POST /v1/embeddings\n(query text)"| LL
    RAG -->|"3. KNN + FTS search"| PG
    RAG -->|"4. ranked chunks"| A
    D2V -->|"embed chunks"| LL
    D2V -->|"write vectors"| PG
```

### End-to-End Call Sequence

```mermaid
sequenceDiagram
    participant U as User / Operator
    participant S as stack CLI
    participant PG as PostgreSQL + pgvector
    participant KC as Keycloak / Logto
    participant LL as llama-server
    participant R as rag-mcp-server
    participant D as docs2vector
    participant A as AI Agent

    Note over U,R: Phase 1 — Stack Startup
    U->>S: stack up
    S->>S: Parse stack.toml, validate, generate configs
    S->>PG: Start (compose up), health poll
    PG-->>S: healthy
    S->>KC: Start (compose up), health poll
    KC-->>S: healthy
    S->>LL: Start (compose up)
    S->>R: Start (compose up)
    S-->>U: Stack is up

    Note over U,D: Phase 2 — Document Ingestion
    U->>S: stack ingest --docs-dir /path
    S->>D: Build image, run container
    D->>D: Walk files, chunk content
    loop For each chunk
        D->>LL: POST /v1/embeddings (chunk text)
        LL-->>D: float32 vector
        D->>D: L2-normalize
        D->>PG: INSERT chunk + embedding
    end
    D->>PG: Build IVFFlat index
    D-->>U: Ingest complete

    Note over A,R: Phase 3 — Agent Query
    A->>KC: POST /token (client_credentials)
    KC-->>A: JWT access_token
    A->>R: POST /mcp + Bearer JWT + query
    R->>R: Validate JWT (sig, iss, aud, exp)
    R->>LL: POST /v1/embeddings (query)
    LL-->>R: float32 query vector
    R->>R: L2-normalize, guardrail L1 check
    R->>PG: Vector KNN search (parallel)
    R->>PG: Full-text search (parallel)
    PG-->>R: Matching chunks (both arms)
    R->>R: RRF merge, optional rerank, guardrail L2
    R-->>A: Ranked document chunks (JSON)
```

### Internal Packages

- `internal/config` — pure TOML parsing and validation
- `internal/engine` — podman/docker detection, all compose/run argv construction
- `internal/generate` — derives values from config, writes ephemeral env/YAML/TOML files
- `internal/compose` — lifecycle: up, down, restart, status, logs
- `internal/ingest` — one-shot docs2vector container run

Only dependency outside the Go standard library: `github.com/BurntSushi/toml`.

## Prerequisites

The stack requires the following tools installed on your system:

- **Go** — to build the stack CLI
- **Podman** (or Docker) — container engine
- **uv** — Python tool installer ([install](https://docs.astral.sh/uv/getting-started/installation/))

Once `uv` is installed, run:

```sh
make prereqs
```

This installs:

- `huggingface_hub[cli]` — used to download GGUF model files
- `podman-compose` — required for container orchestration with Podman

## Build

```sh
make build          # produces ./bin/stack
make test           # runs all unit tests
make clean          # removes ./bin/ and .stack/
```

## Configuration

Copy the example config and fill in your values:

```sh
cp stack.toml.example stack.toml
$EDITOR stack.toml
```

`stack.toml` is gitignored. See `stack.toml.example` for all fields and their defaults.

**Secrets must never be stored in `stack.toml`.** Use environment variables instead,
preferably via a `.envrc` file loaded by [direnv](https://direnv.net/):

```sh
# .envrc — create this file; it is gitignored
export ANTHROPIC_API_KEY="sk-ant-..."   # required only when hyde is enabled
```

Then run `direnv allow` to activate it. Both `.env` and `.envrc` are in `.gitignore`.

Key fields:

| Field | Description |
|---|---|
| `profiles` | Active components: `postgres`, `keycloak`, `logto`, `llama` |
| `runtime.engine` | `podman` or `docker`; omit to auto-detect (prefers podman) |
| `postgres.*` | Shared PostgreSQL+pgvector container or external connection |
| `llama.*` | llama-server container; `extra_flags` appended to invocation |
| `keycloak.*` | Keycloak + internal postgres (mutually exclusive with logto) |
| `logto.*` | Logto + internal postgres (mutually exclusive with keycloak) |
| `rag_mcp_server.*` | RAG MCP server settings, auth provider, search tuning |
| `docs2vector.*` | Document ingestion settings |

## Commands

```sh
stack [--config PATH] [--engine ENGINE] [--dry-run] <command>

  up          Generate configs and start all active components
  down        Stop all active components
  restart     Down then up
  status      Show compose ps for all active components
  logs        Stream logs (--component NAME to filter)
  ingest      Build docs2vector image and run one-shot ingestion
  generate    Write all generated files without starting containers
  validate    Validate stack.toml without writing any files
```

Global flags can appear before or after the subcommand name.

### Examples

```sh
# Start the full stack
stack up

# Start with an explicit config path
stack --config /path/to/stack.toml up

# Generate files only (no container ops)
stack generate

# Run document ingestion
stack ingest

# Ingest without dropping existing tables
stack ingest --no-drop

# Show logs for a single component
stack logs --component llama

# Dry-run: show what would happen without executing
stack --dry-run up
```

## Generated Files

`stack generate` (and `stack up`) writes these files:

| File | Purpose |
|---|---|
| `.stack/postgres.env` | Postgres container env vars |
| `.stack/compose.postgres.yml` | Postgres compose file |
| `.stack/llama.env` | llama-server env vars |
| `.stack/compose.llama.yml` | llama-server compose file (includes extra_flags) |
| `keycloak-testing/.env` | Keycloak env vars |
| `logto-testing/.env` | Logto env vars |
| `rag-mcp-server/.env` | RAG server env vars (DATABASE_URL, API keys) |
| `rag-mcp-server/config.toml` | RAG server config (auth, search, embeddings) |
| `docs2vector/.env` | docs2vector env vars |
| `docs2vector/config.toml` | docs2vector config (embed host, chunk size) |

All `.env` files are written with mode `0600`.

## Networking

All components join an external Docker/Podman network named `stack-net`. The network
is created automatically by `stack up`. Each component runs as a separate compose
project (`stack-postgres`, `stack-keycloak`, `stack-logto`, `stack-llama`, `stack-rag`)
to avoid service name collisions on the shared network.

## Development

```sh
make test           # unit tests
make validate       # validate your stack.toml
make generate       # write generated files (inspect before running up)
make up             # build + generate + start
make down           # stop
make status         # show container status
```

## Troubleshooting

### Keycloak crash-loops with "Killed" on first start

If `podman logs stack-keycloak_keycloak_1` shows repeated lines like:

```
Updating the configuration and installing your custom providers, if any. Please wait.
/opt/keycloak/bin/kc.sh: line 169:    74 Killed   'java' ...
```

Keycloak's JVM is being OOM-killed by the container memory limit. This typically happens
on the first start when Keycloak runs its Quarkus augmentation/build phase, which is more
memory-intensive than normal operation.

**Fix:** Increase the memory limit in `keycloak-testing/compose.yml`:

```yaml
deploy:
  resources:
    limits:
      memory: 2g    # increase from 1g
```

Then restart:

```sh
make down
make up
```

### `make up` hangs after containers start

On older versions of podman (< 5.x), `podman-compose up -d` may not detach properly
when containers have health checks with dependencies. The containers are running — check
with `make status` or `podman ps` in another terminal. Press `Ctrl+C` to return to
your prompt; the containers will continue running in the background.

### `make ingest` fails with "network not found"

The `stack-net` network is created by `make up`. Run `make up` first, then `make ingest`.
