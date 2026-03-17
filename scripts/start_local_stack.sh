#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_PORT="${STUDYCLAW_API_PORT:-38080}"
PARENT_PORT="${STUDYCLAW_PARENT_PORT:-5173}"
PAD_PORT="${STUDYCLAW_PAD_PORT:-55771}"
API_BASE_URL="${STUDYCLAW_API_BASE_URL:-http://127.0.0.1:${API_PORT}}"

printf 'StudyClaw local stack start helper\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'API: %s\n' "$API_BASE_URL"
printf 'Parent Web: http://127.0.0.1:%s\n' "$PARENT_PORT"
printf 'Pad Web: http://127.0.0.1:%s\n\n' "$PAD_PORT"

cat <<EOF
Run the following commands in separate terminals. Important: each command must be started from its own app directory.

1) API
cd "$ROOT_DIR/apps/api-server"
API_PORT=$API_PORT go run ./cmd/studyclaw-server

2) Parent Web
cd "$ROOT_DIR/apps/parent-web"
VITE_API_BASE_URL=$API_BASE_URL npm run dev -- --host 127.0.0.1 --port $PARENT_PORT

3) Pad Web
cd "$ROOT_DIR/apps/pad-app"
flutter run -d web-server --web-hostname 127.0.0.1 --web-port $PAD_PORT --dart-define=API_BASE_URL=$API_BASE_URL

After startup, run:
bash "$ROOT_DIR/scripts/probe_local_stack.sh"
EOF
