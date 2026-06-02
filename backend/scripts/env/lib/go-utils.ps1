<#
.SYNOPSIS
Go 环境配置工具函数库（增强检测版）
.DESCRIPTION
提供 Go 运行时安装、环境变量配置、插件和 CLI 工具安装的通用函数，支持智能跳过已安装项
.NOTES
编码: UTF-8 (NO BOM) | 兼容: PowerShell 5.1+
#>

# ========== 导入通用工具库 ==========
if (-not $global:CommonUtilsLoaded) {
    $LibDir = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
    . (Join-Path $LibDir "common-utils.ps1")
}

# ========== 检测函数 ==========
function Test-GoToolInstalled {
    <#
    .SYNOPSIS
    检测 Go 工具是否已安装
    .PARAMETER ToolName
    工具的可执行文件名（不含 .exe），如 'kratos', 'protoc-gen-go'
    .OUTPUTS
    [bool] 已安装返回 $true
    #>
    param([string]$ToolName)
    
    # 1. 优先检查 GOBIN
    if ($env:GOBIN -and (Test-Path (Join-Path $env:GOBIN "$ToolName.exe"))) {
        return $true
    }
    
    # 2. 检查 GOPATH/bin
    if ($env:GOPATH -and (Test-Path (Join-Path $env:GOPATH "bin\$ToolName.exe"))) {
        return $true
    }
    
    # 3. 检查 PATH 中是否有该命令
    if (Get-Command $ToolName -ErrorAction SilentlyContinue) {
        return $true
    }
    
    return $false
}

function Test-GoProxyConfigured {
    <#
    .SYNOPSIS
    检测 Go 代理配置是否符合预期
    .PARAMETER ExpectedProxy
    期望的 GOPROXY 值
    .OUTPUTS
    [bool] 配置匹配返回 $true
    #>
    param([string]$ExpectedProxy)
    
    try {
        $currentProxy = & go env GOPROXY 2>$null
        # 支持逗号分隔的多个代理，顺序不重要
        $expectedList = $ExpectedProxy -split ',' | ForEach-Object { $_.Trim() }
        $currentList = $currentProxy -split ',' | ForEach-Object { $_.Trim() }
        
        # 检查期望的代理是否都在当前配置中
        $allMatch = $true
        foreach ($exp in $expectedList) {
            if ($currentList -notcontains $exp) {
                $allMatch = $false
                break
            }
        }
        
        if ($allMatch) {
            Log "  [DETECTED] GOPROXY already configured: $currentProxy"
            return $true
        }
    } catch {}
    
    return $false
}

# ========== Go 运行时安装（带检测） ==========
function Install-GoRuntime {
    Log "Checking Go runtime..."
    
    # 🔍 检测是否已安装
    if (Get-Command go -ErrorAction SilentlyContinue) {
        try {
            $goVersion = & go version 2>&1
            SuccessLog "Go already installed: $goVersion"
            return $true
        } catch {}
    }
    
    Log "Go not found, installing via Scoop..."
    try {
        & scoop install go 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) {
            $goVersion = & go version 2>&1
            SuccessLog "Go installed: $goVersion"
            return $true
        } else {
            Warn "Go installation failed (exit code: $LASTEXITCODE)"
            return $false
        }
    } catch {
        ErrorLog "Failed to install Go: $($_.Exception.Message)"
        return $false
    }
}

# ========== 环境变量配置（修复返回值） ==========
function Set-GoEnvironment {
    param(
        [string]$GoPath = (Join-Path $env:USERPROFILE "go"),
        [string]$GoProxy = "https://goproxy.io,direct"
    )

    Log "Configuring Go environment..."
    
    # 创建 GOPATH
    if (-not (Test-Path $GoPath)) {
        Log "  Creating GOPATH: $GoPath"
        New-Item -Path $GoPath -ItemType Directory -Force | Out-Null
    }
    
    # 配置当前会话
    $env:GOPATH = $GoPath
    Log "  [OK] GOPATH = $GoPath (current session)"
    
    # 配置 GOBIN
    $goBinPath = Join-Path $GoPath "bin"
    if (-not (Test-Path $goBinPath)) {
        New-Item -Path $goBinPath -ItemType Directory -Force | Out-Null
    }
    $env:GOBIN = $goBinPath
    
    # 添加到 PATH（避免重复）
    if ($env:PATH -notlike "*$goBinPath*") {
        $env:PATH = "$goBinPath;$env:PATH"
        Log "  [OK] Added to PATH: $goBinPath"
    }
    
    # ✅ 返回哈希表，调用更清晰
    return @{
        GoPath = $GoPath
        GoBin = $goBinPath
    }
}

# ========== Go 代理配置（带检测） ==========
function Set-GoProxy {
    param(
        [string]$GoProxy = "https://goproxy.io,direct",
        [bool]$GoModuleOn = $true
    )

    Log "Configuring Go proxy..."
    
    # 🔍 先检测是否已配置
    if (Test-GoProxyConfigured -ExpectedProxy $GoProxy) {
        SuccessLog "GOPROXY already matches expected value, skip configuration"
        
        # 但仍检查 GO111MODULE
        if ($GoModuleOn) {
            $currentModule = & go env GO111MODULE 2>$null
            if ($currentModule -ne 'on') {
                Log "  Enabling GO111MODULE..."
                & go env -w GO111MODULE=on 2>&1 | Out-Null
            }
        }
        return $true
    }
    
    try {
        # 配置 GOPROXY
        Log "  Setting GOPROXY: $GoProxy"
        & go env -w GOPROXY=$GoProxy 2>&1 | Out-Null
        
        # 配置 GO111MODULE
        if ($GoModuleOn) {
            Log "  Enabling GO111MODULE..."
            & go env -w GO111MODULE=on 2>&1 | Out-Null
        }
        
        # 显示结果
        $envInfo = & go env GOPROXY, GO111MODULE 2>&1
        Log "  [OK] Current config: GOPROXY=$($envInfo[0]), GO111MODULE=$($envInfo[1])"
        return $true
    } catch {
        ErrorLog "Failed to configure Go proxy: $($_.Exception.Message)"
        return $false
    }
}

# ========== 批量安装 Go 包（智能跳过） ==========
function Install-GoPackages {
    param(
        [Parameter(Mandatory=$true)]
        [string[]]$Packages,
        [switch]$Force  # 强制重新安装
    )

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        ErrorLog "Go not found, cannot install packages"
        return
    }

    foreach ($pkg in $Packages) {
        # 提取工具名：google.golang.org/protobuf/cmd/protoc-gen-go@latest → protoc-gen-go
        $toolName = ($pkg -split '/' | Select-Object -Last 1) -replace '@.*$', ''
        
        # 🔍 检测是否已安装（除非强制）
        if (-not $Force -and (Test-GoToolInstalled -ToolName $toolName)) {
            Log "  [SKIP] $toolName already installed"
            continue
        }
        
        Log "  Installing: $pkg"
        try {
            & go install $pkg 2>&1 | Out-Null
            if ($LASTEXITCODE -eq 0) {
                SuccessLog "Installed: $toolName"
            } else {
                Warn "  [FAILED] $pkg (exit code: $LASTEXITCODE)"
            }
        } catch {
            ErrorLog "  [ERROR] Failed to install $pkg : $($_.Exception.Message)"
        }
    }
}

# ========== 插件安装（保持不变，调用上述函数） ==========
function Install-GoPlugins {
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

# ========== CLI 工具安装（保持不变） ==========
function Install-GoCliTools {
    Log "Installing CLI scaffold tools..."
    
    $cliTools = @(
        'github.com/go-kratos/kratos/cmd/kratos/v2@latest',
        'github.com/google/gnostic@latest',
        'github.com/bufbuild/buf/cmd/buf@latest',
        'entgo.io/ent/cmd/ent@latest',
        'github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest',
        'github.com/tx7do/go-wind-toolkit/gowind/cmd/gow@latest'
    )
    
    if ($cliTools.Count -gt 0) {
        Install-GoPackages -Packages $cliTools
    } else {
        Log "No CLI tools to install"
    }
}

# ========== 初始化函数（修复函数名调用） ==========
function Initialize-GoEnvironment {
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
    
    # 2. 配置环境变量 ✅ 修复：函数名改为 Set-GoEnvironment
    Log ""
    $envConfig = Set-GoEnvironment -GoPath $GoPath
    $GoPath = $envConfig.GoPath  # 使用哈希表取值，更清晰
    
    # 3. 配置代理 ✅ 修复：函数名改为 Set-GoProxy
    Log ""
    $proxySuccess = Set-GoProxy -GoProxy $GoProxy -GoModuleOn $true
    if (-not $proxySuccess) {
        Warn "Go proxy configuration failed"
    }
    
    # 4. 安装插件
    if (-not $SkipPlugins) {
        Log ""
        Install-GoPlugins
    } else {
        Log "Skipping Go plugins installation per -SkipPlugins"
    }
    
    # 5. 安装 CLI 工具
    if (-not $SkipCliTools) {
        Log ""
        Install-GoCliTools
    } else {
        Log "Skipping CLI tools installation per -SkipCliTools"
    }
    
    Log ""
    SuccessLog "Go Environment Setup Completed"
    return $true
}
