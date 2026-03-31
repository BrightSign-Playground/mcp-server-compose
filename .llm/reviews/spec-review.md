# Spec Compliance Review: Bedrock Implementation

**Spec document:** `/mcp-server-compose/bedrock.md`
**Review date:** 2026-03-31

## Summary

The implementation is largely compliant with the specification across all three phases. Several minor deviations and one medium-severity gap are noted below.

---

## Phase 1: Go SDK (HyDE)

### Requirement 1.1: Add `provider` field to `HyDEConfig`

**Status: PASS**

- `HyDEConfig` in both `/mcp-server-compose/rag-mcp-server/internal/config/config.go` (line 48: `HyDEProvider string`) and `/mcp-server-compose/internal/config/config.go` (line 102: `Provider string`) include the provider field.
- TOML tag `provider` is present in both `hydeFileCfg` and the stack-level `fileConfig`.
- Default value is `"anthropic"` in `builtinDefaults()` (rag-mcp-server config.go line 143).

### Requirement 1.2: Branch on provider in `NewClaudeGenerator()` area

**Status: PASS**

- `/mcp-server-compose/rag-mcp-server/internal/hyde/claude.go` provides two constructors: `NewClaudeGenerator` (lines 27-47) for Anthropic direct API and `NewBedrockGenerator` (lines 52-71) for Bedrock.
- `NewBedrockGenerator` uses `bedrock.WithLoadDefaultConfig` with `awsconfig.WithRegion()` as specified.
- `/mcp-server-compose/rag-mcp-server/cmd/server/main.go` branches on `cfg.HyDEProvider` at line 74: `case "bedrock"` calls `NewBedrockGenerator`, `default` calls `NewClaudeGenerator`.
- Default Bedrock model is `us.anthropic.claude-haiku-4-5-20251001-v1:0` (line 54), matching the spec's model ID mapping table.

### Requirement 1.3: Update `config.Validate()` -- bedrock does not need `ANTHROPIC_API_KEY`, needs `aws_region`

**Status: PASS**

- **rag-mcp-server level:** `config.go` lines 236-245:
  - Provider is validated to be `"anthropic"` or `"bedrock"` (line 236-239).
  - `aws_region` is required when provider is `"bedrock"` and HyDE is enabled (line 243).
  - `ANTHROPIC_API_KEY` check is in `main.go` (line 79) and only triggers for the `default` (anthropic) case.
- **Stack level:** `/mcp-server-compose/internal/config/config.go` lines 350-357:
  - HyDE with `provider != "bedrock"` requires `AnthropicAPIKey` (line 350).
  - Bedrock requires `aws_region` (line 355-357).
- Tests exist for all validation paths:
  - `TestLoad_InvalidHyDEProvider`, `TestLoad_BedrockRequiresAWSRegion`, `TestLoad_BedrockWithRegion` in rag-mcp-server config_test.go.
  - `TestValidate_hydeBedrockNoKeyRequired`, `TestValidate_hydeBedrockRequiresRegion` in stack config_test.go.

### Requirement 1.4: Update `stack.toml.example` and docs

**Status: PASS**

- `stack.toml.example` lines 96-102 include the full `[rag_mcp_server.hyde]` section with `provider`, `aws_region`, and inline comments explaining valid values.

---

## Phase 2: Eval/search scripts

### Requirement 2.1: Replace curl calls with `aws bedrock-runtime invoke-model` behind a provider flag

**Status: PASS**

- **eval.sh:** Introduces `ANTHROPIC_PROVIDER` env var (default `"anthropic"`, line 100), `AWS_REGION` (default `"us-east-1"`, line 101). The `call_claude()` function (lines 175-198) branches on provider:
  - `"bedrock"`: uses `aws bedrock-runtime invoke-model`, strips model from body, adds `anthropic_version: "bedrock-2023-05-31"`.
  - `"anthropic"`: uses existing curl against `api.anthropic.com`.
- Preflight checks validate `ANTHROPIC_API_KEY` is only required for `anthropic` provider (line 116) and `aws` CLI is required for `bedrock` (line 121-124).
- **search.sh:** Same pattern -- `ANTHROPIC_PROVIDER` (line 75), `AWS_REGION` (line 76). The `--llm` Bedrock path (lines 209-236) constructs the request with `anthropic_version: "bedrock-2023-05-31"`, strips model, calls `aws bedrock-runtime invoke-model`. Preflight checks at lines 79-87.

### Finding 2.1: search.sh Bedrock path does not stream

**Severity: LOW**

The spec does not explicitly require streaming for Bedrock, and this is a developer tool. The Anthropic path streams via SSE (lines 256-269) while the Bedrock path returns a single response (lines 209-236). This is an acceptable divergence since `aws bedrock-runtime invoke-model` does not natively support SSE streaming in the same way, and the spec offered `aws bedrock-runtime invoke-model` as the recommended approach.

### Finding 2.2: eval.sh Bedrock model ID not remapped

**Severity: LOW**

The `EVAL_MODEL` default is `claude-haiku-4-5-20251001` (line 99), which is the direct API model name. When `ANTHROPIC_PROVIDER=bedrock`, the user must manually set `EVAL_MODEL` to the Bedrock model ID (e.g., `us.anthropic.claude-haiku-4-5-20251001-v1:0`). The spec's model ID mapping table (bedrock.md lines 48-53) shows these differ. There is no automatic mapping or documented guidance in the script's usage text about which model ID to use for Bedrock.

The usage text at line 69 says `EVAL_MODEL` is for "Claude model for answer + judge" but does not mention that Bedrock requires a different model ID format.

---

## Phase 3: Config generation -- stack CLI

### Requirement 3.1: Add `aws_region` to the `[rag_mcp_server.hyde]` TOML section

**Status: PASS**

- `/mcp-server-compose/internal/generate/generate.go` line 234: `ragConfigTOML` emits `aws_region` in the `[hyde]` block.
- The `HyDEConfig` struct at stack level (line 100-107 of stack config.go) includes `AWSRegion`.
- The TOML file config struct has the `aws_region` tag (line 201).

### Requirement 3.2: Generate the appropriate config.toml fields

**Status: PASS**

- `ragConfigTOML()` (generate.go lines 197-241) emits all HyDE fields including `provider`, `model`, `base_url`, `aws_region`, and `system_prompt`.
- Default provider is set to `"anthropic"` if empty (line 231-233).

### Requirement 3.3: When provider is "bedrock", pass AWS credentials through to the rag-mcp-server `.env`

**Status: MEDIUM -- NOT IMPLEMENTED**

The spec (bedrock.md lines 135-137) states:

> When provider is `"bedrock"`, pass `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN` through to the rag-mcp-server `.env` (or rely on IAM roles and pass nothing).

The `ragEnvVars()` function (generate.go lines 174-187) only passes `DATABASE_URL`, `MCP_PORT`, and optionally `ANTHROPIC_API_KEY`. It does not conditionally pass AWS credential environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`) when the provider is `"bedrock"`.

The spec does allow "or rely on IAM roles and pass nothing" as an alternative, but this is implicit -- there is no documented decision or code comment explaining that IAM roles are the expected credential mechanism for containerized deployments. Without the env var pass-through, users running in environments where AWS credentials come from environment variables (not IAM roles) will have Bedrock authentication failures in the containerized rag-mcp-server.

---

## Cross-cutting findings

### Finding CC.1: CLAUDE.md config table not updated with provider field

**Severity: LOW**

The `CLAUDE.md` requirements doc for rag-mcp-server (the `[hyde]` row in the config.toml schema table) lists fields as `enabled, model, base_url, system_prompt` but does not include `provider` or `aws_region`. This is a documentation-only gap. The actual code and config.toml are correct.

### Finding CC.2: Bedrock model default consistency

**Severity: LOW**

The default model for Bedrock in `NewBedrockGenerator` is `us.anthropic.claude-haiku-4-5-20251001-v1:0`, while the default model for Anthropic direct in `NewClaudeGenerator` is `claude-haiku-4-5-20251001`. The `builtinDefaults()` in rag-mcp-server config.go sets the default model to `claude-haiku-4-5-20251001` (the Anthropic format) regardless of provider. This means if a user sets `provider = "bedrock"` without changing the model, the config will pass `claude-haiku-4-5-20251001` to `NewBedrockGenerator`, which then overrides it to the Bedrock format only if the model string is empty. Since the config default is non-empty (`claude-haiku-4-5-20251001`), this non-Bedrock model ID would be used, likely causing API errors.

This is mitigated by the fact that `NewBedrockGenerator` only defaults when model is empty (line 53-55 of claude.go), and the stack-level config does not set a default model. But at the rag-mcp-server level, `builtinDefaults()` always populates `Model: "claude-haiku-4-5-20251001"` -- a user switching to bedrock provider without also changing the model will get a Bedrock API error.

---

## Findings Summary

| ID | Severity | Description |
|----|----------|-------------|
| 3.3 | **MEDIUM** | AWS credential env vars (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`) are not passed through to rag-mcp-server `.env` in `generate.go` when provider is "bedrock". The spec allows relying on IAM roles, but this is undocumented. |
| CC.2 | **LOW** | `builtinDefaults()` sets HyDE model to Anthropic format (`claude-haiku-4-5-20251001`) which is invalid for Bedrock. Users switching to bedrock provider without changing the model will get API errors. |
| 2.2 | **LOW** | eval.sh default `EVAL_MODEL` uses Anthropic model ID format; no guidance for Bedrock model IDs in usage text. |
| 2.1 | **LOW** | search.sh Bedrock path does not stream (acceptable given `aws bedrock-runtime invoke-model` limitations). |
| CC.1 | **LOW** | CLAUDE.md config table does not list `provider` or `aws_region` for `[hyde]` section. |
