#!/usr/bin/env bash
# Frontend deployment script for EchoMe
# Usage: deploy_fe.sh <release_tar_path> <git_sha>
set -euo pipefail

if [ "${1:-}" = "" ]; then
  echo "[deploy_fe] ERROR: release tar path required" >&2
  exit 1
fi

RELEASE_TAR="$1"
GIT_SHA="${2:-unknown}"

DEST_DIR="${FE_DEPLOY_PATH:-/opt/echome-fe}"
RELEASES_DIR="$DEST_DIR/releases"
CURRENT_LINK="$DEST_DIR/current"
BACKUP_DIR="$DEST_DIR/backups"
TS=$(date +%Y%m%d-%H%M%S)
SHORT_SHA=${GIT_SHA:0:8}
TARGET_RELEASE="$RELEASES_DIR/${TS}-${SHORT_SHA}"

echo "[deploy_fe] Starting FE deployment: sha=$GIT_SHA dest=$DEST_DIR"

mkdir -p "$RELEASES_DIR" "$BACKUP_DIR"

echo "[deploy_fe] Extracting release to $TARGET_RELEASE"
mkdir -p "$TARGET_RELEASE"
tar -xzf "$RELEASE_TAR" -C "$TARGET_RELEASE"

if [ -L "$CURRENT_LINK" ] || [ -d "$CURRENT_LINK" ]; then
  echo "[deploy_fe] Backing up current version"
  CURR_REAL=$(readlink -f "$CURRENT_LINK" || echo "")
  if [ -n "$CURR_REAL" ]; then
    cp -al "$CURR_REAL" "$BACKUP_DIR/${TS}-${SHORT_SHA}" || true
  fi
fi

echo "[deploy_fe] Switching current symlink"
ln -sfn "$TARGET_RELEASE" "$CURRENT_LINK"

cd "$CURRENT_LINK"

# Ensure pnpm exists
if ! command -v pnpm >/dev/null 2>&1; then
  echo "[deploy_fe] pnpm not found, attempting to enable via corepack"
  if command -v corepack >/dev/null 2>&1; then
    corepack enable || true
  fi
fi

echo "[deploy_fe] Installing production dependencies"
if command -v pnpm >/dev/null 2>&1; then
  pnpm install --frozen-lockfile --prod
else
  echo "[deploy_fe] WARNING: pnpm not found; falling back to npm (may be slower)" >&2
  npm install --omit=dev
fi

APP_NAME="echome-fe"
PORT="${FE_PORT:-3000}"

START_CMD="pnpm start -- -p $PORT"

if command -v pm2 >/dev/null 2>&1; then
  echo "[deploy_fe] Using pm2 to (re)start $APP_NAME"
  if pm2 describe "$APP_NAME" >/dev/null 2>&1; then
    pm2 restart "$APP_NAME" --update-env --time || pm2 restart "$APP_NAME"
  else
    pm2 start bash --name "$APP_NAME" -- -c "$START_CMD"
  fi
  pm2 save || true
else
  echo "[deploy_fe] pm2 not installed; starting with nohup (no auto-restart)" >&2
  nohup bash -c "$START_CMD" > "$DEST_DIR/${APP_NAME}.log" 2>&1 &
fi

echo "[deploy_fe] Deployment finished successfully"