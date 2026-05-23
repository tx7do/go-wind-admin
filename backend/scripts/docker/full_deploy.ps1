<#
.SYNOPSIS
Docker Compose 启动脚本 - 完整应用版本（应用 + 依赖）（Windows PowerShell 版）

.DESCRIPTION
启动完整的 Docker Compose 应用，包括主应用服务和所有依赖

.PARAMETER AppRoot
数据卷根目录路径 (默认: C:\app)
目录结构: APP_ROOT\postgres, APP_ROOT\redis 等

.PARAMETER ComposeFile
Compose 文件路径 (默认: docker-compose.yml)
指向完整应用的 Compose 配置

.EXAMPLE
# 启动完整应用（使用默认配置）
.\full_deploy.ps1

# 自定义数据目录
.\full_deploy.ps1 -AppRoot "D:\app"

# 自定义 Compose 文件
.\full_deploy.ps1 -ComposeFile "docker-compose.yaml"

# 完整自定义
.\full_deploy.ps1 -AppRoot "D:\myapp" -ComposeFile "custom-compose.yaml"

.NOTES
启动的服务（完整）：
  - 主应用服务（根据 docker-compose.yml 定义）
  - PostgreSQL 数据库
  - Redis 缓存
  - MinIO 对象存储
  - Jaeger 分布式追踪

使用 Compose 文件：
  - 使用 docker-compose.yml 或 docker-compose.yaml（项目根目录）

使用场景：
  1. 完整的本地开发环境
  2. 快速验收测试
  3. 生产环境部署
  4. 一键启动所有服务

工作流示例：
  # 启动完整应用
  .\full_deploy.ps1

  # 查看日志
  docker logs -f <container-name>

相关脚本：
  - libs_only.ps1  仅启动依赖（不启动应用）

#>

param(
    [Parameter(Mandatory=$false)]
    [string]$AppRoot = "C:\app",

    [Parameter(Mandatory=$false)]
    [string]$ComposeFile = ""
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

function Find-ComposeFile {
    <#
    .SYNOPSIS
    查找 Compose 文件
    #>
    param([string]$RepoRoot)

    $possibleFiles = @(
        "docker-compose.yml",
        "docker-compose.yaml"
    )

    foreach ($file in $possibleFiles) {
        $filePath = Join-Path $RepoRoot $file
        if (Test-Path -Path $filePath -PathType Leaf) {
            return $file
        }
    }

    return $null
}

# ============================================================================
# 主程序开始
# ============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

Log "========================================"
Log "  Docker Compose - Full Deploy（完整）"
Log "========================================"
Log ""

# 获取项目根目录
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent $scriptDir

Log "Script dir: $scriptDir"
Log "Repo root: $repoRoot"
Log "App root: $AppRoot"

# 进入项目根目录
try {
    Push-Location $repoRoot
    Log "Changed to repo root: $repoRoot"
} catch {
    ErrorLog "Failed to change directory to: $repoRoot"
    exit 1
}

# 确定 Compose 文件
if ([string]::IsNullOrEmpty($ComposeFile)) {
    Log "Searching for Compose file..."
    $ComposeFile = Find-ComposeFile $repoRoot

    if ($null -eq $ComposeFile) {
        ErrorLog "No docker-compose.yml or docker-compose.yaml found in: $repoRoot"
        Pop-Location
        exit 1
    }

    Log "Found Compose file: $ComposeFile"
} else {
    if (-not (Test-Path -Path $ComposeFile -PathType Leaf)) {
        ErrorLog "Compose file not found: $ComposeFile"
        Pop-Location
        exit 1
    }
    Log "Using specified Compose file: $ComposeFile"
}

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
        Log "查看日志："
        Log "  docker logs -f <container-name>"
        Log ""
        Log "停止所有服务："
        Log "  docker-compose -f $ComposeFile down"
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

