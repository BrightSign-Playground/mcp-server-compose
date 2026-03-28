#!/usr/bin/env bash
# eval-stability.sh — Run the eval suite N times and report pass-rate statistics.
#
# Usage:
#   ./scripts/eval-stability.sh [RUNS] [EVAL_FILE]
#
# Defaults:
#   RUNS      = 25
#   EVAL_FILE = docs2vector/data/techdocs/evals/evals.json
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

RUNS="${1:-25}"
EVAL_FILE="${2:-docs2vector/data/techdocs/evals/evals.json}"

[[ -f "$EVAL_FILE" ]] || { echo "Error: eval file not found: $EVAL_FILE" >&2; exit 1; }

TOTAL_EVALS=$(python3 -c "import json,sys; print(len(json.load(open('$EVAL_FILE'))))" 2>/dev/null \
    || jq 'length' "$EVAL_FILE")

echo "Eval file : $EVAL_FILE ($TOTAL_EVALS evals)"
echo "Runs      : $RUNS"
echo "Started   : $(date)"
echo "────────────────────────────────────────────"

declare -a PASS_COUNTS=()
declare -a FAIL_INDICES=()
declare -A FAIL_FREQ=()
total_passed=0

for run in $(seq 1 "$RUNS"); do
    printf "Run %2d/%d ... " "$run" "$RUNS"

    output=$(make eval EVAL_FILE="$EVAL_FILE" 2>&1) || true

    # Extract "N/M passed" from the Results line (strip ANSI color codes first).
    result_line=$(echo "$output" | sed 's/\x1b\[[0-9;]*m//g' | grep -oE 'Results: [0-9]+/[0-9]+ passed' | tail -1)

    if [[ -z "$result_line" ]]; then
        echo "FAILED (no results line found — eval command error?)"
        echo "--- output ---"
        echo "$output" | tail -20
        echo "--------------"
        continue
    fi

    passed=$(echo "$result_line" | grep -oE '^Results: [0-9]+' | grep -oE '[0-9]+$')
    total=$(echo "$result_line" | grep -oE '[0-9]+/[0-9]+' | cut -d/ -f2)

    PASS_COUNTS+=("$passed")
    total_passed=$((total_passed + passed))

    pct=$(awk "BEGIN { printf \"%.1f\", 100 * $passed / $total }")
    printf "%d/%d (%.1f%%)\n" "$passed" "$total" "$pct"

    # Collect failed eval indices for frequency analysis.
    failed_line=$(echo "$output" | sed 's/\x1b\[[0-9;]*m//g' | grep 'Failed eval indices:' | tail -1)
    if [[ -n "$failed_line" ]]; then
        indices="${failed_line#*: }"
        IFS=',' read -ra idx_arr <<< "$indices"
        for idx in "${idx_arr[@]}"; do
            idx="${idx// /}"
            [[ -z "$idx" ]] && continue
            FAIL_FREQ["$idx"]=$(( ${FAIL_FREQ["$idx"]:-0} + 1 ))
        done
    fi
done

actual_runs="${#PASS_COUNTS[@]}"
[[ "$actual_runs" -eq 0 ]] && { echo "No successful runs." >&2; exit 1; }

# ── Summary statistics ────────────────────────────────────────────────────────

min_pass="${PASS_COUNTS[0]}"
max_pass="${PASS_COUNTS[0]}"
for v in "${PASS_COUNTS[@]}"; do
    (( v < min_pass )) && min_pass=$v
    (( v > max_pass )) && max_pass=$v
done

avg_pass=$(awk "BEGIN { printf \"%.2f\", $total_passed / $actual_runs }")
avg_pct=$(awk "BEGIN { printf \"%.1f\", 100 * $total_passed / ($actual_runs * $TOTAL_EVALS) }")
min_pct=$(awk "BEGIN { printf \"%.1f\", 100 * $min_pass / $TOTAL_EVALS }")
max_pct=$(awk "BEGIN { printf \"%.1f\", 100 * $max_pass / $TOTAL_EVALS }")

echo ""
echo "════════════════════════════════════════════"
echo "Results across $actual_runs runs (out of $RUNS attempted)"
echo "────────────────────────────────────────────"
printf "  Average : %.2f / %d  (%s%%)\n" "$avg_pass" "$TOTAL_EVALS" "$avg_pct"
printf "  Min     : %d / %d  (%s%%)\n"   "$min_pass"  "$TOTAL_EVALS" "$min_pct"
printf "  Max     : %d / %d  (%s%%)\n"   "$max_pass"  "$TOTAL_EVALS" "$max_pct"

# ── Most frequently failing evals ────────────────────────────────────────────

if [[ "${#FAIL_FREQ[@]}" -gt 0 ]]; then
    echo ""
    echo "Most frequently failing evals (sorted by failure count):"
    for idx in $(for k in "${!FAIL_FREQ[@]}"; do echo "${FAIL_FREQ[$k]} $k"; done \
                    | sort -rn | awk '{print $2}'); do
        count="${FAIL_FREQ[$idx]}"
        pct=$(awk "BEGIN { printf \"%.0f\", 100 * $count / $actual_runs }")
        printf "  Eval %3s : failed %d/%d runs (%s%%)\n" "$idx" "$count" "$actual_runs" "$pct"
    done
fi

echo "════════════════════════════════════════════"
echo "Finished  : $(date)"
