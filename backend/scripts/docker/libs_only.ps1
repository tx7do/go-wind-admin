<#
.SYNOPSIS
Docker Compose 启动脚本 - 仅依赖版本（不包含应用）（Windows PowerShell 版）

.DESCRIPTION
启动仅包含依赖服务的 Docker Compose，不启动主应用

.PARAMETER AppRoot
数据卷根目录路径 (默认: C:\app)
目录结构: APP_ROOT\postgres, APP_ROOT\redis 等

.PARAMETER ComposeFile
Compose 文件路径 (默认: docker-compose.libs.yaml)
指向仅包含依赖的 Compose 配置

.EXAMPLE
# 启动依赖服务（使用默认配置）
.\libs_only.ps1

# 自定义数据目录
.\libs_only.ps1 -AppRoot "D:\app"

# 自定义 Compose 文件
.\libs_only.ps1 -ComposeFile "compose-deps.yaml"

# 完整自定义
.\libs_only.ps1 -AppRoot "D:\myapp" -ComposeFile "custom-compose.yaml"

.NOTES
启动的服务（仅依赖）：
  - PostgreSQL 数据库
  - Redis 缓存
  - MinIO 对象存储
  - Jaeger 分布式追踪

不启动的服务：
  - 主应用服务（应该本地运行）

使用场景：
  1. 本地开发：Docker 运行依赖，本地运行应用代码
  2. 调试：更容易调试应用代码
  3. 快速迭代：无需重启应用容器
  4. IDE 开发：在 IDE 中直接运行和调试

工作流示例：
  # PowerShell 1: 启动依赖
  .\libs_only.ps1

  # PowerShell 2: 启动应用代码
  gow run admin

相关脚本：
  - full_deploy.ps1  启动完整应用（包含应用服务）

#>

param(
    [Parameter(Mandatory=$false)]
    [string]$AppRoot = "C:\app",

    [Parameter(Mandatory=$false)]
    [string]$ComposeFile = "docker-compose.libs.yaml"
)

# ============================================================================
# 函数定义
# ============================================================================

function Log {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Warn {
    param([string]$Message)
    Write-Host "⚠ WARNING: $Message" -ForegroundColor Yellow
}

function ErrorLog {
    param([string]$Message)
    Write-Host "❌ ERROR: $Message" -ForegroundColor Red
}

function EnsureDirectoryExists {
    param([string]$Path)

    if (-not (Test-Path -Path $Path)) {
        Log "Creating directory: $Path"
        New-Item -Path $Path -ItemType Directory -Force | Out-Null
        Log "  [OK] Directory created"
    } else {
        Log "  [SKIP] Directory already exists: $Path"
    }
}

function Get-DockerComposeCommand {
    <#
    .SYNOPSIS
    检测和获取 Docker Compose 命令

    .DESCRIPTION
    优先使用 Docker CLI 中的 compose 插件（docker compose），
    失败则降级到独立的 docker-compose 命令

    .RETURNS
    返回可用的 docker compose 命令路径或 $null
    #>

    try {
        # 尝试 docker compose 插件（推荐）
        $version = docker compose version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Log "Found Docker Compose plugin: docker compose"
            return "docker"
        }
    } catch {
        # 继续尝试下一个方法
    }

    try {
        # 尝试独立的 docker-compose 命令
        $version = docker-compose --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Log "Found docker-compose command: docker-compose"
            return "docker-compose"
        }
    } catch {
        # 继续
    }

    ErrorLog "Neither 'docker compose' plugin nor 'docker-compose' found"
    ErrorLog "Please install Docker Desktop or docker-compose"
    return $null
}

# ============================================================================
# 主程序开始
# ============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

Log "========================================"
Log "  Docker Compose - Libs Only（仅依赖）"
Log "========================================"
Log ""

# 获取项目根目录
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir

Log "Script dir: $scriptDir"
Log "Repo root: $repoRoot"
Log "App root: $AppRoot"
Log "Compose file: $ComposeFile"
Log ""

# 进入项目根目录
try {
    Push-Location $repoRoot
    Log "Changed to repo root: $repoRoot"
} catch {
    ErrorLog "Failed to change directory to: $repoRoot"
    exit 1
}

# 检查 Compose 文件是否存在
if (-not (Test-Path -Path $ComposeFile -PathType Leaf)) {
    ErrorLog "Compose file not found: $ComposeFile"
    Pop-Location
    exit 1
}
Log "Compose file found: $ComposeFile"
Log ""

# 创建数据卷目录
Log "Creating data directories..."
$dependencies = @('postgres', 'redis', 'etcd', 'minio', 'jaeger')

foreach ($dep in $dependencies) {
    $targetPath = Join-Path $AppRoot $dep
    EnsureDirectoryExists $targetPath
}

Log ""

# 获取 Docker Compose 命令
Log "Checking Docker Compose availability..."
$dockerComposeCmd = Get-DockerComposeCommand

if ($null -eq $dockerComposeCmd) {
    Pop-Location
    exit 1
}

Log ""

# 构建命令
if ($dockerComposeCmd -eq "docker") {
    $command = "docker compose -f $ComposeFile up -d --force-recreate"
} else {
    $command = "$dockerComposeCmd -f $ComposeFile up -d --force-recreate"
}

Log "Executing: $command"
Log ""

try {
    # 执行 Docker Compose 命令
    Invoke-Expression $command

    if ($LASTEXITCODE -eq 0) {
        Log ""
        Log "========================================"
        Log "  启动成功！"
        Log "========================================"
        Log ""
        Log "运行中的服务："

        # 显示运行中的容器
        if ($dockerComposeCmd -eq "docker") {
            docker compose -f $ComposeFile ps
        } else {
            docker-compose -f $ComposeFile ps
        }

        Log ""
        Log "后续步骤："
        Log "1. 在另一个 PowerShell 中启动应用："
        Log "   cd app"
        Log "   go run main.go"
        Log ""
        Log "2. 或在 IDE 中打开项目并按 F5 调试"
        Log ""
    } else {
        ErrorLog "Docker Compose command failed (exit code: $LASTEXITCODE)"
        Pop-Location
        exit 1
    }
} catch {
    ErrorLog "Failed to execute Docker Compose: $_"
    Pop-Location
    exit 1
}

# 返回之前的目录
Pop-Location

Log "========================================" -ForegroundColor Green

