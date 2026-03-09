#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_BASE_URL="${STUDYCLAW_SMOKE_API_BASE_URL:-http://localhost:8080}"
PING_URL="${API_BASE_URL%/}/ping"
TASKS_URL="${API_BASE_URL%/}/api/v1/tasks?family_id=306&user_id=1"

run_step() {
  local label="$1"
  shift
  printf '\n==> %s\n' "$label"
  "$@"
}

printf 'StudyClaw local smoke check\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'API base URL: %s\n' "$API_BASE_URL"

run_step "Validate secret hygiene" bash "$ROOT_DIR/scripts/check_no_tracked_runtime_env.sh"

run_step "Check backend health" bash -lc "
  response=\$(curl -fsS '$PING_URL')
  printf '%s\n' \"\$response\"
  if ! printf '%s\n' \"\$response\" | grep -q 'pong'; then
    echo 'Backend health check did not return pong.'
    exit 1
  fi
"

run_step "Check minimal taskboard API" bash -lc "
  response=\$(curl -fsS '$TASKS_URL')
  printf '%s\n' \"\$response\"
  if ! printf '%s\n' \"\$response\" | grep -q 'summary'; then
    echo 'Taskboard API response is missing summary.'
    exit 1
  fi
"

run_step "Build Parent Web" bash -lc "
  cd '$ROOT_DIR/apps/parent-web'
  npm run build
"

run_step "Build Pad Web" bash -lc "
  cd '$ROOT_DIR/apps/pad-app'
  flutter build web --dart-define=API_BASE_URL='$API_BASE_URL'
"

printf '\nSmoke checks completed successfully.\n'
