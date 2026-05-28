#!/usr/bin/env bash
# ============================================================
# Xemarify Agent - One-liner Installer
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/agent/install.sh | \
#     sudo MANAGER_ENDPOINT=http://<ip>:8089 ENROLLMENT_TOKEN=<token> bash
#
#   # With specific version:
#   curl -fsSL https://raw.githubusercontent.com/hildanku/xemarify/main/deploy/agent/install.sh | \
#     sudo MANAGER_ENDPOINT=http://<ip>:8089 ENROLLMENT_TOKEN=<token> VERSION=v1.1.0-beta bash
#
# Tested on: Ubuntu 22.04 LTS
# ============================================================
set -euo pipefail

VERSION="${VERSION:-v1.1.0-beta}"
ARCH="${ARCH:-$(dpkg --print-architecture 2>/dev/null || echo amd64)}"
MANAGER_ENDPOINT="${MANAGER_ENDPOINT:-}"
ENROLLMENT_TOKEN="${ENROLLMENT_TOKEN:-}"
INSECURE_TLS="${INSECURE_TLS:-false}"
DOWNLOAD_URL="${DOWNLOAD_URL:-}"
FORCE_CONFIG="${FORCE_CONFIG:-false}"

BIN_PATH="/usr/local/bin/xemarify-agent"
SERVICE_PATH="/etc/systemd/system/xemarify-agent.service"
CONFIG_DIR="/etc/xemarify-agent"
CONFIG_PATH="${CONFIG_DIR}/agent.yaml"
STATE_DIR="/var/lib/xemarify-agent/spool"

GITHUB_REPO="hildanku/xemarify"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[xemarify-agent]${NC} $*"; }
warn() { echo -e "${YELLOW}[warning]${NC} $*"; }
err()  { echo -e "${RED}[error]${NC} $*" >&2; }

# Preflight

if [[ "${EUID}" -ne 0 ]]; then
  err "Please run as root (use sudo)."
  exit 1
fi

if [[ -z "${MANAGER_ENDPOINT}" ]]; then
  err "MANAGER_ENDPOINT is required."
  err ""
  err "Usage:"
  err "  curl -fsSL https://raw.githubusercontent.com/${GITHUB_REPO}/main/deploy/agent/install.sh | \\"
  err "    sudo MANAGER_ENDPOINT=http://<manager-ip>:8089 ENROLLMENT_TOKEN=<token> bash"
  exit 1
fi

if [[ "${FORCE_CONFIG}" == "true" || ! -f "${CONFIG_PATH}" ]]; then
  if [[ -z "${ENROLLMENT_TOKEN}" ]]; then
    err "ENROLLMENT_TOKEN is required for first install."
    exit 1
  fi
fi

# Download binary

if [[ -z "${DOWNLOAD_URL}" ]]; then
  DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/xemarify-agent_${VERSION#v}_linux_${ARCH}.tar.gz"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

archive="${tmpdir}/xemarify-agent.tar.gz"

log "Downloading agent ${VERSION} (${ARCH})..."
log "URL: ${DOWNLOAD_URL}"

if command -v curl >/dev/null 2>&1; then
  if ! curl -fSL "${DOWNLOAD_URL}" -o "${archive}"; then
    err "Download failed. Check that version ${VERSION} exists at:"
    err "  https://github.com/${GITHUB_REPO}/releases"
    exit 1
  fi
elif command -v wget >/dev/null 2>&1; then
  if ! wget -q "${DOWNLOAD_URL}" -O "${archive}"; then
    err "Download failed."
    exit 1
  fi
else
  err "curl or wget is required."
  exit 1
fi

# Install binary

log "Installing binary..."
mkdir -p "${tmpdir}/extract"
tar -xzf "${archive}" -C "${tmpdir}/extract"

binary_source="$(find "${tmpdir}/extract" -type f -name 'xemarify-agent' | head -n 1)"
if [[ -z "${binary_source}" ]]; then
  err "xemarify-agent binary not found in archive."
  exit 1
fi

install -m 0755 "${binary_source}" "${BIN_PATH}"
mkdir -p "${CONFIG_DIR}" "${STATE_DIR}"

# Write config

if [[ "${FORCE_CONFIG}" == "true" || ! -f "${CONFIG_PATH}" ]]; then
  log "Writing config to ${CONFIG_PATH}..."

  cat > "${CONFIG_PATH}" <<EOF
server:
  endpoint: "${MANAGER_ENDPOINT}"
  insecure: ${INSECURE_TLS}

enrollment_token: "${ENROLLMENT_TOKEN}"

disk_buffer:
  path: "${STATE_DIR}/events.log"
  max_bytes: 524288000

agent:
  id: ""
  agent_secret: ""
  name: ""
  hostname: ""
  ip_address: ""

syslog:
  listen: ":5514"

filelog:
  enabled: true
  poll_interval: 5s
  paths:
    - /var/log/syslog
    - /var/log/auth.log

inventory:
  enabled: true
  interval: 5m
EOF

  chmod 600 "${CONFIG_PATH}"
fi

# Install systemd service

log "Installing systemd service..."

cat > "${SERVICE_PATH}" <<'EOF'
[Unit]
Description=Xemarify Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/xemarify-agent
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/etc/xemarify-agent /var/lib/xemarify-agent
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now xemarify-agent

# Done

echo ""
log "Xemarify Agent installed successfully!"
log ""
log "  Binary:   ${BIN_PATH}"
log "  Config:   ${CONFIG_PATH}"
log "  Service:  xemarify-agent.service"
log ""
log "  Status:   systemctl status xemarify-agent"
log "  Logs:     journalctl -u xemarify-agent -f"
log "  Restart:  systemctl restart xemarify-agent"
echo ""
