#!/usr/bin/env bash
set -euo pipefail

REPO="AetherRuin/terraless-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/aetherruin-agent"
SERVICE_FILE="/etc/systemd/system/terraless-agent.service"

echo "=== TerraLess Host Agent Installer ==="

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64)  BINARY_ARCH="amd64" ;;
  aarch64) BINARY_ARCH="arm64" ;;
  arm64)   BINARY_ARCH="arm64" ;;
  *) echo "Unsupported architecture: ${ARCH}"; exit 1 ;;
esac

LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "latest")
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/terraless-agent-linux-${BINARY_ARCH}"

echo "Downloading TerraLess Agent (${LATEST_TAG}) for linux-${BINARY_ARCH}..."
curl -sSL "${DOWNLOAD_URL}" -o "${INSTALL_DIR}/terraless-agent"
chmod +x "${INSTALL_DIR}/terraless-agent"

mkdir -p "${CONFIG_DIR}"

if [ ! -f "${SERVICE_FILE}" ]; then
  echo "Installing systemd service..."
  curl -sSL "https://raw.githubusercontent.com/${REPO}/main/terraless-agent.service" -o "${SERVICE_FILE}"
  systemctl daemon-reload
  systemctl enable terraless-agent
fi

echo "=== Installation complete ==="
echo "Configure your agent at ${CONFIG_DIR}/config.json and start with: systemctl start terraless-agent"
