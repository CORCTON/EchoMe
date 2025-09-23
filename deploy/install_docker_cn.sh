#!/usr/bin/env bash
# 适用于中国大陆 Ubuntu 系列服务器的 Docker 安装脚本
# 来源 gist 基础上做了健壮性与版本更新处理
set -euo pipefail

if ! command -v apt-get >/dev/null 2>&1; then
  echo "当前脚本仅适用于基于 apt 的发行版 (Ubuntu/Debian)" >&2
  exit 1
fi

echo "[1/5] 更新软件源索引..."
sudo apt-get update -y

echo "[2/5] 安装依赖..."
sudo apt-get install -y \
  apt-transport-https \
  ca-certificates \
  curl \
  gnupg \
  lsb-release \
  software-properties-common

# 使用中科大镜像 (USTC)
MIRROR="https://mirrors.ustc.edu.cn/docker-ce/linux/ubuntu"
DISTRO_CODENAME=$(lsb_release -cs)
ARCH=$(dpkg --print-architecture)

echo "[3/5] 导入 GPG Key..."
curl -fsSL ${MIRROR}/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

if [ ! -s /usr/share/keyrings/docker-archive-keyring.gpg ]; then
  echo "GPG key 下载失败" >&2
  exit 2
fi

echo "[4/5] 添加镜像源..."
echo "deb [arch=${ARCH} signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] ${MIRROR} ${DISTRO_CODENAME} stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt-get update -y

echo "[5/5] 安装 Docker CE 最新稳定版..."
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 安装独立 docker-compose 二进制（如需指定版本可改 COMPOSE_VERSION）
COMPOSE_VERSION=${COMPOSE_VERSION:-"2.27.1"}
if ! command -v docker-compose >/dev/null 2>&1; then
  echo "安装独立 docker-compose ${COMPOSE_VERSION}..."
  sudo curl -L "https://github.com/docker/compose/releases/download/v${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
  sudo chmod +x /usr/local/bin/docker-compose || true
fi

# 添加当前用户进 docker 组
if getent group docker >/dev/null 2>&1; then
  sudo usermod -aG docker "$USER" || true
fi

cat <<INFO
====================================
Docker 安装完成
版本: $(docker --version | sed 's/,.*//')
Compose Plugin: $(docker compose version 2>/dev/null || echo 'N/A')
独立 docker-compose: $(docker-compose version 2>/dev/null | head -n1 || echo 'N/A')
请重新登录或运行: newgrp docker 以立即生效。
====================================
INFO
