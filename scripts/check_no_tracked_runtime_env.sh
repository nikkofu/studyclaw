#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

tracked_runtime_env_files="$(git ls-files -- '*.env' '*.env.*' 'runtime.env' 'runtime.env.*' 'secrets.env' 'secrets.env.*' ':!*.example' ':!.env.example')"

if [[ -n "$tracked_runtime_env_files" ]]; then
  echo "Tracked runtime env files detected. Move secrets outside the repo and untrack these files:"
  echo "$tracked_runtime_env_files"
  exit 1
fi

echo "No tracked runtime env files detected."
