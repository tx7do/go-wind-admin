#!/usr/bin/env bash

set -euo pipefail

if command -v go >/dev/null 2>&1; then
  echo "go: $(go version)"
else
  echo "go: not installed"
fi

if command -v pm2 >/dev/null 2>&1; then
  echo "pm2: $(pm2 -v)"
else
  echo "pm2: not installed"
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="$script_dir/.."
env_file="$project_root/.env"
project_root="$(cd "$project_root" && pwd)"

err() { printf '%s\n' "$*" >&2; }

# 加载 .env（去掉 CRLF），仅在文件存在且非空时加载
if [ -f "$env_file" ] && [ -s "$env_file" ]; then
  # 自动导出变量到环境，便于 make/go 使用
  set -a
  # shellcheck disable=SC1090
  source <(sed 's/\r$//' "$env_file")
  set +a
else
  err "no .env found at $env_file, continuing without it"
fi

# 进入项目根以保证相对路径一致
pushd "$project_root" >/dev/null

# 构建项目
if ! make build_only; then
  err "make build failed"
  popd >/dev/null
  exit 1
fi

popd >/dev/null

project_name="${PROJECT_NAME:-gwa}"
install_root="${HOME}/app/${project_name}"
app_root="${project_root}/app"

# 收集 app 子目录（只取第一层目录）
mapfile -t apps < <(find "$app_root" -maxdepth 1 -mindepth 1 -type d -print)

# 如果没有服务，直接退出
if [ "${#apps[@]}" -eq 0 ]; then
  err "no apps found under $app_root"
  exit 0
fi

mkdir -p "$install_root"

for app_dir in "${apps[@]}"; do
  app="$(basename "$app_dir")"
  echo "Installing service: $app"

  app_install_root="$install_root/$app"
  bin_src="$app_dir/service/bin/server"
  configs_src_dir="$app_dir/service/configs"
  bin_dest="$app_install_root/service/bin"
  configs_dest="$app_install_root/service/configs"

  mkdir -p "$bin_dest" "$configs_dest"

  # 复制二进制（如果存在）
  if [ -f "$bin_src" ]; then
    cp -f "$bin_src" "$bin_dest/server"
    chmod +x "$bin_dest/server" || true
  else
    err "binary not found: $bin_src (skipping)"
    continue
  fi

  # 拷贝配置文件（如果存在）
  if [ -d "$configs_src_dir" ]; then
    cp -rf "$configs_src_dir"/*.yaml "$configs_dest/" 2>/dev/null || true
  else
    err "configs dir not found: $configs_src_dir"
  fi
done

# 启动/注册到 PM2
for app_dir in "${apps[@]}"; do
  app="$(basename "$app_dir")"
  echo "Starting service: $app"

  app_install_root="$install_root/$app"
  bin_path="$app_install_root/service/bin/server"
  configs_rel="../configs/"

  if [ ! -x "$bin_path" ]; then
    err "executable not found or not executable: $bin_path (skipping)"
    continue
  fi

  # 使用绝对二进制路径启动，传入配置目录作为参数
  # pm2 usage: pm2 start <script> --name <name> --namespace <ns> -- <args...>
  if ! command -v pm2 >/dev/null 2>&1; then
    err "pm2 not installed or not in PATH; please install pm2"
    exit 1
  fi

  pushd "$app_install_root/service/bin" >/dev/null

  # If a process with the same name exists in the given namespace, remove it first
  # to avoid: [PM2][ERROR] Script already launched, add -f option to force re-execution
  if pm2 info "$app" --namespace "$project_name" >/dev/null 2>&1; then
    echo "PM2 process '$app' already exists in namespace '$project_name', deleting before start"
    if ! pm2 delete "$app" --namespace "$project_name" >/dev/null 2>&1; then
      echo "warning: failed to delete existing pm2 process '$app'; attempting force start (-f)"
      pm2 start -f "$bin_path" --name "$app" --namespace "$project_name" -- -c "$configs_rel"
      popd >/dev/null
      continue
    fi
  fi

  pm2 start "$bin_path" --name "$app" --namespace "$project_name" -- -c "$configs_rel"
  popd >/dev/null
done

pm2 save
# 重启同一 namespace 下的所有进程，确保更新生效
pm2 restart all --namespace "$project_name" || true

echo "install and pm2 setup complete for namespace: $project_name"
