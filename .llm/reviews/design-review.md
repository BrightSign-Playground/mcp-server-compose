# Design/Architecture Review: Bedrock Implementation

**Reviewer:** Architecture Review Agent
**Date:** 2026-03-31
**Scope:** Bedrock provider support for HyDE across rag-mcp-server and stack tooling

---

## Summary

The Bedrock implementation adds an alternative HyDE provider that routes Claude API calls through Amazon Bedrock instead of the direct Anthropic API. The implementation spans two layers: the rag-mcp-server (runtime) and the stack tool (config generation). Overall the implementation follows existing patterns well, with a few findings detailed below.

---

## Findings

### CRITICAL

None.

### HIGH

**H-1: DESIGN.md Config struct is stale -- missing `HyDEProvider` and `HyDEAWSRegion` fields**
- File: `/mcp-server-compose/rag-mcp-server/docs/DESIGN.md` (section 4.2, lines 176-181)
- The DESIGN.md Config struct definition does not include `HyDEProvider`, `HyDEAWSRegion`, or the Bedrock-related config fields. The code (`internal/config/config.go`) has these fields, but the design document is now out of date. Per the global CLAUDE.md: "When code and spec disagree, fix the code. Always update specs before changing implementation."
- The TOML schema section (4.4, lines 239-243) likewise omits `provider` and `aws_region` from the `[hyde]` section.
- **Recommendation:** Update DESIGN.md sections 4.2 and 4.4 to reflect the new `HyDEProvider` (string, "anthropic" or "bedrock") and `HyDEAWSRegion` (string, required when provider is "bedrock") fields.

**H-2: DESIGN.md startup sequence references `NewAnthropicHyDE` but code uses `NewClaudeGenerator`/`NewBedrockGenerator`**
- File: `/mcp-server-compose/rag-mcp-server/docs/DESIGN.md` (section 9, line 668)
- The startup flowchart shows `NewAnthropicHyDE` but the code has `NewClaudeGenerator` and `NewBedrockGenerator`. The design doc also lists `AnthropicHyDE` in the interface section (5.3) but the implementation file is `claude.go` with type `ClaudeGenerator`.
- **Recommendation:** Update DESIGN.md to use the actual type names. The startup flowchart should branch on provider (anthropic vs bedrock).

**H-3: DESIGN.md section 5.3 does not document `NewBedrockGenerator`**
- File: `/mcp-server-compose/rag-mcp-server/docs/DESIGN.md` (section 5.3, lines 370-381)
- The Generator interface section only mentions `NoopGenerator` and `AnthropicHyDE`. `NewBedrockGenerator` and the Bedrock code path are not documented.
- **Recommendation:** Document `NewBedrockGenerator(awsRegion, model, systemPrompt string) *ClaudeGenerator` alongside `NewClaudeGenerator`.

### MEDIUM

**M-1: File renamed from `anthropic.go` to `claude.go` but DESIGN.md package structure still lists `anthropic.go`**
- File: `/mcp-server-compose/rag-mcp-server/docs/DESIGN.md` (section 3, line 88)
- The package structure lists `hyde/anthropic.go` but the actual file is `hyde/claude.go`.
- **Recommendation:** Update the package structure listing in DESIGN.md.

**M-2: `NewBedrockGenerator` creates an AWS config at construction time using `context.Background()`**
- File: `/mcp-server-compose/rag-mcp-server/internal/hyde/claude.go` (line 61)
- `bedrock.WithLoadDefaultConfig(context.Background(), ...)` is called inside the constructor. This means the AWS credential chain is resolved with an unbounded context at startup. If the AWS metadata service is slow (e.g., ECS/EC2 with IMDS v2 issues), this could hang the server startup indefinitely. A cancellable context or timeout would be safer.
- **Recommendation:** Accept a `context.Context` parameter in `NewBedrockGenerator` or use a context with timeout internally.

**M-3: `NewClaudeGenerator` godoc says "apiKey is required" but `NewBedrockGenerator` reuses `ClaudeGenerator` without an API key**
- File: `/mcp-server-compose/rag-mcp-server/internal/hyde/claude.go` (lines 26-27, 52)
- The godoc on `NewClaudeGenerator` states "apiKey is required" which is true for the Anthropic path, but `NewBedrockGenerator` creates the same `ClaudeGenerator` type without an API key. The struct type name `ClaudeGenerator` is reused for both paths. This is not incorrect but could cause confusion.
- **Recommendation:** Consider either: (a) renaming `ClaudeGenerator` to something more generic (e.g., `anthropicGenerator`), or (b) adding a brief godoc note on `ClaudeGenerator` that it serves both direct API and Bedrock paths.

**M-4: ANTHROPIC_API_KEY validation gap in main.go for Bedrock path**
- File: `/mcp-server-compose/rag-mcp-server/cmd/server/main.go` (lines 73-87)
- When `HyDEProvider == "bedrock"`, the Bedrock generator is created without checking whether AWS credentials are actually available. The code relies on the AWS SDK to fail at request time. For the Anthropic path, a missing `ANTHROPIC_API_KEY` logs a warning and falls back to NoopGenerator. The Bedrock path has no equivalent early validation.
- This is partially mitigated by `config.Load()` requiring `aws_region` for Bedrock, but the actual credential availability is not checked.
- **Recommendation:** Add a startup log at INFO level noting that Bedrock credentials will be resolved via the default AWS credential chain, so operators know where to look if it fails.

**M-5: Stack config validation does not validate `hyde.provider` enum**
- File: `/mcp-server-compose/internal/config/config.go` (Validate function)
- The stack-level `Validate()` checks `bedrock` for `aws_region` requirement and `anthropic` for API key requirement, but does not explicitly reject invalid provider values (e.g., `provider = "openai"`). An invalid provider would silently pass validation and be written into the generated rag-mcp-server config.toml, where the rag-mcp-server's own config.Load() would then reject it.
- The rag-mcp-server config.Load() does validate the provider enum (lines 236-239), so the error will surface at runtime, but it would be better to catch it earlier.
- **Recommendation:** Add a provider enum validation in stack's `Validate()` to fail fast during `stack generate`.

### LOW

**L-1: Default Bedrock model includes cross-region inference prefix**
- File: `/mcp-server-compose/rag-mcp-server/internal/hyde/claude.go` (line 54)
- The default Bedrock model is `"us.anthropic.claude-haiku-4-5-20251001-v1:0"` which uses the `us.` cross-region inference prefix. This is a reasonable default for US regions but may not work in all regions (e.g., `eu-west-1`). This is acceptable since the model is configurable, but worth noting.
- **Recommendation:** Document in the design doc that the default model uses cross-region inference and may need to be overridden for non-US regions.

**L-2: Test coverage for Bedrock path is structural only**
- File: `/mcp-server-compose/rag-mcp-server/internal/hyde/hyde_test.go` (lines 80-98)
- `TestNewBedrockGenerator_DefaultModel` and `TestNewBedrockGenerator_CustomModel` verify struct field assignment but do not test the `Generate()` path for Bedrock. This is understandable since Bedrock requires AWS credentials, but a test with a mock HTTP backend (similar to `TestClaudeGenerator_ExtractsText`) would strengthen coverage.
- **Recommendation:** Add a test that uses `httptest.Server` to simulate the Bedrock response format, or note in the test file that Bedrock integration is covered by integration tests.

**L-3: Scripts `eval.sh` and `search.sh` use `aws bedrock-runtime invoke-model` with `--body` flag passing JSON directly**
- Files: `/mcp-server-compose/rag-mcp-server/scripts/eval.sh` (line 189), `/mcp-server-compose/rag-mcp-server/scripts/search.sh` (line 233)
- The `--body` flag passes the JSON request directly as a command argument. For very large context payloads this could exceed shell argument length limits. This is low risk since HyDE prompts and eval requests are small, but the RAG answer path in `search.sh` could have larger context blocks.
- **Recommendation:** Consider using `file://` body passing for robustness (`--body "file:///dev/stdin"` with pipe).

---

## Config Flow Verification

The config pipeline was verified across both layers:

1. **stack.toml** -> `[rag_mcp_server.hyde]` section includes `provider`, `model`, `base_url`, `aws_region`, `system_prompt` fields. Correctly parsed by stack's `internal/config/config.go` into `HyDEConfig` struct.

2. **stack Validate()** -> Correctly validates: bedrock requires `aws_region`; anthropic requires `ANTHROPIC_API_KEY` (read from env). Bedrock path correctly skips the API key requirement.

3. **stack generate** -> `ragConfigTOML()` in `internal/generate/generate.go` correctly writes all hyde fields including `provider` and `aws_region` to the generated `config.toml`. Provider defaults to `"anthropic"` if empty.

4. **rag-mcp-server config.Load()** -> Correctly reads `provider` and `aws_region` from `[hyde]` TOML section. Validates provider enum. Requires `aws_region` when provider is bedrock and hyde is enabled.

5. **rag-mcp-server main.go** -> Correctly branches on provider: `"bedrock"` creates `NewBedrockGenerator`, default creates `NewClaudeGenerator`. API key is only required for the Anthropic path.

6. **Secrets flow** -> `ANTHROPIC_API_KEY` is read from environment (not config file) in both layers. Stack writes it to `.env` files only. Bedrock uses the AWS credential chain (env vars, `~/.aws/`, IAM role) -- no AWS credentials in config files. This is correct.

---

## Interface Compliance

- `ClaudeGenerator` (used for both Anthropic and Bedrock) implements `hyde.Generator` interface correctly. The `Generate()` method is identical for both paths -- the SDK handles the routing difference via the client options set at construction.
- The `bedrock` sub-package import (`github.com/anthropics/anthropic-sdk-go/bedrock`) and `awsconfig` import (`github.com/aws/aws-sdk-go-v2/config`) are properly declared in `go.mod`.
- Data flow follows existing patterns: config -> constructor -> interface -> tool handler.

---

## Error Handling Compliance

- Bedrock API errors propagate through the same `Generate()` error path as Anthropic errors. The tool handler in `search.go` logs WARN and falls back to raw query, consistent with DESIGN.md section 13.
- Config validation errors are descriptive and follow the `fmt.Errorf("config: ...")` pattern used throughout.
- Missing `aws_region` for Bedrock is caught at config load time (both layers), preventing a runtime surprise.

---

## Credential Leakage Check

- PASS: No AWS credentials appear in any config file, TOML template, or generated output.
- PASS: `ANTHROPIC_API_KEY` is only in `.env` files (mode 0600) and environment variables, never in `config.toml`.
- PASS: Bedrock uses the standard AWS credential chain -- no explicit key/secret handling in application code.
- PASS: The `redactURL()` function in `main.go` strips userinfo from URLs before logging.

---

## Code Quality

- Naming follows Go conventions. `HyDEProvider`, `HyDEAWSRegion` use consistent casing.
- No magic values: `hydeMaxTokens = 256` is a named constant. Default models are string literals in constructors, which is acceptable for SDK model identifiers.
- The Bedrock default model `"us.anthropic.claude-haiku-4-5-20251001-v1:0"` is a distinct constant from the Anthropic default `"claude-haiku-4-5-20251001"` -- correct, as Bedrock uses different model IDs.
- Tests cover both default and custom model paths, provider validation, and the bedrock-requires-region rule.
