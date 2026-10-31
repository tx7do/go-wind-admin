<#
.SYNOPSIS
Docker Compose startup script - full application version (app + dependencies) (Windows PowerShell edition)

.DESCRIPTION
Starts the full Docker Compose application, including the main app service and all dependencies

.PARAMETER AppRoot
Data volume root directory (default: C:\app)
Directory layout: APP_ROOT\postgres, APP_ROOT\redis, etc.

.PARAMETER ComposeFile
Compose file path (default: docker-compose.yml)
Points to the full-application Compose config

.EXAMPLE
# Start the full application (default config)
.\full_deploy.ps1

# Custom data directory
.\full_deploy.ps1 -AppRoot "D:\app"

# Custom Compose file
.\full_deploy.ps1 -ComposeFile "docker-compose.yaml"

# Fully customized
.\full_deploy.ps1 -AppRoot "D:\myapp" -ComposeFile "custom-compose.yaml"

.NOTES
Services started (full):
  - Main application service (as defined in docker-compose.yml)
  - PostgreSQL database
  - Redis cache
  - SeaweedFS object storage (S3-compatible, service name kept as minio)
  - Jaeger distributed tracing

Compose file:
  - Uses docker-compose.yml or docker-compose.yaml (repo root)

Use cases:
  1. Full local development environment
  2. Quick acceptance testing
  3. Production deployment
  4. One-click startup of all services

Workflow example:
  # Start the full application
  .\full_deploy.ps1

  # View logs
  docker logs -f <container-name>

Related scripts:
  - libs_only.ps1  Dependencies only (no application)

#>

param(
    [Parameter(Mandatory=$false)]
    [string]$AppRoot = "C:\app",

    [Parameter(Mandatory=$false)]
    [string]$ComposeFile = ""
)

# ============================================================================
# Function definitions
# ============================================================================

function Log {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Warn {
    param([string]$Message)
    Write-Host "WARNING: $Message" -ForegroundColor Yellow
}

function ErrorLog {
    param([string]$Message)
    Write-Host "ERROR: $Message" -ForegroundColor Red
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
    Detect and resolve the Docker Compose command
    #>

    try {
        # Try the docker compose plugin (preferred)
        $version = docker compose version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Log "Found Docker Compose plugin: docker compose"
            return "docker"
        }
    } catch {
        # Fall through to the next method
    }

    try {
        # Try the standalone docker-compose command
        $version = docker-compose --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Log "Found docker-compose command: docker-compose"
            return "docker-compose"
        }
    } catch {
        # Continue
    }

    ErrorLog "Neither 'docker compose' plugin nor 'docker-compose' found"
    ErrorLog "Please install Docker Desktop or docker-compose"
    return $null
}

function Find-ComposeFile {
    <#
    .SYNOPSIS
    Locate the Compose file
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
# Main program
# ============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

Log "========================================"
Log "  Docker Compose - Full Deploy"
Log "========================================"
Log ""

# Repo root: the script lives in backend/scripts/docker/, go up two levels to backend/
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)

Log "Script dir: $scriptDir"
Log "Repo root: $repoRoot"
Log "App root: $AppRoot"

# Change to the repo root
try {
    Push-Location $repoRoot
    Log "Changed to repo root: $repoRoot"
} catch {
    ErrorLog "Failed to change directory to: $repoRoot"
    exit 1
}

# Resolve the Compose file
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

# Create data volume directories
Log "Creating data directories..."
$dependencies = @('postgres', 'redis', 'etcd', 'minio', 'jaeger')

foreach ($dep in $dependencies) {
    $targetPath = Join-Path $AppRoot $dep
    EnsureDirectoryExists $targetPath
}

Log ""

# Resolve the Docker Compose command
Log "Checking Docker Compose availability..."
$dockerComposeCmd = Get-DockerComposeCommand

if ($null -eq $dockerComposeCmd) {
    Pop-Location
    exit 1
}

Log ""

# Build the command line
if ($dockerComposeCmd -eq "docker") {
    $command = "docker compose -f $ComposeFile up -d --force-recreate"
} else {
    $command = "$dockerComposeCmd -f $ComposeFile up -d --force-recreate"
}

Log "Executing: $command"
Log ""

try {
    # Run the Docker Compose command
    Invoke-Expression $command

    if ($LASTEXITCODE -eq 0) {
        Log ""
        Log "========================================"
        Log "  Started successfully!"
        Log "========================================"
        Log ""
        Log "Running services:"

        # Show running containers
        if ($dockerComposeCmd -eq "docker") {
            docker compose -f $ComposeFile ps
        } else {
            docker-compose -f $ComposeFile ps
        }

        Log ""
        Log "View logs:"
        Log "  docker logs -f <container-name>"
        Log ""
        Log "Stop all services:"
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

# Return to the previous directory
Pop-Location

Log "========================================" -ForegroundColor Green
