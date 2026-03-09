#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${STUDYCLAW_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/studyclaw}"
ENV_FILE="${STUDYCLAW_ENV_FILE:-$CONFIG_DIR/runtime.env}"
ENV_DIR="$(dirname "$ENV_FILE")"

mkdir -p "$ENV_DIR"
chmod 700 "$ENV_DIR"

if [[ ! -f "$ENV_FILE" ]]; then
  cp "$ROOT_DIR/.env.example" "$ENV_FILE"
  echo "Created private runtime config at $ENV_FILE"
else
  echo "Private runtime config already exists at $ENV_FILE"
fi

chmod 600 "$ENV_FILE"

cat <<EOF

Next steps:
1. Edit $ENV_FILE and fill in the real secrets there.
2. Keep repo-root .env only for non-sensitive local defaults if you still need it.
3. Start services normally. Load order is:
   process environment -> private runtime env -> repo .env fallback

Do not place real API keys in parent-web, pad-app, README, docs, or tracked env files.
EOF
