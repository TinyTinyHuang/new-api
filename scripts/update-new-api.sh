#!/usr/bin/env bash
# 云服务器一键更新 new-api（feat/prompt-audit + SQLite）
# 用法（必须用 bash，不要用 sh）：
#   sed -i 's/\r$//' update-new-api.sh   # 若从 Windows 拷过来，先去掉 CRLF
#   bash update-new-api.sh
#   bash update-new-api.sh --force    # 没有新提交也重新构建并重启
#
if [ -z "${BASH_VERSION-}" ]; then
  exec bash "$0" "$@"
fi
#
# 不会删除 /data 挂载，不会 docker rm -v。失败时回滚到上一版镜像。

set -euo pipefail

# ======== 按部署情况修改（留空则尽量自动检测）========
CONTAINER_NAME="new-api"
IMAGE_NAME="new-api:prompt-audit"
PREV_IMAGE_NAME="new-api:prompt-audit-prev"
BRANCH="feat/prompt-audit"
HOST_BIND="127.0.0.1:3000:3000"
TZ_VALUE="Asia/Shanghai"

# 代码目录：脚本放在仓库里时可留空。若脚本单独放在 /root，请写成例如 /opt/new-api
REPO_DIR=""

# SQLite 数据目录（宿主机路径，对应容器 /data）。留空则从正在运行的容器检测
DATA_DIR=""

BACKUP_DIR="/opt/new-api-backup"
KEEP_BACKUPS=7
HEALTH_URL="http://127.0.0.1:3000/api/status"
HEALTH_TRIES=40
HEALTH_INTERVAL=3
# ====================================================

FORCE=0
if [[ "${1:-}" == "--force" ]]; then
  FORCE=1
fi

log() { echo "[$(date '+%F %T')] $*"; }
die() { echo "[$(date '+%F %T')] ERROR: $*" >&2; exit 1; }

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "找不到命令：$1"
}

need_cmd git
need_cmd docker

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

detect_repo_dir() {
  if [[ -n "$REPO_DIR" ]]; then
    return
  fi
  if git -C "$SCRIPT_DIR" rev-parse --show-toplevel >/dev/null 2>&1; then
    REPO_DIR="$(git -C "$SCRIPT_DIR" rev-parse --show-toplevel)"
    return
  fi
  for candidate in /opt/new-api /root/new-api "$HOME/new-api"; do
    if [[ -d "$candidate/.git" ]]; then
      REPO_DIR="$candidate"
      return
    fi
  done
  die "无法确定代码目录，请编辑脚本顶部的 REPO_DIR"
}

detect_data_dir() {
  if [[ -n "$DATA_DIR" ]]; then
    return
  fi
  if docker inspect "$CONTAINER_NAME" >/dev/null 2>&1; then
    DATA_DIR="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Source}}{{end}}{{end}}' "$CONTAINER_NAME")"
  fi
  if [[ -z "$DATA_DIR" && -d /opt/new-api-data ]]; then
    DATA_DIR="/opt/new-api-data"
  fi
  [[ -n "$DATA_DIR" ]] || die "无法确定 SQLite 数据目录，请编辑脚本顶部的 DATA_DIR"
}

container_is_running() {
  [[ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER_NAME" 2>/dev/null || true)" == "true" ]]
}

health_ok() {
  local body=""
  if command -v curl >/dev/null 2>&1; then
    body="$(curl -fsS --max-time 5 "$HEALTH_URL" 2>/dev/null || true)"
  else
    body="$(wget -q -O - "$HEALTH_URL" 2>/dev/null || true)"
  fi
  [[ "$body" == *'"success"'* ]]
}

wait_healthy() {
  local i
  for i in $(seq 1 "$HEALTH_TRIES"); do
    if health_ok; then
      log "健康检查通过（$HEALTH_URL）"
      return 0
    fi
    sleep "$HEALTH_INTERVAL"
  done
  return 1
}

run_container() {
  local image="$1"
  docker run -d \
    --name "$CONTAINER_NAME" \
    --restart unless-stopped \
    -p "$HOST_BIND" \
    -e "TZ=$TZ_VALUE" \
    -v "$DATA_DIR":/data \
    "$image"
}

backup_data() {
  local stamp dest
  stamp="$(date '+%Y%m%d-%H%M%S')"
  dest="$BACKUP_DIR/$stamp"
  mkdir -p "$dest"
  log "备份 SQLite 数据到 $dest"
  cp -a "$DATA_DIR"/. "$dest"/
  if [[ ! -f "$dest/one-api.db" && ! -f "$dest/one-api.db-wal" ]]; then
    log "警告：备份目录里没看到 one-api.db，请确认 DATA_DIR=$DATA_DIR 是否正确"
  fi
  # 只保留最近 N 份
  ls -1dt "$BACKUP_DIR"/*/ 2>/dev/null | tail -n +$((KEEP_BACKUPS + 1)) | xargs -r rm -rf
}

rollback() {
  log "新版本启动失败，回滚到 $PREV_IMAGE_NAME"
  docker rm -f "$CONTAINER_NAME" >/dev/null 2>&1 || true
  if ! docker image inspect "$PREV_IMAGE_NAME" >/dev/null 2>&1; then
    die "没有上一版镜像，无法自动回滚。数据仍在 $DATA_DIR，请手动启动旧容器"
  fi
  run_container "$PREV_IMAGE_NAME"
  if wait_healthy; then
    die "已回滚到上一版并恢复服务。请查看：docker logs --tail 80 $CONTAINER_NAME"
  fi
  die "回滚后健康检查仍失败。数据在 $DATA_DIR，请执行：docker logs --tail 80 $CONTAINER_NAME"
}

detect_repo_dir
detect_data_dir

[[ -d "$REPO_DIR/.git" ]] || die "REPO_DIR 不是 git 仓库：$REPO_DIR"
[[ -d "$DATA_DIR" ]] || die "数据目录不存在：$DATA_DIR"
docker inspect "$CONTAINER_NAME" >/dev/null 2>&1 || die "找不到容器 $CONTAINER_NAME，请确认名称"

if [[ "$(docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$CONTAINER_NAME")" == *SQL_DSN=* ]]; then
  die "检测到 SQL_DSN，当前脚本按 SQLite 编写。若已改用 MySQL/Postgres，请不要用此脚本"
fi

log "仓库：$REPO_DIR"
log "分支：$BRANCH"
log "数据目录：$DATA_DIR"
log "容器：$CONTAINER_NAME"
log "镜像：$IMAGE_NAME"

cd "$REPO_DIR"

if [[ -n "$(git status --porcelain)" ]]; then
  die "代码目录有未提交修改，自动更新已中止，以免覆盖服务器上的手工改动"
fi

log "拉取 $BRANCH ..."
git fetch origin
git checkout "$BRANCH"
OLD_SHA="$(git rev-parse HEAD)"
git pull --ff-only origin "$BRANCH"
NEW_SHA="$(git rev-parse HEAD)"
SHORT_SHA="$(git rev-parse --short HEAD)"

if [[ "$OLD_SHA" == "$NEW_SHA" && "$FORCE" -ne 1 ]]; then
  log "已是最新提交 $SHORT_SHA，无需更新。若要强制重建，请加 --force"
  exit 0
fi

log "构建镜像（提交 $SHORT_SHA）..."
if docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
  docker tag "$IMAGE_NAME" "$PREV_IMAGE_NAME"
fi
docker build -t "$IMAGE_NAME" -t "${IMAGE_NAME}-${SHORT_SHA}" "$REPO_DIR"

log "停止容器并备份数据库..."
if container_is_running; then
  docker stop "$CONTAINER_NAME"
fi
backup_data

log "用新镜像替换容器（保留 $DATA_DIR 挂载）..."
docker rm "$CONTAINER_NAME"
run_container "$IMAGE_NAME"

if wait_healthy; then
  log "更新完成：$OLD_SHA -> $NEW_SHA"
  log "备份位置：$BACKUP_DIR"
  exit 0
fi

log "新容器日志："
docker logs --tail 80 "$CONTAINER_NAME" || true
rollback
