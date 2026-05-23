#!/usr/bin/env bash
################################################################################
## Docker Compose 启动脚本 - 仅依赖版本（不包含应用）
##
## 功能：
##   启动仅包含依赖服务的 Docker Compose，不启动主应用
##
## 启动的服务（仅依赖）：
##   - PostgreSQL 数据库
##   - Redis 缓存
##   - MinIO 对象存储
##   - Jaeger 分布式追踪
##
## 不启动的服务：
##   - 主应用服务（应该本地运行）
##
## Compose 文件：
##   使用 docker-compose.libs.yaml（项目根目录）
##   此文件只包含依赖服务的定义
##
## 使用场景：
##   1. 本地开发：Docker 运行依赖，本地运行应用代码
##   2. 调试：更容易调试应用代码
##   3. 快速迭代：无需重启应用容器
##   4. IDE 开发：在 IDE 中直接运行和调试
##
## 用法：
##   bash scripts/libs_only.sh
##
## 环境变量：
##   APP_ROOT        数据卷根目录 (默认: /root/app)
##                   目录结构: APP_ROOT/postgres, APP_ROOT/redis 等
##
##   COMPOSE_FILE    Compose 文件路径 (默认: docker-compose.libs.yaml)
##                   指向仅包含依赖的 Compose 配置
##
## 示例：
##   # 启动依赖服务
##   bash scripts/libs_only.sh
##
##   # 然后本地运行应用（在另一个终端）
##   cd app && go run main.go
##
##   # 自定义数据目录和 Compose 文件
##   APP_ROOT=/opt/app COMPOSE_FILE=compose-deps.yaml bash scripts/libs_only.sh
##
## 相关脚本：
##   - full_deploy.sh  启动完整应用（包含应用服务）
##
## 工作流示例：
##   # 终端 1: 启动依赖
##   bash scripts/libs_only.sh
##
##   # 终端 2: 启动应用代码
##   cd app/admin/service
##   go run main.go
##
################################################################################
set -euo pipefail
IFS=$'\n\t'

# 可通过环境变量覆盖目标目录和 compose 文件
APP_ROOT=${APP_ROOT:-/root/app}
COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.libs.yaml}
deps=(postgres redis etcd minio jaeger)

# 切换到脚本所在目录的上一级（项目根）
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$script_dir/.."
cd "$repo_root" || { echo "Failed to cd to repo root: $repo_root" >&2; exit 1; }

echo "Repo root: $repo_root"
echo "App root: $APP_ROOT"
echo "Compose file: $COMPOSE_FILE"

# 检查 docker-compose 文件存在
if [ ! -f "$COMPOSE_FILE" ]; then
  echo "Compose file not found: $COMPOSE_FILE" >&2
  exit 1
fi

# 检查权限 / sudo
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
  target="$APP_ROOT/$dep"
  echo "Preparing $dep at $target"
  mkdir -p "$target"
  if [ "$is_root" = true ]; then
    chown -R 1001:1001 "$target" || true
  elif [ "$has_sudo" = true ]; then
    sudo chown -R 1001:1001 "$target" || true
  else
    echo "Warning: not root and sudo not available; skipping chown for $target" >&2
  fi
done

# 选择 docker compose 命令（支持 v2 插件或旧的 docker-compose）
DOCKER_COMPOSE_CMD=()
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  DOCKER_COMPOSE_CMD=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  DOCKER_COMPOSE_CMD=(docker-compose)
else
  echo "Neither 'docker compose' plugin nor 'docker-compose' found. Please install docker compose." >&2
  exit 1
fi

# 部署
echo "Bringing up services with: $DOCKER_COMPOSE_CMD -f $COMPOSE_FILE up -d --force-recreate"
${DOCKER_COMPOSE_CMD[@]} -f "$COMPOSE_FILE" up -d --force-recreate