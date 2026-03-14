#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

status_output="$(git status --short --untracked-files=all)"
if [[ -z "$status_output" ]]; then
  printf 'Release scope check: clean worktree\n'
  exit 0
fi

status_lines=()
while IFS= read -r line; do
  status_lines+=("$line")
done <<< "$status_output"

changed_paths=()
for entry in "${status_lines[@]}"; do
  changed_paths+=("${entry#?? }")
done

forbidden_hits=()
cleanup_hits=()
allowed_hits=()

for entry in "${status_lines[@]}"; do
  status_code="${entry:0:2}"
  path="${entry#?? }"
  is_forbidden=0
  is_cleanup=0

  case "$path" in
    apps/api-server/.gopath/*)
      cleanup_hits+=("$path")
      is_cleanup=1
      ;;
  esac

  case "$path" in
    .claude/* | */build/* | build/* | */dist/* | dist/* | */.dart_tool/* | .dart_tool/*)
      forbidden_hits+=("$path")
      is_forbidden=1
      ;;
    .env | .env.* | */.env | */.env.* | runtime.env | runtime.env.* | */runtime.env | */runtime.env.*)
      if [[ "$path" != ".env.example" && "$path" != */".env.example" ]]; then
        forbidden_hits+=("$path")
        is_forbidden=1
      fi
      ;;
  esac

  if (( is_forbidden == 0 && is_cleanup == 0 )); then
    allowed_hits+=("$path")
  fi
done

printf 'Release scope check\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'Changed paths: %d\n' "${#changed_paths[@]}"
printf 'Allowed candidate paths: %d\n' "${#allowed_hits[@]}"
printf 'Tracked cache cleanup paths: %d\n' "${#cleanup_hits[@]}"
printf 'Forbidden noise paths: %d\n' "${#forbidden_hits[@]}"

if (( ${#allowed_hits[@]} > 0 )); then
  printf '\nAllowed candidate paths:\n'
  printf '%s\n' "${allowed_hits[@]}"
fi

if (( ${#cleanup_hits[@]} > 0 )); then
  printf '\nTracked cache cleanup paths:\n'
  printf '%s\n' "${cleanup_hits[@]}"
fi

if (( ${#forbidden_hits[@]} > 0 )); then
  printf '\nForbidden noise paths:\n'
  printf '%s\n' "${forbidden_hits[@]}"
  exit 1
fi

printf '\nRelease scope check passed.\n'
