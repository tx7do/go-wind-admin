#!/usr/bin/env bash
# PostgreSQL 定期备份脚本（等保二级"本地备份"要求）。
#
# 用法（cron 示例，每天 02:00）：
#   0 2 * * * /path/to/pg_backup.sh >> /var/log/pg_backup.log 2>&1
#
# 环境变量：
#   PG_CONTAINER   容器部署时的 postgres 容器名（默认 citus-server-standalone；
#                  存在则用 docker exec 容器内 pg_dump，否则用本地 pg_dump）
#   PG_HOST/PG_PORT/PG_USER/PG_PASSWORD/PG_DATABASE  本地直连参数
#                  （默认 host=localhost port=5432 user=postgres password=*Abcd12345 dbname=gwa）
#   BACKUP_DIR     备份存放目录（默认 ./pg-backup）
#   KEEP_COPIES    轮换保留份数（默认 30，配合每天一备即满足 ≥30 天恢复点）
set -euo pipefail

# Git Bash(MSYS) 环境防路径转换：/tmp 等容器内路径不被翻译成 Windows 路径
export MSYS_NO_PATHCONV=1

PG_CONTAINER="${PG_CONTAINER:-citus-server-standalone}"
PG_HOST="${PG_HOST:-localhost}"
PG_PORT="${PG_PORT:-5432}"
PG_USER="${PG_USER:-postgres}"
PG_PASSWORD="${PG_PASSWORD:-*Abcd12345}"
PG_DATABASE="${PG_DATABASE:-gwa}"
BACKUP_DIR="${BACKUP_DIR:-./pg-backup}"
KEEP_COPIES="${KEEP_COPIES:-30}"

mkdir -p "$BACKUP_DIR"
STAMP=$(date +%Y%m%d-%H%M%S)
OUT="$BACKUP_DIR/${PG_DATABASE}-${STAMP}.dump"

if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -qx "$PG_CONTAINER"; then
    echo "[$(date '+%F %T')] dumping via docker exec $PG_CONTAINER ..."
    docker exec -e PGPASSWORD="$PG_PASSWORD" "$PG_CONTAINER" \
        pg_dump -U "$PG_USER" -d "$PG_DATABASE" -Fc -f "/tmp/${PG_DATABASE}-${STAMP}.dump"
    docker cp "$PG_CONTAINER:/tmp/${PG_DATABASE}-${STAMP}.dump" "$OUT"
    docker exec "$PG_CONTAINER" rm -f "/tmp/${PG_DATABASE}-${STAMP}.dump"
else
    echo "[$(date '+%F %T')] dumping via local pg_dump ..."
    PGPASSWORD="$PG_PASSWORD" pg_dump \
        -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DATABASE" -Fc -f "$OUT"
fi

SIZE=$(du -h "$OUT" | cut -f1)
echo "[$(date '+%F %T')] backup written: $OUT ($SIZE)"

# 轮换：只保留最近 KEEP_COPIES 份
ls -1t "$BACKUP_DIR"/"${PG_DATABASE}"-*.dump 2>/dev/null | tail -n +$((KEEP_COPIES + 1)) | while read -r f; do
    echo "[$(date '+%F %T')] rotating out: $f"
    rm -f "$f"
done

echo "[$(date '+%F %T')] done. current copies: $(ls -1 "$BACKUP_DIR"/"${PG_DATABASE}"-*.dump 2>/dev/null | wc -l)"
