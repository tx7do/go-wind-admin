#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

################################################################################
##                    开发环境 (DEV) 初始化脚本
##
## 功能：安装基础工具、Docker、Go、代码生成插件、项目脚手架
##
## 使用方式：
##   bash scripts/install_unix_dev.sh
##
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ============================================================================
# 加载函数库
# ============================================================================

_LIB_DIR="${SCRIPT_DIR}/lib"

# 按依赖顺序加载
source "${_LIB_DIR}/common-utils.sh"
source "${_LIB_DIR}/os-utils.sh"
source "${_LIB_DIR}/basic-tools.sh"
source "${_LIB_DIR}/docker-utils.sh"
source "${_LIB_DIR}/go-utils.sh"
source "${_LIB_DIR}/host-utils.sh"

# 错误处理
trap 'err_trap $LINENO' ERR

# ============================================================================
# 开发者专属安装函数
# ============================================================================

# go install 统一安装方法（带已安装检测）
# 参数: $@ - 要安装的 Go 包列表
# 安全策略: 已安装的工具跳过，不重新安装
go_install_packages() {
    if [ $# -eq 0 ]; then
        warn "go_install_packages: 没有指定要安装的包"
        return 0
    fi

    for package in "$@"; do
        # 从包路径提取工具名
        # 例: google.golang.org/protobuf/cmd/protoc-gen-go@latest → protoc-gen-go
        # 例: github.com/go-kratos/kratos/cmd/kratos/v2@latest → kratos（跳过 /vN 版本后缀）
        local base="${package%@*}"                          # 去掉 @latest
        local last_seg="${base##*/}"                       # 取最后一段
        # 如果最后一段是 vN 版本后缀（如 v2, v3），则取前一段作为工具名
        local tool_name
        if [[ "$last_seg" =~ ^v[0-9]+$ ]]; then
            local without_last="${base%/*}"
            tool_name="${without_last##*/}"
        else
            tool_name="$last_seg"
        fi

        if command -v "$tool_name" >/dev/null 2>&1; then
            log "  [跳过] $tool_name 已安装"
        else
            log "  [安装] $package"
            if go install "$package"; then
                success "成功安装: $tool_name"
            else
                warn "✗ 安装失败: $tool_name"
            fi
        fi
    done
}

install_go_plugins() {
    log "安装 Protobuf 编译器插件..."

    go_install_packages \
        "google.golang.org/protobuf/cmd/protoc-gen-go@latest" \
        "google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest" \
        "github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest" \
        "github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest" \
        "github.com/google/gnostic/cmd/protoc-gen-openapi@latest" \
        "github.com/envoyproxy/protoc-gen-validate@latest" \
        "github.com/menta2k/protoc-gen-redact/v3@latest" \
        "github.com/tx7do/go-wind-toolkit/protoc-gen-typescript-http@latest"
}

install_go_cli_tools() {
    log "安装 CLI 脚手架工具..."

    go_install_packages \
        "github.com/go-kratos/kratos/cmd/kratos/v2@latest" \
        "github.com/google/gnostic@latest" \
        "github.com/bufbuild/buf/cmd/buf@latest" \
        "entgo.io/ent/cmd/ent@latest" \
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest" \
        "github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest"
}

install_go_dev_tools() {
    log "检查 Go 开发环境..."

    # 1. 确保 Go 已安装
    if ! command -v go >/dev/null 2>&1; then
        warn "未检测到 Go，跳过 Go 插件和工具安装。"
        return 0
    fi

    # 2. 配置开发加速（仅当未配置时）
    local current_proxy=$(go env GOPROXY 2>/dev/null || echo "")
    if [[ "$current_proxy" != *"goproxy.io"* ]]; then
        log "配置 Go 环境变量..."
        go env -w GOPROXY=https://goproxy.io,direct
        go env -w GO111MODULE=on
    else
        log "  [跳过] GOPROXY 已配置: $current_proxy"
    fi

    # 3. 安装 Protobuf 插件（带检测）
    install_go_plugins

    # 4. 安装 CLI 脚手架工具（带检测）
    install_go_cli_tools
}

install_dev_binaries() {
    local os_type=$1
    log "检查开发辅助工具 (Protoc)..."

    # 检查 protoc 是否已安装
    if command -v protoc >/dev/null 2>&1; then
        log "  [跳过] protoc 已安装: $(protoc --version 2>/dev/null || echo '版本未知')"
        return 0
    fi

    log "安装 protoc 编译器..."
    case "$os_type" in
        macos)
            brew install protobuf
            ;;
        ubuntu)
            ${SUDO} apt-get install -y protobuf-compiler
            ;;
        centos|rocky|fedora)
            ${SUDO} dnf install -y protobuf-compiler 2>/dev/null || ${SUDO} yum install -y protobuf-compiler
            ;;
        *)
            warn "不支持的系统类型: $os_type，跳过 protoc 安装"
            ;;
    esac
}

maybe_initialize_hosts() {
    local auto_init="${AUTO_INIT_HOSTS:-false}"

    if [[ ! "$auto_init" =~ ^(1|true|TRUE|yes|YES)$ ]]; then
        log "  [跳过] hosts 初始化未启用 (设置 AUTO_INIT_HOSTS=true 可开启)"
        return 0
    fi

    local hosts_ip="${HOSTS_IP:-127.0.0.1}"
    local hosts_domain_suffix="${HOSTS_DOMAIN_SUFFIX:-.local}"
    local hosts_services="${HOSTS_SERVICES:-postgres mysql redis}"

    read -r -a services <<< "$hosts_services"
    if [ "${#services[@]}" -eq 0 ]; then
        warn "HOSTS_SERVICES 为空，跳过 hosts 初始化"
        return 0
    fi

    log "初始化 hosts 记录..."
    initialize_hosts "$hosts_ip" "$hosts_domain_suffix" "${services[@]}"
}

# ============================================================================
# 主执行流程
# ============================================================================

main() {
    # 1. 检测系统
    local info=$(detect_os_and_package_manager)
    IFS='|' read -r os_type pkg_mgr pkg_cmd docker_setup <<< "$info"

    # 2. 基础工具 (Git, Curl, JQ 等)
    install_basic_tools "$os_type" "$pkg_mgr" "$pkg_cmd"

    # 3. Docker (开发环境需要运行资料库容器)
    install_docker "$pkg_mgr" "$docker_setup"

    # 4. Go 运行时（确保已安装）
    install_golang

    # 5. Go 环境与代码生成插件
    install_go_dev_tools
    install_dev_binaries "$os_type"

    # 6. 可选 hosts 初始化（通过环境变量开启）
    maybe_initialize_hosts

    # 7. 设置开发路径 (确保 GOBIN 在 PATH 中)
    local shell_rc="${TARGET_HOME}/.bashrc"
    [[ "$SHELL" == *"zsh"* ]] && shell_rc="${TARGET_HOME}/.zshrc"

    if ! grep -q "go/bin" "$shell_rc"; then
        echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> "$shell_rc"
        log "已将 GOBIN 加入 $shell_rc，请执行 'source $shell_rc' 生效"
    fi

    log "✅ 开发环境准备完成！"
    log "已安装：Docker, Go 插件, Protoc"
}

main
