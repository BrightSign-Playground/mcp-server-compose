#!/usr/bin/env bash
# Query the MCP server directly and show retrieved chunks with scores.
#
# Usage:
#   ./scripts/query.sh "your query here" [limit]
#
# Reads Keycloak credentials from stack.toml via awk.

set -euo pipefail

QUERY="${1:?Usage: $0 \"query\" [limit]}"
LIMIT="${2:-20}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG="$REPO_ROOT/stack.toml"
MCP_PORT=$(awk '/^\[rag_mcp_server\]/{f=1} f && /^port/{print $3; exit}' "$CONFIG")
KC_PORT=$(awk '/^\[keycloak\]/{f=1} f && /^port/{print $3; exit}' "$CONFIG")
KC_REALM=$(awk '/^\[keycloak\]/{f=1} f && /^realm/{gsub(/"/, "", $3); print $3; exit}' "$CONFIG")
KC_CLIENT_ID=$(awk '/^\[keycloak\]/{f=1} f && /^m2m_client_id/{gsub(/"/, "", $3); print $3; exit}' "$CONFIG")
KC_CLIENT_SECRET=$(awk '/^\[keycloak\]/{f=1} f && /^m2m_client_secret/{gsub(/"/, "", $3); print $3; exit}' "$CONFIG")

TOKEN=$(curl -s -X POST \
    "http://localhost:${KC_PORT}/realms/${KC_REALM}/protocol/openid-connect/token" \
    -d grant_type=client_credentials \
    -d "client_id=${KC_CLIENT_ID}" \
    -d "client_secret=${KC_CLIENT_SECRET}" \
    -d audience=rag-api \
    | jq -r .access_token)

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
    echo "error: failed to obtain token from Keycloak" >&2
    exit 1
fi

MCP_URL="http://localhost:${MCP_PORT}/mcp"

# Step 1: initialize session
INIT_RESPONSE=$(curl -sD - -X POST "$MCP_URL" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"query.sh","version":"1.0"}}}')

SESSION_ID=$(echo "$INIT_RESPONSE" | grep -i '^mcp-session-id:' | tr -d '\r' | awk '{print $2}')

if [[ -z "$SESSION_ID" ]]; then
    echo "error: no session ID returned from initialize" >&2
    echo "$INIT_RESPONSE" >&2
    exit 1
fi

# Step 2: call search_documents
PAYLOAD=$(jq -n \
    --arg query "$QUERY" \
    --argjson limit "$LIMIT" \
    '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search_documents","arguments":{"query":$query,"limit":$limit}}}')

RESPONSE=$(curl -s -X POST "$MCP_URL" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -d "$PAYLOAD")

# Parse SSE: extract the data: line and parse as JSON
DATA=$(echo "$RESPONSE" | grep '^data:' | sed 's/^data: //')

echo "$DATA" | jq -r '
  .result.content[0].text | fromjson |
  .results[] |
  "\(.score | . * 1000 | round / 1000)  \(.source_path | split("/") | last):\(.chunk_index)  \(.content[:120] | gsub("\n";" "))"
'
