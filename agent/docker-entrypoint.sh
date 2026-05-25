#!/bin/sh
set -e

CONFIG_PATH="/etc/xemarify-agent/agent.yaml"
CONFIG_TEMPLATE="/etc/xemarify-agent/agent.yaml.tmpl"

# If template exists and has env placeholders, substitute them
if [ -f "${CONFIG_TEMPLATE}" ]; then
  # Use sed to replace ${ENROLLMENT_TOKEN} and other env vars
  sed \
    -e "s|\${ENROLLMENT_TOKEN}|${XEMARIFY_ENROLLMENT_TOKEN:-}|g" \
    -e "s|\${SERVER_ENDPOINT}|${XEMARIFY_SERVER_ENDPOINT:-http://manager:8089}|g" \
    "${CONFIG_TEMPLATE}" > "${CONFIG_PATH}"
  echo "[entrypoint] Config generated from template."
elif [ -f "${CONFIG_PATH}" ]; then
  # Config already exists, do in-place substitution if needed
  sed -i \
    -e "s|\${ENROLLMENT_TOKEN}|${XEMARIFY_ENROLLMENT_TOKEN:-}|g" \
    -e "s|\${SERVER_ENDPOINT}|${XEMARIFY_SERVER_ENDPOINT:-http://manager:8089}|g" \
    "${CONFIG_PATH}"
  echo "[entrypoint] Config updated with env vars."
fi

exec /usr/local/bin/xemarify-agent "$@"