<#
.SYNOPSIS
Docker Desktop 工具函数库
.DESCRIPTION
提供 Docker 安装、配置和验证的通用函数，安装前自动检测避免重复
.NOTES
编码: UTF-8 (NO BOM) | 兼容: PowerShell 5.1+
#>

# 导入通用工具库（如果尚未导入）
if (-not $global:CommonUtilsLoaded) {
    $LibDir = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
    . (Join-Path $LibDir "common-utils.ps1")
}

# ========== 检测函数 ==========
function Test-DockerDesktopInstalled {
    <#
    .SYNOPSIS
    检测 Docker Desktop 是否已安装
    .DESCRIPTION
    通过多种方式检测：命令、注册表、文件路径、包管理器
    .OUTPUTS
    [bool] 已安装返回 $true，否则 $false
    #>
    
    # 1. 检查 docker 命令是否可用（最快）
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        try {
            $null = & docker --version 2>$null
            Log "  [DETECTED] Docker command available"
            return $true
        } catch {}
    }
    
    # 2. 检查注册表（Winget/官方安装器）
    $regPaths = @(
        "HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*",
        "HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*",
        "HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*"
    )
    
    foreach ($regPath in $regPaths) {
        $items = Get-ItemProperty $regPath -ErrorAction SilentlyContinue | 
            Where-Object { $_.DisplayName -like "*Docker Desktop*" -or $_.DisplayName -eq "Docker Desktop" }
        if ($items) {
            Log "  [DETECTED] Docker Desktop found in registry"
            return $true
        }
    }
    
    # 3. 检查安装目录
    $installPaths = @(
        "$env:ProgramFiles\Docker\Docker\Docker Desktop.exe",
        "${env:ProgramFiles(x86)}\Docker\Docker\Docker Desktop.exe",
        "$env:LOCALAPPDATA\Docker\Docker Desktop.exe"
    )
    
    foreach ($path in $installPaths) {
        if (Test-Path $path) {
            Log "  [DETECTED] Docker Desktop.exe found at: $path"
            return $true
        }
    }
    
    # 4. 检查 Scoop 是否已安装
    if (Get-Command scoop -ErrorAction SilentlyContinue) {
        $scoopList = & scoop list 2>$null
        if ($scoopList -match '\bdocker\b') {
            Log "  [DETECTED] Docker found in Scoop packages"
            return $true
        }
    }
    
    # 5. 检查 Winget 是否已安装
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        try {
            $wingetList = & winget list --id Docker.DockerDesktop --exact 2>$null
            if ($wingetList -match 'Docker Desktop') {
                Log "  [DETECTED] Docker Desktop found via Winget"
                return $true
            }
        } catch {}
    }
    
    return $false
}

# ========== 安装函数（增强版） ==========
function Install-DockerDesktop {
    <#
    .SYNOPSIS
    安装 Docker Desktop（先检测，避免重复）
    .DESCRIPTION
    1. 先检查是否已安装
    2. 已安装则跳过
    3. 未安装则优先 Winget，失败则尝试 Scoop
    #>
    
    # 🔍 先检测是否已安装
    Log "Checking if Docker Desktop is already installed..."
    if (Test-DockerDesktopInstalled) {
        SuccessLog "Docker Desktop is already installed, skip installation"
        return $true
    }
    
    Log "Docker Desktop not detected, starting installation..."

    # 🚀 尝试 Winget 安装
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        Log "  Using Winget to install Docker Desktop"
        try {
            # Winget 安装需要交互确认，添加 --silent 减少提示
            $wingetArgs = @(
                'install', '--id', 'Docker.DockerDesktop',
                '-e', '--accept-package-agreements', '--accept-source-agreements',
                '--silent', '--disable-interactivity'
            )
            & winget @wingetArgs 2>&1 | Out-Null
            
            # Winget 返回 0 表示成功，-1 表示需要重启/用户交互
            if ($LASTEXITCODE -eq 0 -or $LASTEXITCODE -eq -1) {
                SuccessLog "Docker Desktop install submitted via Winget"
                Log "  Note: Docker Desktop may require manual completion or system restart"
                return $true
            } else {
                Warn "  Winget install returned exit code: $LASTEXITCODE"
            }
        } catch {
            Warn "  [FAILED] Winget install Docker failed: $($_.Exception.Message)"
        }
    }

    # 🔁 Winget 失败/不可用，尝试 Scoop
    Log "  Winget not available or failed, trying Scoop..."
    if (Get-Command scoop -ErrorAction SilentlyContinue) {
        try {
            & scoop install docker 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                SuccessLog "Docker CLI installed via Scoop"
                Log "  Note: Scoop installs Docker CLI only, not Docker Desktop GUI"
                return $true
            } else {
                Warn "  Scoop install returned exit code: $LASTEXITCODE"
            }
        } catch {
            Warn "  [FAILED] Scoop install Docker failed: $($_.Exception.Message)"
        }
    }

    # ❌ 所有方式都失败
    ErrorLog "All installation methods failed. Please install Docker Desktop manually from: https://www.docker.com/products/docker-desktop"
    return $false
}

# ========== 配置服务函数（保持不变，略优化） ==========
function Configure-DockerService {
    param([bool]$IsAdmin = $false)

    if (-not $IsAdmin) {
        Warn "Docker service configuration requires administrator privileges (skipped)"
        return $false
    }

    Log "Configuring Docker service..."
    $dockerServiceName = $null
    
    # 优先查找 Docker Desktop 服务
    $services = @('com.docker.service', 'docker', 'dockerd')
    foreach ($svc in $services) {
        if (Get-Service -Name $svc -ErrorAction SilentlyContinue) {
            $dockerServiceName = $svc
            break
        }
    }

    if (-not $dockerServiceName) {
        Warn "  Docker service not found (Docker Desktop may not be fully installed yet)"
        return $false
    }

    Log "  Found Docker service: $dockerServiceName"
    try {
        Set-Service -Name $dockerServiceName -StartupType Automatic -ErrorAction Stop
        Start-Service -Name $dockerServiceName -ErrorAction Stop
        SuccessLog "Docker service configured and started"
        return $true
    } catch {
        Warn "  [FAILED] Failed to configure Docker service: $($_.Exception.Message)"
        return $false
    }
}

# ========== 验证函数（保持不变） ==========
function Verify-DockerInstallation {
    Log "Verifying Docker installation..."
    if (Get-Command docker -ErrorAction SilentlyContinue) {
        try {
            $dockerVersion = & docker --version 2>&1
            SuccessLog "Docker is available: $dockerVersion"
            return $true
        } catch {
            Warn "  [WARNING] Docker command found but failed to run: $($_.Exception.Message)"
            return $false
        }
    } else {
        Warn "  [NOT FOUND] Docker command not available"
        return $false
    }
}

# ========== 初始化函数（保持不变） ==========
function Initialize-Docker {
    param(
        [bool]$SkipDocker = $false,
        [bool]$IsAdmin = $false
    )

    if ($SkipDocker) {
        Log "Skipping Docker installation per -SkipDocker"
        return $true
    }

    Log "========== Initializing Docker =========="
    
    $installSuccess = Install-DockerDesktop
    $configSuccess = Configure-DockerService -IsAdmin $IsAdmin
    $verifySuccess = Verify-DockerInstallation

    if (-not $installSuccess -and -not $verifySuccess) {
        Warn "Docker setup incomplete. Please check installation manually."
    }
    
    return $true
}