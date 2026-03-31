# Deferred Findings

## LOW — Health check: 4xx treated as healthy

**Original finding (security-review.md):** `resp.StatusCode < 500` treated 401/403/404 as healthy.

**Status: REMEDIATED.** Changed to `resp.StatusCode < 300` in `internal/ingest/ingest.go`.

---

## LOW — Logto/Keycloak hardcoded internal credentials

**Location:** `logto-testing/compose.yml`, `keycloak-testing/compose.yml`

**Finding:** Hardcoded credentials (`logto:logto`, `keycloak:keycloak`) for internal postgres instances.

**Rationale for deferral:** These are dev-only compose files with no production path. The credentials are for the internal postgres instances within each component's own compose project, not the shared `stack-postgres`. They are documented as dev tooling, gitignored from the user's stack config, and follow the upstream projects' own conventions. Changing them would diverge from the upstream repos and require coordination with those projects.

---

## MEDIUM — Port ranges not validated (originally)

**Status: REMEDIATED.** Added `validatePort` helper and port range checks (1–65535) for all eight port fields in `config.Validate()`. Ports set to zero (use-default) are exempt.

---

## MEDIUM — Postgres credentials not URL-encoded (originally)

**Status: REMEDIATED.** `databaseURLContainer`, `databaseURLHost` in `internal/generate/derive.go` and `databaseURLForContainer` in `internal/ingest/ingest.go` now use `url.URL` with `url.UserPassword` for correct percent-encoding.

---

## HIGH — LLAMA_EXTRA_FLAGS not injected into compose command (originally)

**Status: REMEDIATED.** `llamaComposeYAML` now accepts `[]string` pre-split flags and embeds them as discrete YAML command entries. Call site in `All()` passes `strings.Fields(cfg.Llama.ExtraFlags)`.

---

## Bedrock Implementation — Deferred Findings (2026-03-31)

### MEDIUM (pre-existing, not introduced by Bedrock change)

- **embed.host URL not validated:** `rag-mcp-server/internal/config/config.go` does not call `validateHTTPURL` on `cfg.EmbedHost`, unlike `reranker.host` and `hyde.base_url`. Go's `net/http` rejects non-HTTP schemes at the transport layer.

### LOW

- **Bedrock default model uses cross-region inference prefix:** `us.anthropic.claude-haiku-4-5-20251001-v1:0` uses the `us.` prefix; may not work in non-US regions. Acceptable since the model is configurable.
- **builtinDefaults() sets Anthropic model format regardless of provider:** Users switching to bedrock without changing the model will get a Bedrock API error. Acceptable — users must configure the correct model for their provider.
- **eval.sh EVAL_MODEL default is Anthropic format:** Users must set `EVAL_MODEL` to the Bedrock model ID manually when using `ANTHROPIC_PROVIDER=bedrock`.
- **search.sh Bedrock path does not stream:** `aws bedrock-runtime invoke-model` does not support SSE streaming; returns a single response. Acceptable for a developer tool.
- **Bedrock test coverage is structural only:** Tests verify struct field assignment but not the `Generate()` path for Bedrock. Would need Bedrock response format simulation.
