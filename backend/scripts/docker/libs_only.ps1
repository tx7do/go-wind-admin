<#
.SYNOPSIS
Docker Compose startup script - dependencies-only version (no application) (Windows PowerShell edition)

.DESCRIPTION
Starts Docker Compose with dependency services only; the main application is not started

.PARAMETER AppRoot
Data volume root directory (default: C:\app)
Directory layout: APP_ROOT\postgres, APP_ROOT\redis, etc.

.PARAMETER ComposeFile
Compose file path (default: docker-compose.libs.yaml)
Points to the dependencies-only Compose config

.EXAMPLE
# Start dependency services (default config)
.\libs_only.ps1

# Custom data directory
.\libs_only.ps1 -AppRoot "D:\app"

# Custom Compose file
.\libs_only.ps1 -ComposeFile "compose-deps.yaml"

# Fully customized
.\libs_only.ps1 -AppRoot "D:\myapp" -ComposeFile "custom-compose.yaml"

.NOTES
Services started (dependencies only):
  - PostgreSQL database
  - Redis cache
  - SeaweedFS object storage (S3-compatible, service name kept as minio)
  - Jaeger distributed tracing

Services NOT started:
  - Main application service (expected to run locally)

Use cases:
  1. Local development: dependencies in Docker, application code locally
  2. Debugging: easier to debug application code
  3. Fast iteration: no need to restart the application container
  4. IDE development: run and debug directly from the IDE

Workflow example:
  # PowerShell 1: start dependencies
  .\libs_only.ps1

  # PowerShell 2: run the application code
  gow run admin

Related scripts:
  - full_deploy.ps1  Full application (includes the app service)

#>

param(
    [Parameter(Mandatory=$false)]
    [string]$AppRoot = "C:\app",

    [Parameter(Mandatory=$false)]
    [string]$ComposeFile = "docker-compose.libs.yaml"
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

    .DESCRIPTION
    Prefers the compose plugin in the Docker CLI (docker compose);
    falls back to the standalone docker-compose command if unavailable

    .RETURNS
    The usable docker compose command, or $null
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

# ============================================================================
# Main program
# ============================================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Continue'

Log "========================================"
Log "  Docker Compose - Libs Only"
Log "========================================"
Log ""

# Repo root: the script lives in backend/scripts/docker/, go up two levels to backend/
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)

Log "Script dir: $scriptDir"
Log "Repo root: $repoRoot"
Log "App root: $AppRoot"
Log "Compose file: $ComposeFile"
Log ""

# Change to the repo root
try {
    Push-Location $repoRoot
    Log "Changed to repo root: $repoRoot"
} catch {
    ErrorLog "Failed to change directory to: $repoRoot"
    exit 1
}

# Verify the Compose file exists
if (-not (Test-Path -Path $ComposeFile -PathType Leaf)) {
    ErrorLog "Compose file not found: $ComposeFile"
    Pop-Location
    exit 1
}
Log "Compose file found: $ComposeFile"
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
        Log "Next steps:"
        Log "1. Start the application in another PowerShell:"
        Log "   cd app"
        Log "   go run main.go"
        Log ""
        Log "2. Or open the project in your IDE and press F5 to debug"
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
