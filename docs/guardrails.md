# Guardrails

The search pipeline implements two opt-in guardrail levels that reject irrelevant
queries before they consume database resources (Level 1) or filter out low-confidence
results before they reach the calling agent (Level 2). Both are disabled by default
and impose zero overhead when unused.

---

## Configuration

### Enabling Level 1 -- Topic Relevance

Level 1 rejects queries that are unrelated to the corpus topic before the database
is ever queried. To enable it, set `corpus_topic` to a one-sentence description of
what the corpus covers:

```toml
[guardrails]
corpus_topic    = "Mystery and crime fiction: plot summaries, characters, themes, literary analysis."
min_topic_score = 0.25
```

At startup the server embeds `corpus_topic` via llama-server and stores the
L2-normalized result as the topic vector. Each incoming query vector is compared
against it using cosine similarity. Queries scoring below `min_topic_score` receive
an `off_topic` error immediately.

Leave `corpus_topic` empty (the default) to disable Level 1 entirely.

### Enabling Level 2 -- Match Quality

Level 2 checks the best result's cosine similarity after search (and reranking, if
enabled). Set `min_match_score` to a value greater than zero:

```toml
[guardrails]
min_match_score = 0.40
```

When the top result scores below this threshold the tool returns a `below_threshold`
error that includes the actual best score observed. Set to `0.0` (the default) to
disable.

### Full Config Example

```toml
[guardrails]
# Level 1 -- topic relevance (disabled when empty)
corpus_topic    = "Kubernetes operations: cluster management, networking, storage, observability."
min_topic_score = 0.25

# Level 2 -- match quality (disabled when 0.0)
min_match_score = 0.40
```

### Tuning Against Evals

1. Set `log_level = "debug"` in `[server]` to expose per-query scores:

   ```json
   {"level":"DEBUG","msg":"topic check","score":0.32,"threshold":0.25,"action":"passed"}
   {"level":"DEBUG","msg":"match quality check","best_score":0.71,"threshold":0.40,"action":"passed"}
   ```

2. Run your eval suite and collect the score distributions for on-topic queries,
   off-topic queries, and known-good matches.

3. Adjust thresholds based on the distributions:

   | Parameter | Starting point | Typical range | Direction |
   |---|---|---|---|
   | `min_topic_score` | 0.25 | 0.15 -- 0.40 | Lower = more permissive |
   | `min_match_score` | 0.40 | 0.30 -- 0.70 | Higher = stricter |

4. Re-run evals after each adjustment. Watch for false rejections on edge-case
   queries that are valid but phrased unusually.

### Validation Rules

- `min_topic_score` must be in `[0.0, 1.0]`. Out-of-range values cause a startup
  error.
- `min_match_score` must be in `[0.0, 1.0]`. Out-of-range values cause a startup
  error.
- If `corpus_topic` is non-empty but the embedding call fails at startup, the server
  logs `WARN` and disables Level 1 (startup is not blocked by a transient
  llama-server failure).
- An empty `corpus_topic` with a non-zero `min_topic_score` is valid -- the threshold
  is simply unused.

---

## How It Works

### Struct Layout

```go
// tools/search.go
type GuardrailConfig struct {
    TopicVector   []float32  // nil = Level 1 disabled
    MinTopicScore float64    // cosine similarity threshold [0, 1]
    MinMatchScore float64    // 0 = Level 2 disabled
}
```

`TopicVector` is populated at startup only when `corpus_topic` is non-empty and the
embedding succeeds. `MinTopicScore` and `MinMatchScore` are copied directly from the
parsed config. The struct is passed by value into `RegisterSearchTool`, so the
guardrail state is immutable after initialization.

### Zero-Overhead Pattern

Both levels use a cheap runtime check that short-circuits when the guardrail is
disabled:

```go
// Level 1: len() on nil slice is 0 -- no allocation, no computation
if len(guardrails.TopicVector) > 0 {
    score := vecmath.DotProduct(vector, guardrails.TopicVector)
    if score < guardrails.MinTopicScore { /* reject */ }
}

// Level 2: float comparison against zero -- single branch
if guardrails.MinMatchScore > 0 && len(merged) > 0 {
    if merged[0].Score < guardrails.MinMatchScore { /* reject */ }
}
```

When disabled, each level adds exactly one branch instruction to the hot path. There
is no interface dispatch, no map lookup, no allocation.

### Pipeline Diagram

```
POST /mcp  (with Bearer JWT)
  |
  v
Auth Middleware ── 401 if invalid JWT
  |
  v
Input Validation ── reject empty/oversized query, bad limit
  |
  v
HyDE Expansion (optional) ── generate hypothesis passage
  |
  v
Embed Query ── POST llama-server /v1/embeddings
  |
  v
L2-Normalize query vector
  |
  v
┌─────────────────────────────────────────┐
│ GUARDRAIL LEVEL 1 -- Topic Relevance    │
│ dot_product(queryVec, topicVec)          │
│ score < min_topic_score? → off_topic    │
│ Cost: ~768 multiply-adds (one dot prod) │
│ Skipped entirely if TopicVector is nil  │
└─────────────────────────────────────────┘
  |
  v
Count chunks ── return no_data if DB is empty
  |
  v
Hybrid Search (parallel)
  ├── Vector KNN (pgvector <=>)
  └── Full-Text Search (PostgreSQL tsvector)
  |
  v
RRF Merge ── Reciprocal Rank Fusion
  |
  v
Reranker (optional) ── cross-encoder re-scoring
  |
  v
┌─────────────────────────────────────────┐
│ GUARDRAIL LEVEL 2 -- Match Quality      │
│ merged[0].Score < min_match_score?      │
│ → below_threshold (includes best_score) │
│ Cost: one float comparison              │
│ Skipped entirely if min_match_score = 0 │
└─────────────────────────────────────────┘
  |
  v
Apply limit, return results
```

### Startup Behavior

1. `config.Load()` reads `[guardrails]` from `config.toml` and validates ranges.
2. `main()` copies `MinTopicScore` and `MinMatchScore` into a `GuardrailConfig`.
3. If `corpus_topic` is non-empty:
   - Embed the topic string via llama-server.
   - On success: L2-normalize the vector and store it in `TopicVector`.
   - On failure: log `WARN "topic embedding failed; Level 1 guardrail disabled"` and
     continue. The server starts with Level 1 disabled.
4. If `MinMatchScore > 0`: log `INFO` confirming Level 2 is enabled.
5. The `GuardrailConfig` is passed into `RegisterSearchTool` and is immutable from
   this point forward.

This design ensures that a transient llama-server outage at boot time does not
prevent the server from starting -- it simply operates without topic filtering until
restarted.

### Scoring Math

Both levels use cosine similarity, computed as the dot product of two L2-normalized
vectors:

```
L2Normalize(v):
    magnitude = sqrt( sum( v[i]^2 ) )
    if magnitude > 0:
        v[i] = v[i] / magnitude

CosineSimilarity(a, b) = DotProduct(L2Normalize(a), L2Normalize(b))
                       = sum( a[i] * b[i] )
```

All vectors (corpus topic, query, and stored chunk embeddings) are L2-normalized.
The resulting scores fall in `[0, 1]` for typical embedding models (non-negative
components). A score of 1.0 means identical direction; 0.0 means orthogonal.

The database uses the same metric via pgvector's cosine distance operator:

```sql
1 - (c.embedding <=> $1) AS score
```

Level 1 computes the dot product in Go (single pass over ~768 floats). Level 2
reads the score that was already computed by the database or reranker -- no
additional vector math.

### Error Propagation

Guardrail rejections are returned as MCP tool errors (`IsError: true`) with
structured error codes:

| Error code | Level | Trigger | Includes |
|---|---|---|---|
| `off_topic` | 1 | `dot_product < min_topic_score` | -- |
| `below_threshold` | 2 | `top_score < min_match_score` | `best_score` value |

Both are returned with HTTP 200 (MCP protocol convention -- tool errors are not
HTTP errors). The calling agent receives the error code and can decide how to
handle it (retry with a different query, inform the user, etc.).

Example error payloads:

```json
{
  "content": [{"type": "text", "text": "off_topic: query does not appear to be related to the supported topic area"}],
  "isError": true
}
```

```json
{
  "content": [{"type": "text", "text": "below_threshold: no content found that is sufficiently relevant to this query (best_score: 0.210)"}],
  "isError": true
}
```

### Extension Points for New Guards

The guardrail system is designed for straightforward extension. To add a new guard:

1. **Add fields to `GuardrailConfig`** (`internal/tools/search.go`):

   ```go
   type GuardrailConfig struct {
       TopicVector   []float32
       MinTopicScore float64
       MinMatchScore float64
       // New guard:
       SafetyVector  []float32  // nil = disabled
       MaxSafetyScore float64   // reject if similarity exceeds this
   }
   ```

2. **Add config fields** (`internal/config/config.go`):

   ```go
   type guardrailsFileCfg struct {
       CorpusTopic    string  `toml:"corpus_topic"`
       MinTopicScore  float64 `toml:"min_topic_score"`
       MinMatchScore  float64 `toml:"min_match_score"`
       SafetyTopic    string  `toml:"safety_topic"`
       MaxSafetyScore float64 `toml:"max_safety_score"`
   }
   ```

   Add validation in `Load()` alongside the existing range checks.

3. **Initialize at startup** (`cmd/server/main.go`):

   ```go
   if cfg.SafetyTopic != "" {
       vec, err := embedder.Embed(context.Background(), cfg.SafetyTopic)
       if err != nil {
           logger.Warn("safety embedding failed; safety guardrail disabled", "error", err)
       } else {
           vecmath.L2Normalize(vec)
           guardrails.SafetyVector = vec
       }
   }
   ```

4. **Insert the check in the pipeline** (`internal/tools/search.go`), following the
   same pattern as Level 1:

   ```go
   if len(guardrails.SafetyVector) > 0 {
       score := vecmath.DotProduct(vector, guardrails.SafetyVector)
       if score > guardrails.MaxSafetyScore {
           logger.Debug("safety check", "score", score, "threshold", guardrails.MaxSafetyScore, "action", "rejected")
           return toolError("unsafe_content", "query flagged by safety filter"), SearchResult{}, nil
       }
   }
   ```

The key invariants to preserve:

- **Disabled = zero cost.** Gate on `len(vec) > 0` or `threshold > 0`.
- **Graceful startup.** Embedding failures log `WARN` and disable the guard; they
  never prevent the server from starting.
- **Unique error code.** Each guard returns a distinct code so agents can
  differentiate rejection reasons.
- **DEBUG logging.** Log the score, threshold, and action for every check so
  operators can tune thresholds against eval data.
