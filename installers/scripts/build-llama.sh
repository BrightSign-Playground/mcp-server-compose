#!/usr/bin/env bash
# build-llama.sh — Build llama-server from the llama.cpp submodule.
# Detects Metal (macOS), CUDA (Linux), or falls back to CPU.
# Installs the resulting binary to /usr/local/bin/llama-server.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
LLAMA_DIR="${REPO_ROOT}/llama.cpp"
BUILD_DIR="${LLAMA_DIR}/build"

die() { echo "Error: $*" >&2; exit 1; }

ncpu() {
    nproc 2>/dev/null || sysctl -n hw.logicalcpu 2>/dev/null || echo 4
}

[[ -f "${LLAMA_DIR}/CMakeLists.txt" ]] \
    || die "llama.cpp submodule not initialised — run: make submodules"

# ── Install build dependencies ────────────────────────────────────────────────

install_deps_mac() {
    command -v cmake &>/dev/null && return
    echo "==> Installing cmake via Homebrew"
    command -v brew &>/dev/null || die "Homebrew is required. Install from https://brew.sh"
    brew install cmake
}

install_deps_apt() {
    echo "==> Installing build dependencies via apt"
    sudo apt-get update -qq
    sudo apt-get install -y cmake build-essential
}

install_deps_dnf() {
    echo "==> Installing build dependencies via dnf"
    sudo dnf install -y cmake gcc-c++ make
}

case "$(uname -s)" in
    Darwin) install_deps_mac ;;
    Linux)
        if command -v apt-get &>/dev/null; then install_deps_apt
        elif command -v dnf &>/dev/null; then install_deps_dnf
        fi
        ;;
esac

# ── Detect GPU and select cmake flags ─────────────────────────────────────────

CMAKE_FLAGS="-DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF"

case "$(uname -s)" in
    Darwin)
        # Metal is the GPU backend on macOS; enabled by default on Apple Silicon.
        CMAKE_FLAGS="${CMAKE_FLAGS} -DGGML_METAL=ON"
        echo "==> GPU backend: Metal"
        ;;
    Linux)
        if command -v nvcc &>/dev/null || [[ -d /usr/local/cuda ]]; then
            CMAKE_FLAGS="${CMAKE_FLAGS} -DGGML_CUDA=ON"
            echo "==> GPU backend: CUDA"
        elif command -v rocminfo &>/dev/null 2>&1; then
            CMAKE_FLAGS="${CMAKE_FLAGS} -DGGML_HIP=ON"
            echo "==> GPU backend: ROCm/HIP"
        else
            echo "==> GPU backend: CPU only (no CUDA or ROCm detected)"
        fi
        ;;
esac

# ── Configure and build ───────────────────────────────────────────────────────

echo "==> Configuring"
# shellcheck disable=SC2086
cmake -S "${LLAMA_DIR}" -B "${BUILD_DIR}" \
    -DCMAKE_BUILD_TYPE=Release \
    ${CMAKE_FLAGS}

echo "==> Building llama-server ($(ncpu) parallel jobs)"
cmake --build "${BUILD_DIR}" --target llama-server -j"$(ncpu)"

# ── Locate the binary (path differs across llama.cpp versions) ────────────────

BINARY=""
for candidate in \
    "${BUILD_DIR}/bin/llama-server" \
    "${BUILD_DIR}/llama-server" \
    "${BUILD_DIR}/src/llama-server"
do
    if [[ -f "$candidate" ]]; then
        BINARY="$candidate"
        break
    fi
done

[[ -n "$BINARY" ]] || die "llama-server binary not found after build — check build output above"

# ── Install ───────────────────────────────────────────────────────────────────

echo "==> Installing ${BINARY} → /usr/local/bin/llama-server"
sudo install -m 755 "${BINARY}" /usr/local/bin/llama-server

echo ""
echo "==> llama-server installed: $(llama-server --version 2>&1 | head -1)"
echo ""
echo "Usage:"
echo "  make llama-server        # embed server on port 16000 (nomic)"
echo "  make reranker-server     # reranker on port 16001 (bge-reranker-v2-m3)"
