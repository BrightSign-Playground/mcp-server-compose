# Spec Compliance Review

## Overall: 13 PASS, 1 FAIL (intentional deviation documented), 1 FAIL (genuine miss)

---

## FAIL — LLAMA_EXTRA_FLAGS not injected into llama-server command

**Severity: HIGH**

`llama.extra_flags` is parsed, validated against shell metacharacters, and written to `.stack/llama.env` as `LLAMA_EXTRA_FLAGS`, but the value is never injected into the llama-server `command:` block in `llamaComposeYAML()`. The requirement states extra_flags must be appended verbatim to the llama-server invocation.

- `internal/generate/generate.go`: `llamaComposeYAML()` is a static string with no extra_flags
- `internal/generate/generate.go`: `llamaEnvVars()` writes `LLAMA_EXTRA_FLAGS` to env but this has no effect on the command
- Compose YAML `command:` arrays do not word-split env var references — `${LLAMA_EXTRA_FLAGS}` would be passed as one token, not split flags

**Fix:** Make `llamaComposeYAML` accept `[]string` pre-split extra flags and embed them as discrete YAML command entries.

---

## INTENTIONAL DEVIATION — postgres service name is `stack-postgres` not `postgres`

**Severity: NONE (design decision)**

REQUIREMENTS.md synthesised compose template shows service name `postgres`. Implementation uses `stack-postgres`. This is a documented design decision to avoid DNS collision on `stack-net` when both shared postgres and keycloak/logto internal postgres services are active simultaneously. Accepted deviation — update REQUIREMENTS.md to reflect this.

---

## PASS items

1. `make build` produces `./bin/stack` ✓
2. All 8 commands implemented (up, down, restart, status, ingest, logs, generate, validate) ✓
3. All 8 validation rules enforced ✓
4. All 10 generated files wired up ✓
5. Engine auto-detection (podman preferred, docker fallback) ✓
6. DATABASE_URL derivation (container-side and host-side) ✓
7. Keycloak auth field derivation ✓
8. Logto auth field derivation ✓
9. embed.host derivation for podman, docker, and llama-active cases ✓
10. .gitignore contains all required entries ✓
11. Makefile has all required targets ✓
12. keycloak/logto mutual exclusion enforced ✓
13. ingest --no-drop flag implemented ✓
14. stack-net added to all three component compose files ✓
