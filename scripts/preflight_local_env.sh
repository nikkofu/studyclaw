#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -n "${STUDYCLAW_ENV_FILE:-}" ]]; then
  RUNTIME_ENV_FILE="${STUDYCLAW_ENV_FILE}"
elif [[ -n "${STUDYCLAW_CONFIG_DIR:-}" ]]; then
  RUNTIME_ENV_FILE="${STUDYCLAW_CONFIG_DIR%/}/runtime.env"
elif [[ -n "${XDG_CONFIG_HOME:-}" ]]; then
  RUNTIME_ENV_FILE="${XDG_CONFIG_HOME%/}/studyclaw/runtime.env"
else
  RUNTIME_ENV_FILE="${HOME}/.config/studyclaw/runtime.env"
fi

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

print_ok() {
  printf '[OK] %s\n' "$1"
  PASS_COUNT=$((PASS_COUNT + 1))
}

print_warn() {
  printf '[WARN] %s\n' "$1"
  WARN_COUNT=$((WARN_COUNT + 1))
}

print_fail() {
  printf '[FAIL] %s\n' "$1"
  FAIL_COUNT=$((FAIL_COUNT + 1))
}

extract_version() {
  local raw_text="$1"
  local version
  version="$(printf '%s\n' "$raw_text" | grep -Eo '[0-9]+([.][0-9]+){1,2}' | head -n 1 || true)"
  printf '%s' "$version"
}

version_ge() {
  local current="$1"
  local required="$2"
  local IFS=.
  local current_parts required_parts
  local index current_value required_value max_parts

  read -r -a current_parts <<< "$current"
  read -r -a required_parts <<< "$required"

  max_parts="${#current_parts[@]}"
  if (( ${#required_parts[@]} > max_parts )); then
    max_parts="${#required_parts[@]}"
  fi

  for ((index = 0; index < max_parts; index++)); do
    current_value="${current_parts[index]:-0}"
    required_value="${required_parts[index]:-0}"
    if (( current_value > required_value )); then
      return 0
    fi
    if (( current_value < required_value )); then
      return 1
    fi
  done

  return 0
}

check_tool_version() {
  local label="$1"
  local command_name="$2"
  local version_command="$3"
  local minimum_version="$4"
  local raw_output version

  if ! command -v "$command_name" >/dev/null 2>&1; then
    print_fail "$label 未安装。"
    return
  fi

  raw_output="$(bash -lc "$version_command" 2>/dev/null || true)"
  version="$(extract_version "$raw_output")"

  if [[ -z "$version" ]]; then
    print_warn "$label 已安装，但无法解析版本号。原始输出: ${raw_output:-<empty>}"
    return
  fi

  if version_ge "$version" "$minimum_version"; then
    print_ok "${label} 已安装，版本 ${version}，满足 >= ${minimum_version}。"
  else
    print_fail "${label} 版本过低，当前 ${version}，要求 >= ${minimum_version}。"
  fi
}

check_directory() {
  local label="$1"
  local target_path="$2"
  if [[ -d "$target_path" ]]; then
    print_ok "$label 目录存在: $target_path"
  else
    print_fail "$label 目录不存在: $target_path"
  fi
}

resolve_docker_bin() {
  local candidate

  if candidate="$(command -v docker 2>/dev/null || true)" && [[ -n "$candidate" ]]; then
    printf '%s' "$candidate"
    return 0
  fi

  for candidate in \
    "/Applications/Docker.app/Contents/Resources/bin/docker" \
    "/usr/local/bin/docker" \
    "/opt/homebrew/bin/docker"; do
    if [[ -x "$candidate" ]]; then
      printf '%s' "$candidate"
      return 0
    fi
  done

  return 1
}

printf 'StudyClaw local preflight\n'
printf 'Repository: %s\n' "$ROOT_DIR"
printf 'Runtime env: %s\n\n' "$RUNTIME_ENV_FILE"

check_tool_version "Go" "go" "go version" "1.25.0"
check_tool_version "Node.js" "node" "node --version" "20.0.0"
check_tool_version "npm" "npm" "npm --version" "10.0.0"
check_tool_version "Flutter" "flutter" "flutter --version" "3.24.0"

DOCKER_BIN="$(resolve_docker_bin || true)"
if [[ -n "$DOCKER_BIN" ]]; then
  check_tool_version "Docker" "$DOCKER_BIN" "'$DOCKER_BIN' --version" "20.10.0"
  if "$DOCKER_BIN" compose version >/dev/null 2>&1; then
    print_ok "Docker Compose 可用。"
  else
    print_warn "Docker Compose 不可用。当前演示链路不依赖 Redis，但如果后续启用 Redis，需要补齐 compose。"
  fi

  if "$DOCKER_BIN" info >/dev/null 2>&1; then
    print_ok "Docker daemon 正常运行。"
  else
    print_warn "Docker 已安装，但 daemon 当前不可用。当前演示链路可继续；如需 Redis，请先启动 Docker。"
  fi
else
  print_warn "Docker 未安装。当前演示链路可继续，但若后续启用 Redis，需要安装 Docker。"
fi

check_directory "Go backend" "$ROOT_DIR/apps/api-server"
check_directory "Parent Web" "$ROOT_DIR/apps/parent-web"
check_directory "Pad app" "$ROOT_DIR/apps/pad-app"
check_directory "Docs" "$ROOT_DIR/docs"
check_directory "Scripts" "$ROOT_DIR/scripts"

if [[ -f "$RUNTIME_ENV_FILE" ]]; then
  print_ok "私有 runtime.env 存在。"
else
  print_fail "私有 runtime.env 不存在。先执行: bash scripts/init_private_runtime_env.sh"
fi

if bash "$ROOT_DIR/scripts/check_no_tracked_runtime_env.sh" >/dev/null 2>&1; then
  print_ok "仓库中没有被跟踪的运行时密钥文件。"
else
  print_fail "检测到被跟踪的运行时密钥文件，请先处理。"
fi

printf '\nSummary: %d passed, %d warnings, %d failed\n' "$PASS_COUNT" "$WARN_COUNT" "$FAIL_COUNT"

if (( FAIL_COUNT > 0 )); then
  exit 1
fi
