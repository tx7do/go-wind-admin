#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

################################################################################
##                    开发环境 (DEV) 初始化脚本
##
## 功能：安装基础工具、Docker、Go、代码生成插件、项目脚手架
################################################################################

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
log() { echo "==> $*"; }
warn() { echo "⚠ WARNING: $*" >&2; }

# sudo 检查
SUDO=${SUDO:-}
if [ "$EUID" -ne 0 ]; then SUDO='sudo'; fi
TARGET_USER=${SUDO_USER:-$(whoami)}
TARGET_HOME=$(eval echo "~${TARGET_USER}")

# [复用您原有的 detect_os_and_package_manager 函数内容...]
# 此处省略 detect_os 函数，逻辑保持一致

# ============================================================================
# 开发者专属安装函数
# ============================================================================

# go install 统一安装方法
# 参数: $@ - 要安装的 Go 包列表
go_install_packages() {
    if [ $# -eq 0 ]; then
        warn "go_install_packages: 没有指定要安装的包"
        return 0
    fi

    for package in "$@"; do
        log "安装 Go 包: $package"
        if go install "$package"; then
            log "✓ 成功安装: $package"
        else
            warn "✗ 安装失败: $package"
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
        "github.com/go-kratos/protoc-gen-typescript-http@latest"
}

install_go_cli_tools() {
    log "安装 CLI 脚手架工具..."

    go_install_packages \
        "github.com/go-kratos/kratos/cmd/kratos/v2@latest" \
        "github.com/google/gnostic@latest" \
        "github.com/bufbuild/buf/cmd/buf@latest" \
        "entgo.io/ent/cmd/ent@latest" \
        "github.com/golangci/golangci-lint/cmd/golangci-lint@latest" \
        "github.com/tx7do/go-wind-toolkit/config-exporter/cmd/cfgexp@latest" \
        "github.com/tx7do/go-wind-toolkit/sql-orm/cmd/sql2orm@latest" \
        "github.com/tx7do/go-wind-toolkit/sql-proto/cmd/sql2proto@latest" \
        "github.com/tx7do/go-wind-toolkit/sql-kratos/cmd/sql2kratos@latest" \
        "github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest"
}

install_go_dev_tools() {
    log "配置 Go 开发环境..."

    # 1. 确保 Go 已安装
    if ! command -v go >/dev/null 2>&1; then
        warn "未检测到 Go，请先确保系统已安装 Go。"
        return 1
    fi

    # 2. 配置开发加速 (针对中国大陆或网络环境)
    log "配置 Go 环境变量..."
    go env -w GOPROXY=https://goproxy.io,direct
    go env -w GO111MODULE=on

    # 3. 安装 Protobuf 插件
    install_go_plugins

    # 4. 安装 CLI 脚手架工具
    install_go_cli_tools
}

install_dev_binaries() {
    local os_type=$1
    log "安装开发辅助工具 (Protoc 二进制)..."

    case "$os_type" in
        macos)
            brew install protobuf golangci-lint
            ;;
        ubuntu|centos|rocky|fedora)
            # 安装 protobuf 编译器
            if [[ "$os_type" == "ubuntu" ]]; then
                ${SUDO} apt-get install -y protobuf-compiler
            else
                ${SUDO} dnf install -y protobuf-compiler || ${SUDO} yum install -y protobuf-compiler
            fi
            ;;
    esac
}

# ============================================================================
# 主执行流程
# ============================================================================

main() {
    # 1. 检测系统
    # (此处调用 detect_os_and_package_manager)
    local info=$(detect_os_and_package_manager)
    IFS='|' read -r os_type pkg_mgr pkg_cmd docker_setup <<< "$info"

    # 2. 基础工具 (Git, Curl, JQ 等)
    install_basic_tools "$os_type" "$pkg_mgr" "$pkg_cmd"

    # 3. Docker (开发环境需要运行资料库容器)
    install_docker "$pkg_mgr" "$docker_setup"

    # 4. Go 环境与代码生成插件
    install_go_dev_tools
    install_dev_binaries "$os_type"

    # 5. 设置开发路径 (确保 GOBIN 在 PATH 中)
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
