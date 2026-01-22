#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILL_NAME="video-agent-skills"
SKILL_SRC="${ROOT_DIR}/claude-skill/${SKILL_NAME}"
TARGET_DIR="${HOME}/.claude/skills/${SKILL_NAME}"

if [[ ! -d "${SKILL_SRC}" ]]; then
  echo "Skill source not found at ${SKILL_SRC}" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  if [[ -x /usr/local/go/bin/go ]]; then
    export PATH="/usr/local/go/bin:${PATH}"
  else
    echo "go not found in PATH, and /usr/local/go/bin/go not available." >&2
    exit 1
  fi
fi

BIN_TMP="$(mktemp -d)"
trap 'rm -rf "${BIN_TMP}"' EXIT

pushd "${ROOT_DIR}" >/dev/null
  echo "Building ${SKILL_NAME} binary..."
  go build -o "${BIN_TMP}/${SKILL_NAME}" ./
popd >/dev/null

mkdir -p "${TARGET_DIR}/bin"

cp "${SKILL_SRC}/SKILL.md" "${TARGET_DIR}/SKILL.md"
install -m 0755 "${BIN_TMP}/${SKILL_NAME}" "${TARGET_DIR}/bin/${SKILL_NAME}"

echo "Installed Claude Code skill to ${TARGET_DIR}"
