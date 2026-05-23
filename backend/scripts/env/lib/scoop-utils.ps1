<#
.SYNOPSIS
Scoop 包管理器工具函数库
.DESCRIPTION
提供 Scoop 安装、配置和包管理的通用函数
.NOTES
保存编码：UTF-8 with BOM | 兼容：PowerShell 5.1+
#>

# 导入通用工具库（如果尚未导入）
if (-not $global:CommonUtilsLoaded) {
    $LibDir = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
    . (Join-Path $LibDir "common-utils.ps1")
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

# Scoop 配置函数
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

# Scoop 初始化函数
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
