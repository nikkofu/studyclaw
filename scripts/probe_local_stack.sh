#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_PORT="${STUDYCLAW_API_PORT:-38080}"
PARENT_PORT="${STUDYCLAW_PARENT_PORT:-5173}"
PAD_PORT="${STUDYCLAW_PAD_PORT:-55771}"
API_BASE_URL="${STUDYCLAW_API_BASE_URL:-http://127.0.0.1:${API_PORT}}"
PARENT_URL="${STUDYCLAW_PARENT_WEB_URL:-http://127.0.0.1:${PARENT_PORT}}"
PAD_URL="${STUDYCLAW_PAD_WEB_URL:-http://127.0.0.1:${PAD_PORT}}"

check_port_open() {
  local host="$1"
  local port="$2"
  if command -v nc >/dev/null 2>&1; then
    nc -z "$host" "$port"
  else
    bash -lc "exec 3<>/dev/tcp/$host/$port" >/dev/null 2>&1
  fi
}

printf 'StudyClaw local stack probe\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'API: %s\n' "$API_BASE_URL"
printf 'Parent Web: %s\n' "$PARENT_URL"
printf 'Pad Web: %s\n\n' "$PAD_URL"

printf '==> API ping\n'
api_response="$(curl -fsS "${API_BASE_URL%/}/ping")"
printf '%s\n' "$api_response"
if ! printf '%s\n' "$api_response" | grep -q 'pong'; then
  echo 'API probe failed: /ping did not return pong.'
  exit 1
fi

printf '\n==> Parent Web root\n'
parent_headers="$(curl -I -sS "$PARENT_URL/")"
printf '%s\n' "$parent_headers"
if ! printf '%s\n' "$parent_headers" | grep -q '200'; then
  echo 'Parent Web probe failed: root did not return HTTP 200.'
  exit 1
fi

printf '\n==> Pad Web port\n'
if check_port_open 127.0.0.1 "$PAD_PORT"; then
  echo "Pad Web port $PAD_PORT is accepting connections."
else
  echo "Pad Web probe failed: port $PAD_PORT is not accepting connections."
  exit 1
fi

printf '\n==> Pad Web root (informational)\n'
pad_status="$(curl -I -sS -o /dev/null -w '%{http_code}' "$PAD_URL/" || true)"
printf 'HTTP %s\n' "$pad_status"
if [[ "$pad_status" == "200" ]]; then
  echo 'Pad Web root returned HTML.'
elif [[ "$pad_status" == "404" ]]; then
  echo 'Pad Web root returned 404, but this is expected for flutter web-server in some debug sessions; rely on the open port plus flutter stdout as the health signal.'
else
  echo 'Pad Web root returned a non-200 status; if the port is open and flutter is waiting for a debug connection, treat this as non-blocking for local debug.'
fi

printf '\nProbe completed successfully.\n'
