<#
.SYNOPSIS
Windows 开发环境自动配置脚本（Scoop/Docker/Go）
.NOTES
保存编码：UTF-8 with BOM | 运行权限：普通/管理员均可 | 兼容：PowerShell 5.1+、所有 Scoop 版本
#>
param(
    [switch]$SkipDocker,
    [switch]$AutoConfirm
)

$previewBuckets = @('main','extras')
$previewPkgs = @('wget','unzip','git','jq','make','grep','gawk','sed','touch','mingw','nodejs','go')

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'  # 改为Continue，避免非致命错误直接退出

# 日志函数（纯英文避免编码问题）
function Log { param($m) Write-Host "==> $m" -ForegroundColor Cyan }
function Warn { param($m) Write-Host "$m" -ForegroundColor Yellow }
function ErrorLog { param($m) Write-Host "$m" -ForegroundColor Red }

# 错误处理：仅严重错误提示，不强制退出
function ErrTrap {
    param($Line, $Exception)
    ErrorLog "Error at line $Line : $($Exception.Message)"
    # 仅记录错误，不退出（避免整个脚本中断）
}
trap {
    # 过滤已知非致命错误
    if ($_.Exception.Message -match '不支持所指定的方法|not supported|permission denied|权限|Option -q not recognized') {
        Warn "Non-critical error: $($_.Exception.Message)"
        continue
    } elseif ($_.Exception.Message -notmatch 'exists|已存在|skip|跳过') {
        ErrTrap -Line $_.InvocationInfo.ScriptLineNumber -Exception $_.Exception
        continue
    } else {
        Warn $_.Exception.Message
        continue
    }
}

# 检测管理员权限
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$IsAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $IsAdmin) {
    Warn "Not running as administrator! Docker auto-start will be skipped."
}

# ========== Scoop 安装 ==========
if (-not (Get-Command scoop -ErrorAction SilentlyContinue)) {
    Log "Installing Scoop (non-interactive)"
    try {
        Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force -ErrorAction Stop
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-RestMethod -Uri https://get.scoop.sh -UseBasicParsing | Invoke-Expression
    } catch {
        ErrorLog "Scoop install failed: $($_.Exception.Message)"
    }
}

# 配置 Scoop Buckets
$neededBuckets = @('main','extras')
$bucketList = @(& scoop bucket list 2>$null)
foreach ($b in $neededBuckets) {
    if ($bucketList -notcontains $b) {
        Log "Adding Scoop bucket: $b"
        try { & scoop bucket add $b --no-update 2>$null }
        catch { Warn "Failed to add bucket $b : $($_.Exception.Message)" }
    } else {
        Log "Bucket $b already exists, skip"
    }
}

# 安装基础工具（移除 -q 参数，兼容旧版 Scoop）
$pkgs = @('wget','unzip','git','jq','make','grep','gawk','sed','touch','mingw','nodejs','go')
foreach ($p in $pkgs) {
    if (-not (Get-Command $p -ErrorAction SilentlyContinue)) {
        Log "Installing tool: $p"
        try { & scoop install $p 2>$null }  # 移除 -q 参数
        catch { Warn "Failed to install $p : $($_.Exception.Message)" }
    } else {
        Log "$p already installed, skip"
    }
}

# ========== Docker 安装 ==========
function Install-DockerDesktop {
    Log "Installing Docker Desktop"
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        try {
            & winget install --id Docker.DockerDesktop `
                -e --accept-package-agreements --accept-source-agreements `
                --silent --disable-interactivity 2>$null
            Log "Docker Desktop install submitted via Winget (wait for background completion)"
        } catch {
            Warn "Winget install Docker failed: $($_.Exception.Message)"
        }
    } else {
        Warn "Winget not found, try install Docker CLI via Scoop"
        try { & scoop install docker 2>$null }  # 移除 -q 参数
        catch { Warn "Scoop install Docker CLI failed: $($_.Exception.Message)" }
    }
}

# 执行 Docker 安装
if (-not $SkipDocker) {
    try { Install-DockerDesktop }
    catch { Warn "Docker install process error: $($_.Exception.Message)" }
} else {
    Log "Skipping Docker installation per -SkipDocker"
}

# 配置 Docker 服务（仅管理员）
if (-not $SkipDocker) {
    $dockerServiceName = $null
    if (Get-Service -Name com.docker.service -ErrorAction SilentlyContinue) {
        $dockerServiceName = "com.docker.service"
    } elseif (Get-Service -Name docker -ErrorAction SilentlyContinue) {
        $dockerServiceName = "docker"
    }

    if ($dockerServiceName -and $IsAdmin) {
        Log "Configuring Docker service: $dockerServiceName"
        try {
            Set-Service -Name $dockerServiceName -StartupType Automatic -ErrorAction Stop
            Start-Service -Name $dockerServiceName -ErrorAction Stop
            Log "Docker service set to auto-start and started successfully"
        } catch {
            Warn "Failed to operate Docker service: $($_.Exception.Message)"
        }
    } elseif ($dockerServiceName) {
        Warn "Non-admin: skip Docker service configuration (run as admin to auto-start)"
    } else {
        Warn "Docker service not found (Docker Desktop may not be installed)"
    }
} else {
    Log "Skipping Docker service configuration per -SkipDocker"
}

# ========== Go 环境配置（仅当前会话生效，避免写入错误） ==========
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Log "Installing Go environment"
    try { & scoop install go 2>$null }  # 移除 -q 参数
    catch { Warn "Scoop install Go failed: $($_.Exception.Message)" }
}

# 配置GOPATH（仅当前会话）
$gopath = Join-Path $env:USERPROFILE "go"
New-Item -Path $gopath -ItemType Directory -Force | Out-Null
if (-not (Test-Path env:GOPATH)) {
    $env:GOPATH = $gopath
    Log "Set GOPATH for current session: $gopath"
}
$goBinPath = Join-Path $gopath "bin"
if (-not ($env:PATH -like "*$goBinPath*")) {
    $env:PATH += ";$goBinPath"
    Log "Added GOPATH/bin to current session: $goBinPath"
}

# ========== 手动配置提示（避免自动写入Profile触发错误） ==========
Log "Environment setup completed (current session only)!"
Write-Host @"

==================== MANUAL CONFIG TIPS ====================
1. To make GOPATH permanent:
   Add these lines to your PowerShell Profile:
   `$env:GOPATH = "$gopath"
   `$env:PATH += ";$gopath\bin"

2. Check service status (admin):
   Get-Service com.docker.service
"@ -ForegroundColor Green
