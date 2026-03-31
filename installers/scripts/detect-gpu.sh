#!/usr/bin/env bash
# Detect GPU hardware and output optimal llama-server flags.
#
# Usage:
#   eval $(./scripts/detect-gpu.sh)
#   llama-server --model ... $LLAMA_GPU_FLAGS
#
# Outputs shell variable assignments:
#   GPU_TYPE        — "amd", "nvidia", "apple", or "cpu"
#   GPU_NAME        — human-readable GPU name
#   GPU_VRAM_MB     — total VRAM in MB (0 for CPU/Apple)
#   LLAMA_PARALLEL  — recommended --parallel value
#   LLAMA_BATCH     — recommended --batch-size value
#   LLAMA_UBATCH    — recommended --ubatch-size value
#   LLAMA_GPU_FLAGS — combined flags string for llama-server

set -euo pipefail

GPU_TYPE="cpu"
GPU_NAME="none"
GPU_VRAM_MB=0

# ── AMD GPU (ROCm) ──────────────────────────────────────────────────────────
if command -v rocm-smi &>/dev/null; then
    vram_line=$(rocm-smi --showmeminfo vram 2>/dev/null | grep "Total" | head -1 || true)
    if [[ -n "${vram_line}" ]]; then
        GPU_TYPE="amd"
        # rocm-smi reports bytes; convert to MB
        vram_bytes=$(echo "${vram_line}" | awk '{print $NF}')
        GPU_VRAM_MB=$((vram_bytes / 1024 / 1024))
        GPU_NAME=$(rocm-smi --showproductname 2>/dev/null | grep "Card Series" | head -1 | sed 's/.*: *//' || echo "AMD GPU")
    fi

# ── NVIDIA GPU (nvidia-smi) ─────────────────────────────────────────────────
elif command -v nvidia-smi &>/dev/null; then
    vram_line=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader,nounits 2>/dev/null | head -1 || true)
    if [[ -n "${vram_line}" ]]; then
        GPU_TYPE="nvidia"
        GPU_VRAM_MB=$(echo "${vram_line}" | tr -d ' ')
        GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 || echo "NVIDIA GPU")
    fi

# ── Apple Silicon (Metal) ───────────────────────────────────────────────────
elif [[ "$(uname -s)" == "Darwin" ]] && sysctl -n machdep.cpu.brand_string 2>/dev/null | grep -qi "apple"; then
    GPU_TYPE="apple"
    # Apple unified memory — report total system RAM as available
    total_bytes=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
    GPU_VRAM_MB=$((total_bytes / 1024 / 1024))
    GPU_NAME=$(sysctl -n machdep.cpu.brand_string 2>/dev/null || echo "Apple Silicon")
fi

# ── Compute optimal flags based on VRAM ──────────────────────────────────────
# These are conservative defaults that work well across model sizes.
# The key insight: more parallel slots = more concurrent requests, but each
# slot reserves ctx_size tokens of KV cache memory.

if [[ ${GPU_VRAM_MB} -ge 16000 ]]; then
    # 16GB+ VRAM (e.g. RX 7800 XT, RTX 4080, M2 Pro 16GB)
    LLAMA_PARALLEL=8
    LLAMA_BATCH=4096
    LLAMA_UBATCH=4096
elif [[ ${GPU_VRAM_MB} -ge 8000 ]]; then
    # 8-16GB VRAM (e.g. RX 7600, RTX 4060, M2 8GB)
    LLAMA_PARALLEL=4
    LLAMA_BATCH=2048
    LLAMA_UBATCH=2048
elif [[ ${GPU_VRAM_MB} -ge 4000 ]]; then
    # 4-8GB VRAM
    LLAMA_PARALLEL=2
    LLAMA_BATCH=1024
    LLAMA_UBATCH=1024
else
    # CPU or low VRAM
    LLAMA_PARALLEL=2
    LLAMA_BATCH=1024
    LLAMA_UBATCH=512
fi

LLAMA_GPU_FLAGS="--parallel ${LLAMA_PARALLEL} --batch-size ${LLAMA_BATCH} --ubatch-size ${LLAMA_UBATCH}"

# ── Output ───────────────────────────────────────────────────────────────────
cat <<EOF
GPU_TYPE="${GPU_TYPE}"
GPU_NAME="${GPU_NAME}"
GPU_VRAM_MB=${GPU_VRAM_MB}
LLAMA_PARALLEL=${LLAMA_PARALLEL}
LLAMA_BATCH=${LLAMA_BATCH}
LLAMA_UBATCH=${LLAMA_UBATCH}
LLAMA_GPU_FLAGS="${LLAMA_GPU_FLAGS}"
EOF

# Print summary to stderr so eval doesn't capture it
echo "detected: ${GPU_NAME} (${GPU_TYPE}, ${GPU_VRAM_MB}MB) → parallel=${LLAMA_PARALLEL} batch=${LLAMA_BATCH}" >&2
