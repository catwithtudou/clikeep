#!/usr/bin/env bash
set -u

MODULE="github.com/catwithtudou/clikeep/cmd/clikeep"
VERSION="latest"

ACTION="check"
INIT_CONFIG="false"
JSON="false"

usage() {
  cat <<'USAGE'
usage: bootstrap_clikeep.sh [--check | --install] [--init] [--json]

Options:
  --check    Inspect clikeep availability without changing the machine.
  --install  Install clikeep with `go install github.com/catwithtudou/clikeep/cmd/clikeep@latest`.
  --init     Run `clikeep init` after clikeep is available.
  --json     Emit machine-readable JSON.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --check)
      ACTION="check"
      ;;
    --install)
      ACTION="install"
      ;;
    --init)
      INIT_CONFIG="true"
      ;;
    --json)
      JSON="true"
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
  shift
done

if [ "$ACTION" = "check" ] && [ "$INIT_CONFIG" = "true" ]; then
  ACTION="init"
fi

json_string() {
  local value="${1:-}"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  value="${value//$'\r'/\\r}"
  printf '"%s"' "$value"
}

bool_value() {
  if [ "${1:-false}" = "true" ]; then
    printf "true"
  else
    printf "false"
  fi
}

first_gopath() {
  local gopath="${1:-}"
  if [ -z "$gopath" ]; then
    return 1
  fi
  printf "%s" "${gopath%%:*}"
}

config_path() {
  if [ -n "${XDG_CONFIG_HOME:-}" ]; then
    printf "%s/clikeep/config.toml" "$XDG_CONFIG_HOME"
  elif [ -n "${HOME:-}" ]; then
    printf "%s/.config/clikeep/config.toml" "$HOME"
  fi
}

state_path() {
  if [ -n "${XDG_STATE_HOME:-}" ]; then
    printf "%s/clikeep" "$XDG_STATE_HOME"
  elif [ -n "${HOME:-}" ]; then
    printf "%s/.local/state/clikeep" "$HOME"
  fi
}

detect() {
  CLI_PATH="$(command -v clikeep 2>/dev/null || true)"
  GO_PATH="$(command -v go 2>/dev/null || true)"
  CLI_PRESENT="false"
  GO_PRESENT="false"
  VERSION_OUTPUT=""
  CONFIG_PATH="$(config_path)"
  STATE_PATH="$(state_path)"
  CONFIG_INITIALIZED="false"
  STATE_INITIALIZED="false"

  if [ -n "$CLI_PATH" ]; then
    CLI_PRESENT="true"
    VERSION_OUTPUT="$("$CLI_PATH" version 2>/dev/null || true)"
  fi
  if [ -n "$GO_PATH" ]; then
    GO_PRESENT="true"
  fi
  if [ -n "$CONFIG_PATH" ] && [ -f "$CONFIG_PATH" ]; then
    CONFIG_INITIALIZED="true"
  fi
  if [ -n "$STATE_PATH" ] && [ -d "$STATE_PATH/runs" ]; then
    STATE_INITIALIZED="true"
  fi
}

expected_go_bin() {
  if [ -z "${GO_PATH:-}" ]; then
    return 1
  fi
  local gobin
  local gopath
  gobin="$(go env GOBIN 2>/dev/null || true)"
  if [ -n "$gobin" ]; then
    printf "%s/clikeep" "$gobin"
    return 0
  fi
  gopath="$(go env GOPATH 2>/dev/null || true)"
  gopath="$(first_gopath "$gopath")"
  if [ -n "$gopath" ]; then
    printf "%s/bin/clikeep" "$gopath"
    return 0
  fi
  return 1
}

emit_json() {
  local ok="$1"
  local status="$2"
  local message="$3"
  local exit_code="$4"
  printf '{'
  printf '"ok":%s,' "$(bool_value "$ok")"
  printf '"status":%s,' "$(json_string "$status")"
  printf '"message":%s,' "$(json_string "$message")"
  printf '"cli_present":%s,' "$(bool_value "$CLI_PRESENT")"
  printf '"cli_path":%s,' "$(json_string "$CLI_PATH")"
  printf '"go_present":%s,' "$(bool_value "$GO_PRESENT")"
  printf '"go_path":%s,' "$(json_string "$GO_PATH")"
  printf '"version":%s,' "$(json_string "$VERSION_OUTPUT")"
  printf '"config_initialized":%s,' "$(bool_value "$CONFIG_INITIALIZED")"
  printf '"config_path":%s,' "$(json_string "$CONFIG_PATH")"
  printf '"state_initialized":%s,' "$(bool_value "$STATE_INITIALIZED")"
  printf '"state_path":%s,' "$(json_string "$STATE_PATH")"
  printf '"installed_path":%s,' "$(json_string "${INSTALLED_PATH:-}")"
  printf '"path_visible":%s,' "$(bool_value "${PATH_VISIBLE:-false}")"
  printf '"path_hint":%s,' "$(json_string "${PATH_HINT:-}")"
  printf '"exit_code":%s' "$exit_code"
  printf '}\n'
}

emit_text() {
  local ok="$1"
  local status="$2"
  local message="$3"
  echo "status: $status"
  echo "ok: $ok"
  echo "message: $message"
  [ -n "$CLI_PATH" ] && echo "clikeep: $CLI_PATH"
  [ -n "$VERSION_OUTPUT" ] && echo "version: $VERSION_OUTPUT"
  [ -n "$GO_PATH" ] && echo "go: $GO_PATH"
  [ -n "${CONFIG_PATH:-}" ] && echo "config: $CONFIG_PATH"
  [ -n "${STATE_PATH:-}" ] && echo "state: $STATE_PATH"
  [ -n "${INSTALLED_PATH:-}" ] && echo "installed_path: $INSTALLED_PATH"
  [ -n "${PATH_HINT:-}" ] && echo "path_hint: $PATH_HINT"
}

finish() {
  local ok="$1"
  local status="$2"
  local message="$3"
  local exit_code="$4"
  if [ "$JSON" = "true" ]; then
    emit_json "$ok" "$status" "$message" "$exit_code"
  else
    emit_text "$ok" "$status" "$message"
  fi
  exit "$exit_code"
}

detect
INSTALLED_PATH=""
PATH_VISIBLE="$CLI_PRESENT"
PATH_HINT=""

if [ "$ACTION" = "check" ]; then
  if [ "$CLI_PRESENT" != "true" ]; then
    if [ "$GO_PRESENT" != "true" ]; then
      finish "false" "go_missing" "clikeep is not installed and go is required to install it" 0
    fi
    finish "false" "cli_missing" "clikeep is not installed or is not on PATH" 0
  fi
  if [ "$CONFIG_INITIALIZED" != "true" ] || [ "$STATE_INITIALIZED" != "true" ]; then
    finish "false" "config_missing" "clikeep is installed but config/state are not initialized" 0
  fi
  finish "true" "ready" "clikeep is installed and initialized" 0
fi

if [ "$ACTION" = "install" ]; then
  if [ "$GO_PRESENT" != "true" ]; then
    finish "false" "go_missing" "go is required to install clikeep" 1
  fi

  if [ "$CLI_PRESENT" != "true" ]; then
    if [ "$JSON" = "true" ]; then
      INSTALL_OUTPUT="$(go install "${MODULE}@${VERSION}" 2>&1)"
      INSTALL_CODE="$?"
      if [ "$INSTALL_CODE" -ne 0 ]; then
        finish "false" "install_failed" "go install failed: $INSTALL_OUTPUT" 1
      fi
    else
      if ! go install "${MODULE}@${VERSION}"; then
        finish "false" "install_failed" "go install failed" 1
      fi
    fi
  fi

  INSTALLED_PATH="$(expected_go_bin || true)"
  detect
  PATH_VISIBLE="$CLI_PRESENT"
  if [ "$CLI_PRESENT" != "true" ]; then
    if [ -n "$INSTALLED_PATH" ] && [ -x "$INSTALLED_PATH" ]; then
      PATH_HINT="Add $(dirname "$INSTALLED_PATH") to PATH so agents can run clikeep."
      VERSION_OUTPUT="$("$INSTALLED_PATH" version 2>/dev/null || true)"
      finish "false" "path_missing" "clikeep was installed but is not visible on PATH" 1
    fi
    finish "false" "path_missing" "clikeep was installed but the binary location could not be found on PATH" 1
  fi

  if [ -z "$VERSION_OUTPUT" ]; then
    finish "false" "version_failed" "clikeep is on PATH but version verification failed" 1
  fi

  if [ "$INIT_CONFIG" = "true" ]; then
    if ! "$CLI_PATH" init >/dev/null; then
      finish "false" "init_failed" "clikeep init failed" 1
    fi
    detect
  fi
  finish "true" "ready" "clikeep is installed and ready" 0
fi

if [ "$ACTION" = "init" ]; then
  if [ "$CLI_PRESENT" != "true" ]; then
    finish "false" "cli_missing" "clikeep is not installed or is not on PATH" 1
  fi
  if ! "$CLI_PATH" init >/dev/null; then
    finish "false" "init_failed" "clikeep init failed" 1
  fi
  detect
  finish "true" "ready" "clikeep is initialized" 0
fi

finish "false" "invalid_action" "unknown action" 2
