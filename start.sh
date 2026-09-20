#!/usr/bin/env bash
set -euo pipefail

: "${OWNER_PASSWORD:?OWNER_PASSWORD is required}"
: "${OAUTH_SIGNING_SECRET:?OAUTH_SIGNING_SECRET is required}"

export DATA_DIR="${DATA_DIR:-/data}"
mkdir -p "${DATA_DIR}/home" "${DATA_DIR}/config"

export COOKIES_PATH="${COOKIES_PATH:-${DATA_DIR}/cookies.json}"
export HOME="${HOME:-${DATA_DIR}/home}"
export XDG_CONFIG_HOME="${XDG_CONFIG_HOME:-${DATA_DIR}/config}"

if [[ -z "${UPSTREAM_AUTH_TOKEN:-}" ]]; then
  UPSTREAM_AUTH_TOKEN="$(head -c 64 /dev/urandom | base64 | tr -dc 'A-Za-z0-9' | head -c 48)"
  export UPSTREAM_AUTH_TOKEN
fi

export AUTH_TOKEN="${UPSTREAM_AUTH_TOKEN}"
export UPSTREAM_URL="${UPSTREAM_URL:-http://127.0.0.1:18061}"

echo "[sol-xiaohongshu] starting upstream MCP on :18061"
/app/app -port ":18061" &
UPSTREAM_PID=$!

cleanup() {
  kill "${UPSTREAM_PID}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Give the embedded browser/MCP process a moment to initialize.
sleep 2

echo "[sol-xiaohongshu] starting OAuth gateway on :${PORT:-8080}"
exec /app/sol-gateway
