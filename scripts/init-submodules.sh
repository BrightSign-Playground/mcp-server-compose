#!/usr/bin/env bash
# init-submodules.sh
# Adds component directories as git submodules and checks out their contents.
# Run this from the root of the mcp-servers repo after it has been initialised
# with `git init` and has at least one commit. Move existing directories out of
# the way manually before running.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

# Associative array: directory name -> remote URL
declare -A REMOTES=(
  [bs-ai-support-model]="git@github.com:BrightSign-Playground/bs-ai-support-model.git"
  [docs2vector]="git@github.com:BrightSign-Playground/docs2vector.git"
  [keycloak-testing]="git@github.com:BrightSign-Playground/keycloak-testing.git"
  [llama.cpp]="ssh://git@github.com/ggml-org/llama.cpp"
  [logto-testing]="git@github.com:BrightSign-Playground/logto-testing.git"
  [rag-mcp-server]="git@github.com:BrightSign-Playground/rag-mcp-server.git"
)

# llama.cpp is a very large repo; clone it shallow by default.
# Set to empty string to do a full clone.
LLAMA_SHALLOW_DEPTH=1

add_submodule() {
  local name="$1"
  local url="$2"

  if git config --file .gitmodules "submodule.${name}.url" &>/dev/null; then
    echo "  submodule '${name}' already registered — skipping git submodule add"
  else
    if [[ "$name" == "llama.cpp" && -n "${LLAMA_SHALLOW_DEPTH}" ]]; then
      echo "  adding submodule ${name} (shallow depth ${LLAMA_SHALLOW_DEPTH})"
      git submodule add --depth "${LLAMA_SHALLOW_DEPTH}" -- "$url" "$name"
    else
      echo "  adding submodule ${name}"
      git submodule add -- "$url" "$name"
    fi
  fi
}

echo "==> Adding submodules"
for name in "${!REMOTES[@]}"; do
  url="${REMOTES[$name]}"
  add_submodule "$name" "$url"
done

echo ""
echo "==> Initialising and updating submodules"
git submodule update --init --recursive

echo ""
echo "==> Done. Submodule status:"
git submodule status

echo ""
echo "Next steps:"
echo "  1. Review the changes: git status && git diff --cached"
echo "  2. Commit:             git add .gitmodules <submodule-dirs> && git commit"
