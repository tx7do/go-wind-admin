#!/usr/bin/env bash
################################################################################
## Docker Compose 启动脚本 - 完整应用版本（应用 + 依赖）
##
## 功能：
##   启动完整的 Docker Compose 应用，包括主应用服务和所有依赖
##
## 启动的服务：
##   - 主应用服务（根据 docker-compose.yml 定义）
##   - PostgreSQL 数据库
##   - Redis 缓存
##   - MinIO 对象存储
##   - Jaeger 分布式追踪
##
## Compose 文件：
##   使用 docker-compose.yml 或 docker-compose.yaml（项目根目录）
##
## 使用场景：
##   1. 完整的本地开发环境
##   2. 快速验收测试
##   3. 生产环境部署
##   4. 一键启动所有服务
##
## 用法：
##   bash scripts/full_deploy.sh
##
## 环境变量：
##   APP_ROOT        数据卷根目录 (默认: /root/app)
##                   目录结构: APP_ROOT/postgres, APP_ROOT/redis 等
##
## 示例：
##   # 使用默认设置
##   bash scripts/full_deploy.sh
##
##   # 自定义数据目录
##   APP_ROOT=/opt/app bash scripts/full_deploy.sh
##
## 相关脚本：
##   - libs_only.sh  仅启动依赖，不启动应用
##
################################################################################
set -euo pipefail
IFS=$'\n\t'

# 可通过环境变量覆盖，默认 /root/app
APP_ROOT=${APP_ROOT:-/root/app}
deps=(postgres redis etcd minio jaeger)

# 切换到脚本所在目录的上一级（项目根）
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_dir/.." || { echo "Failed to cd to repo root ($script_dir/..)" >&2; exit 1; }

# 检查 docker-compose 文件
if [ ! -f docker-compose.yml ] && [ ! -f docker-compose.yaml ]; then
  echo "No docker-compose.yml or docker-compose.yaml found in project root." >&2;
  exit 1
fi

# 检查是否为 root，或是否有 sudo
is_root=false
if [ "$(id -u)" -eq 0 ]; then
  is_root=true
fi
has_sudo=false
if command -v sudo >/dev/null 2>&1; then
  has_sudo=true
fi

# 创建目录并设置权限（尽量使用 sudo 当非 root 时）
mkdir -p "$APP_ROOT"
for dep in "${deps[@]}"; do
  echo "Preparing $dep..."
  mkdir -p "$APP_ROOT/$dep"
  if [ "$is_root" = true ]; then
    chown -R 1001:1001 "$APP_ROOT/$dep"
  elif [ "$has_sudo" = true ]; then
    sudo chown -R 1001:1001 "$APP_ROOT/$dep"
  else
    echo "Warning: not root and sudo not available; skipping chown for $APP_ROOT/$dep" >&2
  fi
done

# 选择 docker compose 命令（支持 v2 插件或旧的 docker-compose 可执行文件）
DOCKER_COMPOSE_CMD=()
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  DOCKER_COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  DOCKER_COMPOSE_CMD=(docker-compose)
else
  echo "Neither 'docker compose' plugin nor 'docker-compose' found. Please install docker compose." >&2
  exit 1
fi

# 启动服务
echo "Bringing up services with: $DOCKER_COMPOSE_CMD up -d --force-recreate"
${DOCKER_COMPOSE_CMD[@]} up -d --force-recreate