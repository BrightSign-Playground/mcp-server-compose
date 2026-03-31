# Overall System Sequence Diagram

This document describes the full end-to-end lifecycle of the stack: from startup
through document ingestion to agent-driven RAG queries.

## System Block Diagram

All components run as separate compose projects on a shared `stack-net` network.
The `stack` CLI reads `stack.toml`, generates per-component configuration files,
and drives podman/docker compose to start each service.

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

**Components:**

- **stack CLI** — Go binary that parses `stack.toml`, validates configuration,
  generates `.env` and compose files, and orchestrates container lifecycle.
- **PostgreSQL + pgvector** — Shared vector database storing document chunks and
  their embeddings. Supports both KNN vector search and full-text search.
- **llama-server** — Local embedding model server exposing an OpenAI-compatible
  `/v1/embeddings` endpoint. Used by both docs2vector (ingestion) and
  rag-mcp-server (query-time).
- **Keycloak / Logto** — OIDC identity provider issuing JWTs for machine-to-machine
  authentication. Only one can be active at a time.
- **rag-mcp-server** — MCP endpoint that authenticates requests, embeds queries,
  searches the vector store, and returns ranked document chunks.
- **docs2vector** — One-shot ingestion container that walks a document directory,
  chunks content, embeds each chunk via llama-server, and writes vectors to PostgreSQL.

## End-to-End Call Sequence

The sequence has three phases: stack startup, document ingestion, and agent query.

### Phase 1 — Stack Startup

The operator runs `stack up`. The CLI parses and validates `stack.toml`, generates
all configuration files (`.env`, compose YAML, `config.toml`), creates the shared
`stack-net` network, and starts components sequentially. PostgreSQL and the OIDC
provider are health-polled before downstream services start, ensuring dependencies
are ready.

### Phase 2 — Document Ingestion

The operator runs `stack ingest --docs-dir /path`. The CLI builds the docs2vector
image and runs it as a one-shot container. docs2vector walks the document directory,
splits files into chunks, and for each chunk: requests an embedding from llama-server,
L2-normalizes the vector, and inserts the chunk text plus embedding into PostgreSQL.
After all chunks are ingested, it builds an IVFFlat index for fast approximate
nearest-neighbor search.

### Phase 3 — Agent Query

An AI agent authenticates with the OIDC provider using client credentials to obtain
a JWT. It then sends a query to rag-mcp-server via `POST /mcp` with the Bearer token.
The server validates the JWT, embeds the query text via llama-server, L2-normalizes
the vector, and optionally runs a Level 1 guardrail (topic relevance check). It then
executes parallel vector KNN and full-text searches against PostgreSQL, merges results
using Reciprocal Rank Fusion (RRF), optionally applies a Level 2 guardrail (match
quality check) and reranking, and returns the ranked document chunks to the agent.

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
