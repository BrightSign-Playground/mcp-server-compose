# Security Review — Bedrock Implementation

## Summary: 0 CRITICAL, 1 HIGH, 3 MEDIUM, 3 LOW, 8 PASS

---

## HIGH — Shell scripts pass JSON body to `aws` CLI via shell expansion

**Location:** `rag-mcp-server/scripts/eval.sh` line 189; `rag-mcp-server/scripts/search.sh` line 233

The `call_claude` function in `eval.sh` and the Bedrock path in `search.sh` pass the JSON request body to `aws bedrock-runtime invoke-model --body "${body}"`. The `${body}` variable contains JSON built by `jq`, which safely encodes user input (the query string). However, the value is passed through shell double-quote expansion on the command line. If the JSON body exceeds `ARG_MAX` (typically 2MB on Linux), the command will fail silently or be truncated. More critically, the `--body` parameter accepts a string that the AWS CLI interprets — if a future code change constructs `body` without `jq` (as `search.sh` already does for the fallback non-jq path at line 150-153), shell metacharacters in user input could corrupt the JSON or cause unexpected behavior.

Current risk is **mitigated by jq** encoding all user-controlled values. The `jq -cn` calls properly escape special characters. But the pattern is fragile: any path that constructs `body` without jq is vulnerable.

**Recommendation:** Use `--body fileb://<(echo "${body}")` or write the body to a temp file and use `--body file:///tmp/request.json` to avoid shell expansion entirely. This also removes the `ARG_MAX` limitation.

---

## MEDIUM — `aws_region` not validated against injection patterns

**Location:** `rag-mcp-server/internal/config/config.go` lines 236-245; `rag-mcp-server/scripts/eval.sh` line 101; `rag-mcp-server/scripts/search.sh` line 76

The Go server validates that `hyde.provider` is one of `"anthropic"` or `"bedrock"`, and that `hyde.aws_region` is non-empty when provider is bedrock. However, `aws_region` is not validated against a pattern (e.g., `^[a-z]{2}-[a-z]+-\d+$`). In the Go code this flows into the AWS SDK which validates it internally, so the server-side risk is low. In the shell scripts, `AWS_REGION` is passed directly to `aws --region "${AWS_REGION}"` — the AWS CLI validates region format, but a malformed value could produce confusing errors rather than a clean validation failure at startup.

**Recommendation:** Add a regex validation for `aws_region` in the Go config loader (e.g., `^[a-z]{2}(-gov)?-[a-z]+-\d+$`) and in the shell scripts before first use.

---

## MEDIUM — Postgres credentials not URL-encoded in DATABASE_URL

**Location:** `internal/generate/derive.go` (previously identified)

Credentials are interpolated directly into `postgres://` URLs. If a password contains `@`, `:`, `/`, or `?`, the URL is malformed. This pre-dates the Bedrock changes but remains unfixed.

**Fix:** URL-encode username and password with `url.PathEscape`.

---

## MEDIUM — `embed.host` URL not validated in rag-mcp-server config

**Location:** `rag-mcp-server/internal/config/config.go`

The `reranker.host` and `hyde.base_url` are validated via `validateHTTPURL` to prevent SSRF (non-HTTP schemes like `file://`, `gopher://`). However, `embed.host` (`cfg.EmbedHost`) is never validated. A malicious or misconfigured `embed.host = "file:///etc/passwd"` would be passed to the HTTP client. The Go `net/http` client rejects non-HTTP schemes at the transport layer, so exploitation is unlikely, but defense-in-depth calls for consistent validation.

**Recommendation:** Apply `validateHTTPURL` to `cfg.EmbedHost` as well.

---

## LOW — Bedrock model ID not validated in shell scripts

**Location:** `rag-mcp-server/scripts/eval.sh` line 180; `rag-mcp-server/scripts/search.sh` line 228-230

The `EVAL_MODEL` / `CLAUDE_MODEL` variable is extracted from the JSON request and passed to `aws bedrock-runtime invoke-model --model-id "${model}"`. The model ID comes from the `EVAL_MODEL` environment variable (user-controlled) and is used as a CLI argument. The AWS CLI treats `--model-id` as a string identifier, not a path or URL, so injection risk is negligible. However, no format validation is performed — a typo or garbage value produces an unhelpful AWS error rather than an early validation failure.

**Recommendation:** Validate model ID format (e.g., must match `^[a-zA-Z0-9._:/-]+$`) before invoking the AWS CLI.

---

## LOW — `context.Background()` used for AWS config loading at init time

**Location:** `rag-mcp-server/internal/hyde/claude.go` line 61

`NewBedrockGenerator` calls `bedrock.WithLoadDefaultConfig(context.Background(), ...)`. This is called during server initialization, not in a request context. Using `context.Background()` means there is no timeout on the initial AWS credential resolution. If the IMDS endpoint is unreachable (e.g., running outside AWS without credentials configured), this could hang indefinitely.

**Recommendation:** Pass a context with a timeout (e.g., 10 seconds) or document that AWS credentials must be resolvable at startup.

---

## LOW — Health check accepts 4xx as healthy

**Location:** Previously identified in `internal/ingest/ingest.go:146`

Pre-existing issue. `resp.StatusCode < 500` treats 401, 403, 404 as healthy.

---

## PASS

1. **Credential handling — AWS credentials use credential chain only.** `NewBedrockGenerator` uses `bedrock.WithLoadDefaultConfig` which delegates to the standard AWS SDK credential chain (env vars, `~/.aws/credentials`, IAM role, IMDS). No AWS credentials are stored in config files, TOML, or generated `.env` files.

2. **Credential handling — ANTHROPIC_API_KEY read from environment only.** The API key is read via `os.Getenv("ANTHROPIC_API_KEY")` in `main.go` (line 79) and in the stack-level `config.go` (line 307). It is never stored in `config.toml` or logged. The `ragEnvVars` function in `generate.go` writes it to a `.env` file with `0600` permissions, which is acceptable for container orchestration.

3. **No credential leakage in logs.** The `main.go` logger at line 77 logs `provider`, `model`, and `aws_region` when Bedrock HyDE is enabled — none of these are secrets. The `redactURL` function (line 150) strips userinfo from URLs before logging. Error messages from the Anthropic SDK are wrapped with `fmt.Errorf("HyDE API call: %w", err)` which does not include credentials.

4. **SSRF prevention.** `validateHTTPURL` rejects non-HTTP/HTTPS schemes for `reranker.host` and `hyde.base_url`. The Bedrock path does not introduce any new URL endpoints — it uses the AWS SDK which handles endpoint construction internally.

5. **Provider field validated.** `config.go` line 236-239 validates `hyde.provider` against an allowlist (`"anthropic"`, `"bedrock"`). Unknown values cause a startup error.

6. **Shell scripts use jq for JSON construction.** Both `eval.sh` and `search.sh` use `jq -cn` with `--arg` to safely construct JSON payloads. User-controlled values (queries, model names) are properly JSON-encoded by jq, preventing JSON injection.

7. **No new SSRF vectors.** The Bedrock integration routes through the AWS SDK, not through user-controlled URLs. The `aws bedrock-runtime invoke-model` command connects to the Bedrock service endpoint determined by the region — there is no user-controlled URL that could be redirected to internal services.

8. **Generated config.toml contains no secrets.** The `ragConfigTOML` function in `generate.go` writes `provider`, `model`, `base_url`, `aws_region`, and `system_prompt` to the generated TOML file. None of these are secrets. AWS credentials are resolved at runtime via the credential chain, not stored in config files.
