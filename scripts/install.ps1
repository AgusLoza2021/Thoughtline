#requires -Version 5.1
<#
.SYNOPSIS
    Auto-installer for Thoughtline on Windows.

.DESCRIPTION
    Installs the `thoughtline` MCP server and registers it with Claude Code
    (which the VS Code extension also reads). Idempotent — safe to re-run.

    Steps:
      1. Verify Windows + PowerShell 5.1+
      2. Ensure Go is installed (uses winget if missing)
      3. go install github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@<version>
      4. Add %USERPROFILE%\go\bin to user PATH (if missing)
      5. Register `thoughtline` as an MCP server in %USERPROFILE%\.claude.json

    No admin/UAC required. Everything is user-scoped.

.PARAMETER Version
    Module version to install. Defaults to "latest".

.PARAMETER SkipPathSetup
    Don't touch the user PATH variable.

.PARAMETER SkipMcp
    Don't register the MCP server in ~/.claude.json.

.PARAMETER Force
    Overwrite an existing `thoughtline` entry in ~/.claude.json.

.EXAMPLE
    # Run from PowerShell
    .\scripts\install.ps1

.EXAMPLE
    # Pin to a specific version
    .\scripts\install.ps1 -Version v0.0.1

.EXAMPLE
    # Remote one-liner
    irm https://raw.githubusercontent.com/AgusLoza2021/Thoughtline/main/scripts/install.ps1 | iex
#>
[CmdletBinding()]
param(
    [string]$Version = "latest",
    [switch]$SkipPathSetup,
    [switch]$SkipMcp,
    [switch]$Force
)

$ErrorActionPreference = 'Stop'

# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------
function Write-Step { param([string]$Msg) Write-Host "==> $Msg" -ForegroundColor Cyan }
function Write-Ok   { param([string]$Msg) Write-Host "  OK $Msg" -ForegroundColor Green }
function Write-Info { param([string]$Msg) Write-Host "    $Msg" -ForegroundColor Gray }
function Write-Warn { param([string]$Msg) Write-Host "  ! $Msg" -ForegroundColor Yellow }

# ---------------------------------------------------------------------------
# 1. Pre-flight
# ---------------------------------------------------------------------------
function Test-Prereqs {
    Write-Step "Pre-flight"

    # PS 5.1 implies Windows. PS 7+ has $IsWindows.
    if ($PSVersionTable.PSVersion.Major -ge 6 -and -not $IsWindows) {
        throw "This installer is Windows-only. Detected: $($PSVersionTable.OS)"
    }

    Write-Ok "PowerShell $($PSVersionTable.PSVersion) on Windows"
}

# ---------------------------------------------------------------------------
# 2. Ensure Go
# ---------------------------------------------------------------------------
function Install-GoIfMissing {
    Write-Step "Checking Go toolchain"

    if (Get-Command go -ErrorAction SilentlyContinue) {
        $goVersion = (& go version) 2>$null
        Write-Ok $goVersion
        return
    }

    Write-Warn "Go not found. Installing via winget..."

    if (-not (Get-Command winget -ErrorAction SilentlyContinue)) {
        throw "winget is not available. Install Go manually from https://go.dev/dl/ and re-run this script."
    }

    & winget install --id GoLang.Go -e --silent --accept-source-agreements --accept-package-agreements
    if ($LASTEXITCODE -ne 0) {
        throw "winget failed to install Go (exit $LASTEXITCODE). Try installing manually from https://go.dev/dl/."
    }

    # Refresh PATH for the current session so we can call `go` immediately.
    $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
    $userPath    = [Environment]::GetEnvironmentVariable('Path', 'User')
    $env:Path = "$machinePath;$userPath"

    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Go was installed but is not on PATH. Open a new terminal and re-run the script."
    }

    Write-Ok "Go installed: $((& go version) 2>$null)"
}

# ---------------------------------------------------------------------------
# 3. go install thoughtline
# ---------------------------------------------------------------------------
function Install-Thoughtline {
    Write-Step "Installing thoughtline ($Version)"

    $modulePath = "github.com/AgusLoza2021/Thoughtline/cmd/thoughtline@$Version"
    Write-Info "go install $modulePath"

    & go install $modulePath
    if ($LASTEXITCODE -ne 0) {
        throw "go install failed (exit $LASTEXITCODE). Check network access and module path."
    }

    Write-Ok "thoughtline binary built"
}

# ---------------------------------------------------------------------------
# 4. PATH setup
# ---------------------------------------------------------------------------
function Get-GoBinDir {
    # Resolution order: GOBIN, GOPATH/bin, %USERPROFILE%\go\bin
    $gobin = [Environment]::GetEnvironmentVariable('GOBIN', 'User')
    if ($gobin) { return $gobin }

    $gopath = [Environment]::GetEnvironmentVariable('GOPATH', 'User')
    if (-not $gopath) {
        # `go env GOPATH` is the source of truth; falls back to ~/go
        $gopath = (& go env GOPATH) 2>$null
        if (-not $gopath) {
            $gopath = Join-Path $env:USERPROFILE 'go'
        }
    }

    return Join-Path $gopath 'bin'
}

function Add-GoBinToPath {
    if ($SkipPathSetup) {
        Write-Warn "Skipping PATH setup (--SkipPathSetup)"
        return
    }

    Write-Step "Ensuring Go bin is on user PATH"

    $goBin = Get-GoBinDir
    Write-Info "Go bin dir: $goBin"

    if (-not (Test-Path $goBin)) {
        Write-Warn "$goBin does not exist yet (it will after first build)."
    }

    $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    $entries  = if ($userPath) { $userPath -split ';' | Where-Object { $_ } } else { @() }

    $normalizedTarget = $goBin.TrimEnd('\').ToLowerInvariant()
    $alreadyIn = $false
    foreach ($e in $entries) {
        if ($e.TrimEnd('\').ToLowerInvariant() -eq $normalizedTarget) {
            $alreadyIn = $true
            break
        }
    }

    if ($alreadyIn) {
        Write-Ok "PATH already contains $goBin"
    } else {
        $newPath = if ($userPath) { "$userPath;$goBin" } else { $goBin }
        [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
        Write-Ok "Added $goBin to user PATH"
        Write-Info "(New terminals will see it automatically.)"
    }

    # Make it visible in the current session too.
    if (-not ($env:Path -split ';' | Where-Object { $_.TrimEnd('\').ToLowerInvariant() -eq $normalizedTarget })) {
        $env:Path = "$env:Path;$goBin"
    }
}

# ---------------------------------------------------------------------------
# 5. Register MCP in ~/.claude.json
# ---------------------------------------------------------------------------
function Register-McpServer {
    if ($SkipMcp) {
        Write-Warn "Skipping MCP registration (--SkipMcp)"
        return
    }

    Write-Step "Registering MCP server in ~/.claude.json"

    $configPath = Join-Path $env:USERPROFILE '.claude.json'

    if (Test-Path $configPath) {
        $raw = Get-Content $configPath -Raw -Encoding UTF8
        if ([string]::IsNullOrWhiteSpace($raw)) {
            $config = [PSCustomObject]@{}
        } else {
            try {
                $config = $raw | ConvertFrom-Json
            } catch {
                throw "Could not parse $configPath as JSON: $($_.Exception.Message). Fix it manually or rename it and re-run."
            }
        }
    } else {
        $config = [PSCustomObject]@{}
    }

    # Ensure mcpServers object exists.
    if (-not ($config.PSObject.Properties.Name -contains 'mcpServers')) {
        $config | Add-Member -MemberType NoteProperty -Name 'mcpServers' -Value ([PSCustomObject]@{})
    }

    $existing = $config.mcpServers.PSObject.Properties['thoughtline']
    if ($existing -and -not $Force) {
        Write-Ok "'thoughtline' already registered in $configPath"
        Write-Info "(Use -Force to overwrite.)"
        return
    }

    $serverDef = [PSCustomObject]@{
        type    = 'stdio'
        command = 'thoughtline'
        args    = @('serve')
    }

    if ($existing) {
        $config.mcpServers.thoughtline = $serverDef
    } else {
        $config.mcpServers | Add-Member -MemberType NoteProperty -Name 'thoughtline' -Value $serverDef
    }

    $json = $config | ConvertTo-Json -Depth 100
    # Write UTF-8 without BOM (PS 5.1 default Out-File is UTF-16; we explicitly use UTF8).
    Set-Content -Path $configPath -Value $json -Encoding UTF8

    if ($existing -and $Force) {
        Write-Ok "Replaced existing 'thoughtline' entry in $configPath"
    } else {
        Write-Ok "Added 'thoughtline' to $configPath"
    }
}

# ---------------------------------------------------------------------------
# 6. Verify
# ---------------------------------------------------------------------------
function Confirm-Installation {
    Write-Step "Verifying"

    $goBin = Get-GoBinDir
    $exe   = Join-Path $goBin 'thoughtline.exe'

    if (-not (Test-Path $exe)) {
        throw "Binary not found at $exe. Something went wrong during go install."
    }

    $version = & $exe version 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "thoughtline binary exists but failed to run: $version"
    }

    Write-Ok $version
    Write-Info "Path: $exe"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
try {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host " Thoughtline -- Windows auto-installer"     -ForegroundColor Cyan
    Write-Host "==========================================" -ForegroundColor Cyan
    Write-Host ""

    Test-Prereqs
    Install-GoIfMissing
    Install-Thoughtline
    Add-GoBinToPath
    Register-McpServer
    Confirm-Installation

    Write-Host ""
    Write-Host "All done." -ForegroundColor Green
    Write-Host ""
    Write-Host "Next:" -ForegroundColor Green
    Write-Host "  - Restart Claude Code (CLI or VS Code extension) so it picks up the new MCP server."
    Write-Host "  - Try the TUI:  " -NoNewline
    Write-Host "thoughtline ui" -ForegroundColor Yellow
    Write-Host ""
} catch {
    Write-Host ""
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    Write-Host ""
    exit 1
}
