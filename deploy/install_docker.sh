#!/usr/bin/env bash
set -euo pipefail
# 安装 Docker & docker compose plugin (Ubuntu/Debian)
if ! command -v docker >/dev/null 2>&1; then
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
fi
sudo usermod -aG docker "$USER" || true
mkdir -p /opt/echome
echo "Docker installed. You may need to re-login for group changes."
