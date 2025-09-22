#!/usr/bin/env bash
# Backend deployment script for EchoMe
# Usage: deploy_be.sh <release_tar_path> <git_sha>
set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "[deploy_be] ERROR: release tar path required" >&2
  exit 1
fi

RELEASE_TAR="$1"
GIT_SHA="${2:-unknown}"

DEST_DIR="${BE_DEPLOY_PATH:-/opt/echome-be}"
RELEASES_DIR="$DEST_DIR/releases"
CURRENT_LINK="$DEST_DIR/current"
BACKUP_DIR="$DEST_DIR/backups"
TS=$(date +%Y%m%d-%H%M%S)
SHORT_SHA=${GIT_SHA:0:8}
TARGET_RELEASE="$RELEASES_DIR/${TS}-${SHORT_SHA}"

echo "[deploy_be] Starting BE deployment: sha=$GIT_SHA dest=$DEST_DIR"

mkdir -p "$RELEASES_DIR" "$BACKUP_DIR"

echo "[deploy_be] Extracting release to $TARGET_RELEASE"
mkdir -p "$TARGET_RELEASE"
tar -xzf "$RELEASE_TAR" -C "$TARGET_RELEASE"

if [ -L "$CURRENT_LINK" ] || [ -d "$CURRENT_LINK" ]; then
  echo "[deploy_be] Backing up current version"
  CURR_REAL=$(readlink -f "$CURRENT_LINK" || echo "")
  if [ -n "$CURR_REAL" ]; then
    cp -al "$CURR_REAL" "$BACKUP_DIR/${TS}-${SHORT_SHA}" || true
  fi
fi

echo "[deploy_be] Switching current symlink"
ln -sfn "$TARGET_RELEASE" "$CURRENT_LINK"

cd "$CURRENT_LINK"

chmod +x echome-be || true

CONFIG_FILE="${BE_CONFIG_FILE:-config/etc/config.yaml}"
if [ ! -f "$CONFIG_FILE" ]; then
  echo "[deploy_be] WARNING: config file $CONFIG_FILE not found; using example if exists" >&2
  if [ -f config/etc/config.yaml.example ]; then
    cp config/etc/config.yaml.example "$CONFIG_FILE"
  fi
fi

APP_NAME="echome-be"
START_CMD="./echome-be -f $CONFIG_FILE"

if command -v systemctl >/dev/null 2>&1 && systemctl list-unit-files | grep -q "${APP_NAME}.service"; then
  echo "[deploy_be] Detected systemd service; reloading"
  systemctl daemon-reload || true
  systemctl restart ${APP_NAME}.service
else
  if command -v pm2 >/dev/null 2>&1; then
    echo "[deploy_be] Using pm2 to (re)start $APP_NAME"
    if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
      pm2 restart "$APP_NAME" --update-env --time || pm2 restart "$APP_NAME"
    else
      pm2 start bash --name "$APP_NAME" -- -c "$START_CMD"
    fi
    pm2 save || true
  else
    echo "[deploy_be] Starting with nohup (no process manager)" >&2
    nohup bash -c "$START_CMD" > "$DEST_DIR/${APP_NAME}.log" 2>&1 &
  fi
fi

echo "[deploy_be] Deployment finished successfully"