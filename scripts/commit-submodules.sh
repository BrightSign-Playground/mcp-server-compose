#!/usr/bin/env bash
# Commit and push all submodule changes, then update the parent repo.
#
# Each submodule is in detached HEAD state (normal for submodules).
# This script creates a branch from the detached HEAD with uncommitted
# changes, then fast-forward merges it into main.
#
# Usage:
#   ./scripts/commit-submodules.sh [--dry-run]

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DRY_RUN=false
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN=true

run() {
    echo "  $ $*"
    if [[ "${DRY_RUN}" == "false" ]]; then
        "$@"
    fi
}

die() { echo "ERROR: $*" >&2; exit 1; }

# Helper: commit changes in a submodule.
# Args: submodule_dir, commit_message, files_to_add...
commit_submodule() {
    local dir="$1"; shift
    local msg="$1"; shift
    local files=("$@")

    echo ""
    echo "━━━ $(basename "${dir}") ━━━"
    cd "${dir}"

    # Check if any of the listed files have changes
    local has_changes=false
    for f in "${files[@]}"; do
        if ! git diff --quiet -- "${f}" 2>/dev/null; then
            has_changes=true
            break
        fi
    done

    if [[ "${has_changes}" == "false" ]]; then
        echo "  no changes to commit"
        return 0
    fi

    # Stage the files
    run git add "${files[@]}"

    # Create a temporary branch from the current detached HEAD + staged changes
    local tmp_branch="auto-commit-$(date +%s)"
    run git checkout -b "${tmp_branch}"
    run git commit -m "${msg}"

    # Switch to main and fast-forward merge
    run git checkout main
    run git merge --ff-only "${tmp_branch}"
    run git branch -d "${tmp_branch}"

    # Push
    run git push origin main

    echo "  done"
}

# ── Preflight ────────────────────────────────────────────────────────────────
cd "${REPO_ROOT}"
parent_branch=$(git rev-parse --abbrev-ref HEAD)
[[ "${parent_branch}" == "main" ]] || die "parent repo is on '${parent_branch}', expected 'main'"

if [[ "${DRY_RUN}" == "true" ]]; then
    echo "[dry-run mode — no changes will be made]"
    echo ""
fi

# ── Discard artifacts that shouldn't be committed ────────────────────────────
echo "cleaning up build artifacts..."
git -C docs2vector checkout -- ingest 2>/dev/null || true
git -C rag-mcp-server checkout -- config.toml 2>/dev/null || true

# ── docs2vector ──────────────────────────────────────────────────────────────
commit_submodule "${REPO_ROOT}/docs2vector" \
    "$(cat <<'EOF'
feat: read embed_dim from config instead of hardcoded constant

- Remove const EmbeddingDimension = 1024 from store.go
- NewPostgresStore accepts embedDim parameter; DDL uses fmt.Sprintf
- Add EmbedDim to Config struct with TOML parsing and --embed-dim flag
- Add validation: embed_dim must be positive
- Update CLAUDE.md to reflect dynamic dimension
- Change default chunk_size from 256 to 512

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)" \
    CLAUDE.md \
    cmd/ingest/main.go \
    internal/config/config.go \
    internal/store/postgres.go \
    internal/store/store.go

# ── keycloak-testing ─────────────────────────────────────────────────────────
commit_submodule "${REPO_ROOT}/keycloak-testing" \
    "$(cat <<'EOF'
fix: join stack-net network, add JVM container-awareness flags

- Add stack-net external network to all services so rag-mcp-server
  can resolve keycloak hostname for JWKS fetching
- Add JAVA_OPTS_APPEND with UseContainerSupport and MaxRAMPercentage=70
  to prevent JVM OOM kills within the 2GB container limit
- Sync CLAUDE.md with actual compose.yml (memory 2g/1g, retries 15,
  docker.io image prefix, KC_PORT in init env)

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)" \
    CLAUDE.md \
    compose.yml

# ── rag-mcp-server ───────────────────────────────────────────────────────────
commit_submodule "${REPO_ROOT}/rag-mcp-server" \
    "$(cat <<'EOF'
feat: join stack-net, improve eval reporting, remove book dead code

- Add stack-net external network so server can reach keycloak/logto
- Make host port configurable via MCP_PORT env var
- Eval summary now shows pass percentage and per-label breakdown
  (good/answerable vs bad/fabricated)
- Remove all book-related filtering, counters, and display code

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)" \
    compose.yaml \
    scripts/eval.sh

# ── Parent repo: update submodule pointers ───────────────────────────────────
echo ""
echo "━━━ parent (mcp-server-compose) ━━━"
cd "${REPO_ROOT}"

run git add docs2vector keycloak-testing rag-mcp-server

if git diff --cached --quiet 2>/dev/null; then
    echo "  submodule pointers already up to date"
else
    run git commit -m "$(cat <<'EOF'
chore: update submodule refs

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
    run git push origin main
    echo "  done"
fi

echo ""
echo "all submodules committed and pushed."
