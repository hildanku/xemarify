#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-v0.1.0}"
ARCH="${ARCH:-amd64}"
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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEMPLATE_CONFIG="${SCRIPT_DIR}/agent.production.yaml"
TEMPLATE_SERVICE="${SCRIPT_DIR}/xemarify-agent.service"

usage() {
  cat <<EOF
Usage:
  sudo MANAGER_ENDPOINT=http://<manager-host>:8089 ENROLLMENT_TOKEN=<token> ./install-agent.sh [options]

Options:
  --version <tag>            Release tag binary (default: ${VERSION})
  --arch <arch>              Target arch: amd64|arm64 (default: ${ARCH})
  --manager-endpoint <url>   Manager URL, example: http://10.0.0.20:8089
  --enrollment-token <token> Enrollment token from manager
  --insecure                 Set server.insecure=true (for self-signed / non-TLS)
  --download-url <url>       Override binary tar.gz URL directly
  --force-config             Overwrite existing /etc/xemarify-agent/agent.yaml
  -h, --help                 Show this help

Env alternatives:
  VERSION, ARCH, MANAGER_ENDPOINT, ENROLLMENT_TOKEN, INSECURE_TLS, DOWNLOAD_URL, FORCE_CONFIG
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --arch)
      ARCH="$2"
      shift 2
      ;;
    --manager-endpoint)
      MANAGER_ENDPOINT="$2"
      shift 2
      ;;
    --enrollment-token)
      ENROLLMENT_TOKEN="$2"
      shift 2
      ;;
    --insecure)
      INSECURE_TLS="true"
      shift
      ;;
    --download-url)
      DOWNLOAD_URL="$2"
      shift 2
      ;;
    --force-config)
      FORCE_CONFIG="true"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
done

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root (use sudo)."
  exit 1
fi

if [[ -z "${MANAGER_ENDPOINT}" ]]; then
  echo "MANAGER_ENDPOINT is required."
  exit 1
fi

if [[ "${FORCE_CONFIG}" == "true" || ! -f "${CONFIG_PATH}" ]]; then
  if [[ -z "${ENROLLMENT_TOKEN}" ]]; then
    echo "ENROLLMENT_TOKEN is required for first install/registration."
    exit 1
  fi
fi

if ! command -v tar >/dev/null 2>&1; then
  echo "tar is required but not found"
  exit 1
fi

if [[ -z "${DOWNLOAD_URL}" ]]; then
  DOWNLOAD_URL="https://github.com/hildanku/xemarify/releases/download/${VERSION}/xemarify-agent_${VERSION#v}_linux_${ARCH}.tar.gz"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

archive="${tmpdir}/xemarify-agent.tar.gz"

echo "Downloading agent binary: ${DOWNLOAD_URL}"
if command -v curl >/dev/null 2>&1; then
  curl -fL "${DOWNLOAD_URL}" -o "${archive}"
elif command -v wget >/dev/null 2>&1; then
  wget -O "${archive}" "${DOWNLOAD_URL}"
else
  echo "curl or wget is required"
  exit 1
fi

mkdir -p "${tmpdir}/extract"
tar -xzf "${archive}" -C "${tmpdir}/extract"

binary_source="$(find "${tmpdir}/extract" -type f -name 'xemarify-agent' | head -n 1)"
if [[ -z "${binary_source}" ]]; then
  echo "xemarify-agent binary not found in archive"
  exit 1
fi

install -m 0755 "${binary_source}" "${BIN_PATH}"
mkdir -p "${CONFIG_DIR}" "${STATE_DIR}"

if [[ "${FORCE_CONFIG}" == "true" || ! -f "${CONFIG_PATH}" ]]; then
  if [[ -f "${TEMPLATE_CONFIG}" ]]; then
    cp "${TEMPLATE_CONFIG}" "${CONFIG_PATH}"
  else
    cat > "${CONFIG_PATH}" <<'EOF'
server:
  endpoint: "__MANAGER_ENDPOINT__"
  insecure: __INSECURE_TLS__

enrollment_token: "__ENROLLMENT_TOKEN__"

disk_buffer:
  path: "/var/lib/xemarify-agent/spool/events.log"
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
  fi

  sed -i "s|__MANAGER_ENDPOINT__|${MANAGER_ENDPOINT}|g" "${CONFIG_PATH}"
  sed -i "s|__ENROLLMENT_TOKEN__|${ENROLLMENT_TOKEN}|g" "${CONFIG_PATH}"
  sed -i "s|__INSECURE_TLS__|${INSECURE_TLS}|g" "${CONFIG_PATH}"
  chmod 600 "${CONFIG_PATH}"
fi

if [[ -f "${TEMPLATE_SERVICE}" ]]; then
  install -m 0644 "${TEMPLATE_SERVICE}" "${SERVICE_PATH}"
else
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
fi

systemctl daemon-reload
systemctl enable --now xemarify-agent

echo "Xemarify agent installed successfully."
echo "Config: ${CONFIG_PATH}"
echo "Service status command: systemctl status xemarify-agent"
