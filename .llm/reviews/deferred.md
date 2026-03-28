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
