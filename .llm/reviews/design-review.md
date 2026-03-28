# Design/Architecture Review

## Summary: 1 LOW, 10 PASS

---

## LOW — LLAMA_EXTRA_FLAGS written to env but not used in compose command

`llamaComposeYAML()` is a static string; it does not include extra_flags. The env var `LLAMA_EXTRA_FLAGS` is written but compose YAML `command:` arrays do not word-split env var references. The value is effectively ignored at runtime. Fix: make `llamaComposeYAML` accept pre-split flags and embed them as discrete YAML entries.

---

## PASS

1. **Package boundaries** — clear separation: `engine` is pure argv-building (no I/O), `generate` is pure file-writing (no container ops), `compose` drives lifecycle, `config` is pure parsing. No circular deps. ✓
2. **Error handling** — all errors wrapped with `fmt.Errorf("context: %w", err)`. No silently ignored errors. ✓
3. **Idiomatic Go naming** — camelCase, no abbreviations, meaningful exported names. ✓
4. **Magic numbers** — critical values defined as package-level constants (`healthTimeout`, `sharedNetwork`, `llamaTimeout`). ✓
5. **TOML field order** — `profiles` placed before first `[section]` header in both `stack.toml.example` and test fixtures. ✓
6. **Postgres service name consistency** — `stack-postgres` used consistently across `derive.go`, `generate.go`, `compose.go`, and the YAML template. ✓
7. **Health poll container names** — compose naming convention `{project}-{service}-1` correctly applied: `stack-keycloak-keycloak-1`, `stack-logto-logto-1`, `stack-postgres-stack-postgres-1`. ✓
8. **Dry-run** — generates files unconditionally, skips all container and network operations when `--dry-run` set. ✓
9. **Test coverage** — all 7 derive functions tested; validation rules tested individually; edge cases (shell metacharacters, inactive profiles, override fields) covered. ✓
10. **Dependency ordering** — `activeProjects` returns projects in startup-dependency order; `stack-rag` is always last. ✓
