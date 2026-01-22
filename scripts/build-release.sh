#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILL_NAME="video-agent-skills"
VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  if command -v node >/dev/null 2>&1; then
    VERSION="$(node -p "require('./package.json').version" --prefix "${ROOT_DIR}")"
  else
    echo "Version not provided and node not available." >&2
    exit 1
  fi
fi

DIST_DIR="${ROOT_DIR}/dist"
mkdir -p "${DIST_DIR}"

build_target() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  local archive_ext="tar.gz"

  if [[ "${goos}" == "windows" ]]; then
    ext=".exe"
    archive_ext="zip"
  fi

  local out_dir
  out_dir="$(mktemp -d)"
  trap 'rm -rf "${out_dir}"' RETURN

  echo "Building ${SKILL_NAME} for ${goos}/${goarch}..."
  (cd "${ROOT_DIR}" && CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build -trimpath -ldflags "-s -w" -o "${out_dir}/${SKILL_NAME}${ext}" ./)

  local suffix="${goos}_${goarch}"
  local archive="${SKILL_NAME}_${VERSION}_${suffix}.${archive_ext}"

  if [[ "${archive_ext}" == "zip" ]]; then
    (cd "${out_dir}" && zip -q "${DIST_DIR}/${archive}" "${SKILL_NAME}${ext}")
  else
    tar -czf "${DIST_DIR}/${archive}" -C "${out_dir}" "${SKILL_NAME}${ext}"
  fi

  echo "Wrote ${DIST_DIR}/${archive}"
}

build_target darwin amd64
build_target darwin arm64
build_target linux amd64
build_target linux arm64
build_target windows amd64

