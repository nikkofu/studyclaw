#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_BASE_URL="${STUDYCLAW_SMOKE_API_BASE_URL:-http://localhost:8080}"
PARENT_WEB_URL="${STUDYCLAW_PARENT_WEB_URL:-http://localhost:5173}"
PAD_WEB_HINT="${STUDYCLAW_PAD_WEB_HINT:-flutter run --dart-define=API_BASE_URL=${API_BASE_URL} -d chrome}"

printf 'StudyClaw local demo entry\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'API base URL: %s\n' "$API_BASE_URL"
printf 'Parent Web URL: %s\n\n' "$PARENT_WEB_URL"

printf 'Step 1/3: local preflight\n'
bash "$ROOT_DIR/scripts/preflight_local_env.sh"

printf '\nStep 2/3: smoke validation\n'
bash "$ROOT_DIR/scripts/smoke_local_stack.sh"

printf '\nStep 3/3: demo walkthrough\n'
cat <<EOF
The local environment is ready for a live demo.

Recommended demo flow:
1. Keep the Go backend running at ${API_BASE_URL}
2. Start Parent Web:
   cd ${ROOT_DIR}/apps/parent-web
   npm run dev -- --host 0.0.0.0
3. Start Pad App in a separate terminal:
   cd ${ROOT_DIR}/apps/pad-app
   ${PAD_WEB_HINT}
4. Open Parent Web:
   ${PARENT_WEB_URL}
5. Demo story:
   - Publish homework for a fixed assigned date
   - Review parsed tasks and confirm creation
   - Switch to Pad and load the same date
   - Mark one task, one group, and all tasks complete
   - Refresh Parent Web and review same-day stats
   - If available in the current build, demo word playback and points changes

Suggested reference docs:
- ${ROOT_DIR}/docs/06_RUNBOOK.md
- ${ROOT_DIR}/docs/13_RELEASE_CHECKLIST.md
- ${ROOT_DIR}/docs/16_FIRST_PHASE_DEMO_CHECKLIST.md
- ${ROOT_DIR}/docs/14_NEXT_PHASE_DISPATCH.md
EOF
