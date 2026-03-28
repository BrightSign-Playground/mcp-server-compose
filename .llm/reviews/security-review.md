# Security Review

## Summary: 1 HIGH, 2 MEDIUM, 2 LOW, 5 PASS

---

## HIGH — LLAMA_EXTRA_FLAGS unused (dead validated code)

`llama.extra_flags` is validated against shell metacharacters but never reaches the container command. The validation is correct (rejects `;|$><&``) but serves no purpose if the value is never used. Fix: inject extra_flags into the llama compose command so the validation is meaningful.

---

## MEDIUM — Postgres credentials not URL-encoded in DATABASE_URL

**Location:** `internal/generate/derive.go` lines 43, 51-54; `internal/ingest/ingest.go` line 162

Credentials are interpolated directly into postgres:// URLs. If a password contains `@`, `:`, `/`, or `?`, the URL is malformed and the connection will fail — or worse, part of the password could be interpreted as a hostname, leaking it in log output.

**Fix:** URL-encode username and password with `url.PathEscape`.

---

## MEDIUM — Port ranges not validated

Port numbers (`keycloak.port`, `logto.port`, `postgres.port`, `llama.host_port`, etc.) are stored as `int` with no range check. A misconfigured `port = 99999` or `port = -1` will cause runtime failures with confusing errors.

**Fix:** Add a port range validation (1–65535) in `config.Validate`.

---

## LOW — Health check accepts 4xx as healthy

`internal/ingest/ingest.go:146`: `resp.StatusCode < 500` treats 401, 403, 404 as a healthy llama-server. A misconfigured or wrong endpoint would pass the check.

**Fix:** Require `resp.StatusCode < 300` (or specifically 200).

---

## LOW — Logto/Keycloak compose files have hardcoded internal credentials

`logto-testing/compose.yml` and `keycloak-testing/compose.yml` embed credentials (`logto:logto`, `keycloak:keycloak`) for their internal postgres instances. These are dev-only and already documented as such, with no path to production. Acceptable for dev tooling; not remediated.

---

## PASS

1. **Command injection prevention** — all `exec.Command` calls use explicit `[]string` argv; no `sh -c` with user input anywhere ✓
2. **File permissions** — `.env` files written with mode `0600`; YAML/TOML with `0644`; atomic temp+rename pattern ✓
3. **Port binding** — all generated and existing compose files bind to `127.0.0.1` only ✓
4. **Path validation** — `validateDir` rejects null bytes, requires absolute path, requires directory ✓
5. **Extra flags metachar validation** — `validateExtraFlags` rejects `;|`` $><&` before any use ✓
