# Amazon Bedrock as an Alternative to Direct Anthropic API

## Current State

The codebase uses the Anthropic API in three places, all optional:

1. **HyDE (Hypothetical Document Embeddings)** — opt-in query expansion in rag-mcp-server.
   Uses the `anthropic-sdk-go` library to call Claude via the Messages API.
   - `rag-mcp-server/internal/hyde/claude.go` — Go SDK client
   - `rag-mcp-server/cmd/server/main.go` — reads `ANTHROPIC_API_KEY` from env
   - Default model: `claude-haiku-4-5-20251001`

2. **Eval harness** (`rag-mcp-server/scripts/eval.sh`) — shell script that calls
   `https://api.anthropic.com/v1/messages` directly via curl for LLM-judged evaluation.

3. **Search helper** (`rag-mcp-server/scripts/search.sh`) — shell script that calls
   the same endpoint for optional LLM answer generation.

All three are disabled or unused in normal operation. HyDE is off by default
(`hyde.enabled = false`). The scripts are developer tools, not part of the
running service.

## What Would Change for Bedrock

### Go SDK (HyDE)

The `anthropic-sdk-go` library already supports Bedrock natively. The change
is small — swap the client constructor:

```go
// Current: direct Anthropic API
client := anthropic.NewClient(option.WithAPIKey(apiKey))

// Bedrock alternative
import "github.com/anthropics/anthropic-sdk-go/bedrock"

client := anthropic.NewClient(
    bedrock.WithAWSRegion("us-east-1"),
    // Uses default AWS credential chain (env vars, ~/.aws, IAM role, etc.)
)
```

No changes to the `Messages.New()` call itself — the SDK abstracts the
endpoint differences.

**Model ID mapping:** Bedrock uses different model identifiers:

| Direct API                    | Bedrock model ID                                   |
|-------------------------------|-----------------------------------------------------|
| `claude-haiku-4-5-20251001`   | `us.anthropic.claude-haiku-4-5-20251001-v1:0`       |
| `claude-sonnet-4-6-20250514`  | `us.anthropic.claude-sonnet-4-6-20250514-v1:0`      |
| `claude-opus-4-6-20250514`    | `us.anthropic.claude-opus-4-6-20250514-v1:0`        |

The model name in `config.toml` would need to use the Bedrock variant.

### Shell Scripts (eval.sh, search.sh)

These use raw curl against `api.anthropic.com`. Bedrock requires AWS Signature
Version 4 authentication, which curl cannot do natively. Options:

1. **Use `aws` CLI** — `aws bedrock-runtime invoke-model` handles signing.
   Requires restructuring the JSON payload slightly (Bedrock wraps the
   Messages API format).

2. **Use a signing proxy** — run a local proxy that adds SigV4 to requests
   and forwards to Bedrock. The scripts stay mostly unchanged.

3. **Rewrite scripts in Python/Go** — use the AWS SDK directly. More work
   but cleaner.

### Configuration Changes

**New config fields** (stack.toml / rag-mcp-server config.toml):

```toml
[rag_mcp_server.hyde]
enabled = true
provider = "bedrock"        # "anthropic" (default) or "bedrock"
model = "us.anthropic.claude-haiku-4-5-20251001-v1:0"
aws_region = "us-east-1"    # required for bedrock
system_prompt = ""
```

**Environment variables:**

| Direct API            | Bedrock                                              |
|-----------------------|------------------------------------------------------|
| `ANTHROPIC_API_KEY`   | `AWS_ACCESS_KEY_ID` + `AWS_SECRET_ACCESS_KEY`        |
|                       | (or IAM role / instance profile — no env vars needed)|

On EC2/ECS/EKS with an instance role or IRSA, no credentials need to be
configured at all — the SDK uses the default credential chain.

### AWS Prerequisites

1. **Model access** — Bedrock requires explicit model enablement per region.
   Go to the Bedrock console → Model access → Request access for the Claude
   models you need.

2. **IAM permissions** — the calling identity needs:
   ```json
   {
     "Effect": "Allow",
     "Action": "bedrock:InvokeModel",
     "Resource": "arn:aws:bedrock:*::foundation-model/anthropic.*"
   }
   ```

3. **Region availability** — not all Claude models are available in all
   regions. `us-east-1` and `us-west-2` have the broadest availability.

## Implementation Plan

### Phase 1: Go SDK (HyDE) — small change

1. Add a `provider` field to `HyDEConfig` (`"anthropic"` | `"bedrock"`).
2. In `hyde.NewClaudeGenerator()`, branch on provider:
   - `"anthropic"` — existing code (API key auth).
   - `"bedrock"` — use `bedrock.WithAWSRegion()` constructor.
3. Update `config.Validate()` — when `provider = "bedrock"`, do not require
   `ANTHROPIC_API_KEY`; optionally validate `aws_region` is set.
4. Update stack.toml.example and docs.

### Phase 2: Eval/search scripts — medium change

1. Replace curl calls with `aws bedrock-runtime invoke-model` behind a
   provider flag.
2. Or rewrite the LLM-calling portions in Go (a small CLI tool) that
   supports both providers via the same SDK.

### Phase 3: Config generation — stack CLI

1. Add `aws_region` to the `[rag_mcp_server.hyde]` TOML section.
2. Generate the appropriate config.toml fields.
3. When provider is `"bedrock"`, pass `AWS_ACCESS_KEY_ID` /
   `AWS_SECRET_ACCESS_KEY` / `AWS_SESSION_TOKEN` through to the
   rag-mcp-server `.env` (or rely on IAM roles and pass nothing).

## Effort Estimate

| Component              | Scope         | Notes                                    |
|------------------------|---------------|------------------------------------------|
| Go SDK swap (HyDE)     | ~50 lines     | SDK already supports Bedrock natively     |
| Config plumbing        | ~30 lines     | New provider field, region, validation    |
| Eval script            | ~100 lines    | Replace curl with aws CLI or Go tool      |
| Search script          | ~50 lines     | Same approach as eval                     |
| Documentation          | ~1 page       | Config examples, IAM setup                |

The Go SDK change is straightforward because `anthropic-sdk-go` treats
Bedrock as a transport option, not a different API. The scripts are the
messier part because Bedrock's auth model (SigV4) does not work with
plain curl.
