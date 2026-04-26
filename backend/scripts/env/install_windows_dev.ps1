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

# ========== Scoop 安装函数 ==========
function Install-Scoop {
    <#
    .SYNOPSIS
    安装 Scoop 包管理器
    .DESCRIPTION
    安装 Scoop 并配置基本设置
    #>
    Log "Installing Scoop..."
    try {
        Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force -ErrorAction SilentlyContinue
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
        Invoke-RestMethod -Uri https://get.scoop.sh -UseBasicParsing | Invoke-Expression
        Log "Scoop installed successfully"
    } catch {
        ErrorLog "Scoop install failed: $($_.Exception.Message)"
        return $false
    }
    return $true
}

function Add-ScoopBuckets {
    <#
    .SYNOPSIS
    添加 Scoop Buckets
    .PARAMETER Buckets
    要添加的 Bucket 列表（数组）
    #>
    param(
        [string[]]$Buckets = @('main', 'extras')
    )

    Log "Configuring Scoop Buckets..."
    $bucketNames = @(& scoop bucket list 2>$null | ForEach-Object {
        if ($_ -is [string]) { $_.Trim() } else { $_.Name }
    })

    foreach ($bucket in $Buckets) {
        if ($bucketNames -notcontains $bucket) {
            Log "  Adding bucket: $bucket"
            try {
                & scoop bucket add $bucket --no-update 2>$null
                Log "    [OK] Bucket added: $bucket"
            } catch {
                Warn "    [FAILED] Failed to add bucket $bucket : $($_.Exception.Message)"
            }
        } else {
            Log "  [SKIP] Bucket already exists: $bucket"
        }
    }
}

function Install-ScoopPackages {
    <#
    .SYNOPSIS
    通过 Scoop 安装包
    .PARAMETER Packages
    要安装的包列表（数组）
    #>
    param(
        [Parameter(Mandatory=$true)]
        [string[]]$Packages
    )

    Log "Installing Scoop packages..."
    foreach ($pkg in $Packages) {
        # 检查包是否已安装
        if (Get-Command $pkg -ErrorAction SilentlyContinue) {
            Log "  [SKIP] $pkg already installed"
            continue
        }

        Log "  Installing: $pkg"
        try {
            & scoop install $pkg 2>$null
            if ($LASTEXITCODE -eq 0) {
                Log "    [OK] Successfully installed: $pkg"
            } else {
                Warn "    [FAILED] Installation failed: $pkg (exit code: $LASTEXITCODE)"
            }
        } catch {
            Warn "    [ERROR] Failed to install $pkg : $($_.Exception.Message)"
        }
    }
}

function Initialize-Scoop {
    <#
    .SYNOPSIS
    初始化 Scoop（安装 Scoop 和基础包）
    .PARAMETER Buckets
    要添加的 Bucket 列表
    .PARAMETER Packages
    要安装的包列表
    #>
    param(
        [string[]]$Buckets = @('main', 'extras'),
        [string[]]$Packages = @('wget', 'unzip', 'git', 'jq', 'make', 'grep', 'gawk', 'sed', 'touch', 'mingw', 'nodejs', 'go')
    )

    # 1. 检查并安装 Scoop
    if (-not (Get-Command scoop -ErrorAction SilentlyContinue)) {
        if (-not (Install-Scoop)) {
            return $false
        }
    } else {
        Log "Scoop already installed, skip installation"
    }

    # 2. 配置 Buckets
    Add-ScoopBuckets -Buckets $Buckets

    # 3. 安装包
    Install-ScoopPackages -Packages $Packages

    return $true
}

# ========== Scoop 安装
Initialize-Scoop -Buckets @('main', 'extras') -Packages @('wget', 'unzip', 'git', 'jq', 'make', 'grep', 'gawk', 'sed', 'touch', 'mingw', 'nodejs', 'go')

# ========== Docker 安装函数 ==========
function Install-DockerDesktop {
    <#
    .SYNOPSIS
    安装 Docker Desktop
    .DESCRIPTION
    优先使用 Winget 安装，失败则尝试 Scoop
    #>
    Log "Installing Docker Desktop..."

    if (Get-Command winget -ErrorAction SilentlyContinue) {
        Log "  Using Winget to install Docker Desktop"
        try {
            & winget install --id Docker.DockerDesktop `
                -e --accept-package-agreements --accept-source-agreements `
                --silent --disable-interactivity 2>$null
            Log "    [OK] Docker Desktop install submitted via Winget (wait for background completion)"
            return $true
        } catch {
            Warn "    [FAILED] Winget install Docker failed: $($_.Exception.Message)"
        }
    }

    # Winget 失败，尝试 Scoop
    Log "  Winget not available, trying Scoop..."
    try {
        & scoop install docker 2>$null
        Log "    [OK] Docker CLI installed via Scoop"
        return $true
    } catch {
        ErrorLog "    [ERROR] Failed to install Docker via Scoop: $($_.Exception.Message)"
        return $false
    }
}

function Configure-DockerService {
    <#
    .SYNOPSIS
    配置 Docker 服务（自动启动）
    .DESCRIPTION
    设置 Docker 服务为自动启动并立即启动
    需要管理员权限
    #>
    if (-not $IsAdmin) {
        Warn "Docker service configuration requires administrator privileges (skipped)"
        return $false
    }

    Log "Configuring Docker service..."

    # 查找 Docker 服务
    $dockerServiceName = $null
    if (Get-Service -Name com.docker.service -ErrorAction SilentlyContinue) {
        $dockerServiceName = "com.docker.service"
    } elseif (Get-Service -Name docker -ErrorAction SilentlyContinue) {
        $dockerServiceName = "docker"
    }

    if (-not $dockerServiceName) {
        Warn "  Docker service not found (Docker Desktop may not be installed yet)"
        return $false
    }

    Log "  Found Docker service: $dockerServiceName"

    try {
        # 设置为自动启动
        Log "  Setting startup type to Automatic..."
        Set-Service -Name $dockerServiceName -StartupType Automatic -ErrorAction Stop
        Log "    [OK] Startup type set to Automatic"

        # 启动服务
        Log "  Starting Docker service..."
        Start-Service -Name $dockerServiceName -ErrorAction Stop
        Log "    [OK] Docker service started successfully"

        return $true
    } catch {
        Warn "  [FAILED] Failed to configure Docker service: $($_.Exception.Message)"
        return $false
    }
}

function Verify-DockerInstallation {
    <#
    .SYNOPSIS
    验证 Docker 安装
    .DESCRIPTION
    检查 Docker 是否可用，并显示版本信息
    #>
    Log "Verifying Docker installation..."

    if (Get-Command docker -ErrorAction SilentlyContinue) {
        try {
            $dockerVersion = & docker --version 2>&1
            Log "  [OK] Docker is available: $dockerVersion"
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

function Initialize-Docker {
    <#
    .SYNOPSIS
    初始化 Docker（安装 + 配置 + 验证）
    .PARAMETER SkipDocker
    是否跳过 Docker 安装
    #>
    param(
        [bool]$SkipDocker = $false
    )

    if ($SkipDocker) {
        Log "Skipping Docker installation per -SkipDocker"
        return $true
    }

    Log "========== Initializing Docker =========="

    # 1. 安装 Docker
    $installSuccess = Install-DockerDesktop

    if (-not $installSuccess) {
        Warn "Docker installation failed or was skipped"
    }

    # 2. 配置 Docker 服务
    $configSuccess = Configure-DockerService

    if (-not $configSuccess) {
        Log "Docker service configuration skipped (may require admin privileges or Docker not installed yet)"
    }

    # 3. 验证 Docker
    $verifySuccess = Verify-DockerInstallation

    if (-not $verifySuccess) {
        Warn "Docker verification failed (Docker Desktop may still be installing in background)"
    }

    return $true
}

# ========== Docker 安装
Initialize-Docker -SkipDocker $SkipDocker

# ========== Go 环境配置函数 ==========
function Install-GoRuntime {
    <#
    .SYNOPSIS
    安装 Go 运行时
    .DESCRIPTION
    通过 Scoop 安装 Go，如果已安装则跳过
    #>
    Log "Installing Go runtime..."

    if (Get-Command go -ErrorAction SilentlyContinue) {
        $goVersion = & go version 2>&1
        Log "  [SKIP] Go already installed: $goVersion"
        return $true
    }

    try {
        Log "  Installing Go via Scoop..."
        & scoop install go 2>$null
        if ($LASTEXITCODE -eq 0) {
            $goVersion = & go version 2>&1
            Log "    [OK] Go installed successfully: $goVersion"
            return $true
        } else {
            Warn "    [FAILED] Go installation failed"
            return $false
        }
    } catch {
        Warn "  [ERROR] Failed to install Go: $($_.Exception.Message)"
        return $false
    }
}

function Configure-GoEnvironment {
    <#
    .SYNOPSIS
    配置 Go 环境变量
    .PARAMETER GoPath
    GOPATH 目录路径
    .PARAMETER GoProxy
    Go 代理地址
    #>
    param(
        [string]$GoPath = (Join-Path $env:USERPROFILE "go"),
        [string]$GoProxy = "https://goproxy.io,direct"
    )

    Log "Configuring Go environment variables..."

    # 创建 GOPATH 目录
    if (-not (Test-Path $GoPath)) {
        Log "  Creating GOPATH directory: $GoPath"
        New-Item -Path $GoPath -ItemType Directory -Force | Out-Null
        Log "    [OK] GOPATH directory created"
    } else {
        Log "  [SKIP] GOPATH directory already exists: $GoPath"
    }

    # 配置当前会话的 GOPATH
    Log "  Setting GOPATH for current session..."
    $env:GOPATH = $GoPath
    Log "    [OK] GOPATH set to: $GoPath"

    # 配置 GOBIN
    $goBinPath = Join-Path $GoPath "bin"
    if (-not (Test-Path $goBinPath)) {
        New-Item -Path $goBinPath -ItemType Directory -Force | Out-Null
    }

    # 添加 GOBIN 到 PATH
    if (-not ($env:PATH -like "*$goBinPath*")) {
        Log "  Adding GOBIN to PATH: $goBinPath"
        $env:PATH += ";$goBinPath"
        Log "    [OK] GOBIN added to PATH"
    } else {
        Log "  [SKIP] GOBIN already in PATH"
    }

    return $GoPath, $goBinPath
}

function Configure-GoProxy {
    <#
    .SYNOPSIS
    配置 Go 代理和 GO111MODULE
    .PARAMETER GoProxy
    Go 代理地址
    .PARAMETER GoModuleOn
    是否启用 GO111MODULE（默认 $true）
    #>
    param(
        [string]$GoProxy = "https://goproxy.io,direct",
        [bool]$GoModuleOn = $true
    )

    Log "Configuring Go proxy and module..."

    try {
        # 配置 GOPROXY
        Log "  Setting GOPROXY: $GoProxy"
        & go env -w GOPROXY=$GoProxy 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            Log "    [OK] GOPROXY configured"
        } else {
            Warn "    [FAILED] Failed to set GOPROXY"
        }

        # 配置 GO111MODULE
        if ($GoModuleOn) {
            Log "  Enabling GO111MODULE..."
            & go env -w GO111MODULE=on 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Log "    [OK] GO111MODULE enabled"
            } else {
                Warn "    [FAILED] Failed to enable GO111MODULE"
            }
        } else {
            Log "  [SKIP] GO111MODULE disabled"
        }

        # 显示配置结果
        Log "  Current Go environment:"
        $goEnv = & go env GOPROXY 2>&1
        Log "    GOPROXY: $goEnv"
        $goModule = & go env GO111MODULE 2>&1
        Log "    GO111MODULE: $goModule"

        return $true
    } catch {
        Warn "Failed to configure Go proxy: $($_.Exception.Message)"
        return $false
    }
}

function Initialize-GoEnvironment {
    <#
    .SYNOPSIS
    初始化 Go 环境（安装 + 配置代理 + 安装插件和工具）
    .PARAMETER GoPath
    GOPATH 目录路径
    .PARAMETER GoProxy
    Go 代理地址
    .PARAMETER SkipPlugins
    是否跳过插件安装
    .PARAMETER SkipCliTools
    是否跳过 CLI 工具安装
    #>
    param(
        [string]$GoPath = (Join-Path $env:USERPROFILE "go"),
        [string]$GoProxy = "https://goproxy.io,direct",
        [bool]$SkipPlugins = $false,
        [bool]$SkipCliTools = $false
    )

    Log "========== Initializing Go Environment =========="

    # 1. 安装 Go 运行时
    $runtimeSuccess = Install-GoRuntime
    if (-not $runtimeSuccess) {
        Warn "Go runtime installation failed, skipping further setup"
        return $false
    }

    # 2. 配置 Go 环境变量
    Log ""
    $configPaths = Configure-GoEnvironment -GoPath $GoPath
    $GoPath = $configPaths[0]
    $goBinPath = $configPaths[1]

    # 3. 配置 Go 代理
    Log ""
    $proxySuccess = Configure-GoProxy -GoProxy $GoProxy -GoModuleOn $true
    if (-not $proxySuccess) {
        Warn "Go proxy configuration failed"
    }

    # 4. 安装 Protobuf 插件
    if (-not $SkipPlugins) {
        Log ""
        Install-GoPlugins
    } else {
        Log "Skipping Go plugins installation per -SkipPlugins"
    }

    # 5. 安装 CLI 脚手架工具
    if (-not $SkipCliTools) {
        Log ""
        Install-GoCliTools
    } else {
        Log "Skipping CLI tools installation per -SkipCliTools"
    }

    Log ""
    Log "========== Go Environment Setup Completed =========="

    return $true
}

# ========== Go 环境配置
function Install-GoPackages {
    <#
    .SYNOPSIS
    统一安装 Go 包的函数
    .PARAMETER Packages
    要安装的 Go 包列表（数组）
    #>
    param(
        [Parameter(Mandatory=$true)]
        [string[]]$Packages
    )

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        ErrorLog "Go not found, cannot install packages"
        return
    }

    foreach ($pkg in $Packages) {
        Log "Installing Go package: $pkg"
        try {
            & go install $pkg 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                Log "  [OK] Successfully installed: $pkg"
            } else {
                Warn "  [FAILED] Installation failed: $pkg"
            }
        } catch {
            Warn "  [ERROR] Failed to install $pkg : $($_.Exception.Message)"
        }
    }
}

function Install-GoPlugins {
    <#
    .SYNOPSIS
    安装 Protobuf 编译器插件
    #>
    Log "Installing Protobuf compiler plugins..."

    $plugins = @(
        'google.golang.org/protobuf/cmd/protoc-gen-go@latest',
        'google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest',
        'github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest',
        'github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@latest',
        'github.com/google/gnostic/cmd/protoc-gen-openapi@latest',
        'github.com/envoyproxy/protoc-gen-validate@latest',
        'github.com/menta2k/protoc-gen-redact/v3@latest',
        'github.com/go-kratos/protoc-gen-typescript-http@latest'
    )

    Install-GoPackages -Packages $plugins
}

function Install-GoCliTools {
    <#
    .SYNOPSIS
    安装 CLI 脚手架工具
    .DESCRIPTION
    在此函数中添加项目所需的 CLI 工具
    #>
    Log "Installing CLI scaffold tools..."

    # 示例：安装常用开发工具
    $cliTools = @(
        'github.com/go-kratos/kratos/cmd/kratos/v2@latest',
        'github.com/google/gnostic@latest',
        'github.com/bufbuild/buf/cmd/buf@latest',
        'entgo.io/ent/cmd/ent@latest',
        'github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest',
        'github.com/tx7do/go-wind-toolkit/config-exporter/cmd/cfgexp@latest',
        'github.com/tx7do/go-wind-toolkit/sql-orm/cmd/sql2orm@latest',
        'github.com/tx7do/go-wind-toolkit/sql-proto/cmd/sql2proto@latest',
        'github.com/tx7do/go-wind-toolkit/sql-kratos/cmd/sql2kratos@latest',
        'github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest'
    )

    if ($cliTools.Count -gt 0) {
        Install-GoPackages -Packages $cliTools
    } else {
        Log "No CLI tools to install (can be extended in Install-GoCliTools function)"
    }
}

# ========== Go 环境配置
Initialize-GoEnvironment -GoPath (Join-Path $env:USERPROFILE "go") -GoProxy "https://goproxy.io,direct" -SkipPlugins $false -SkipCliTools $false

# ========== 手动配置提示
$goPathValue = $env:GOPATH
if (-not $goPathValue) { $goPathValue = Join-Path $env:USERPROFILE "go" }
Log "Environment setup completed (current session only)!"
Write-Host @"

==================== MANUAL CONFIG TIPS ====================
1. To make GOPATH permanent:
   Add these lines to your PowerShell Profile:
   `$env:GOPATH = "$goPathValue"
   `$env:PATH += ";$goPathValue\bin"

2. Check service status (admin):
   Get-Service com.docker.service
"@ -ForegroundColor Green
