#!/usr/bin/env bash
set -euo pipefail

show_help() {
  cat <<'EOF'
Usage: scripts/install.sh [options]

Builds the video-agent-skills CLI, copies it to ~/bin, and copies .env to a
config directory in your home.

Options:
  -b, --bindir DIR    Destination bin directory (default: ~/bin)
  -c, --config DIR    Config directory (default: ~/.config/video-agent-skills)
  -e, --env FILE      .env file to copy (default: ./.env)
  -h, --help          Show this help
EOF
}

BINDIR="${HOME}/bin"
CONFIG_DIR="${HOME}/.config/video-agent-skills"
ENV_FILE="./.env"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -b|--bindir)
      BINDIR="$2"
      shift 2
      ;;
    -c|--config)
      CONFIG_DIR="$2"
      shift 2
      ;;
    -e|--env)
      ENV_FILE="$2"
      shift 2
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      show_help >&2
      exit 1
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  if [[ -x /usr/local/go/bin/go ]]; then
    export PATH="/usr/local/go/bin:${PATH}"
  else
    echo "go not found in PATH, and /usr/local/go/bin/go not available." >&2
    exit 1
  fi
fi

echo "Building video-agent-skills..."
go build -o ./video-agent-skills

echo "Copying binary to ${BINDIR}..."
mkdir -p "${BINDIR}"
install -m 0755 ./video-agent-skills "${BINDIR}/video-agent-skills"

echo "Copying env file to ${CONFIG_DIR}..."
mkdir -p "${CONFIG_DIR}"
if [[ -f "${ENV_FILE}" ]]; then
  install -m 0600 "${ENV_FILE}" "${CONFIG_DIR}/.env"
else
  echo "Warning: ${ENV_FILE} not found; skipping env copy." >&2
fi

echo "Done."
