<#
.SYNOPSIS
    vStats CLI Installer for Windows
    
.DESCRIPTION
    Downloads and installs the vStats CLI on Windows.
    Supports multiple installation methods: Scoop, Chocolatey, or direct binary.
    
.PARAMETER Method
    Installation method: auto, scoop, choco, binary (default: auto)
    
.PARAMETER Version
    Specific version to install (default: latest)
    
.PARAMETER InstallDir
    Custom installation directory (for binary method)
    
.PARAMETER Uninstall
    Uninstall vStats CLI
    
.PARAMETER Upgrade
    Upgrade to latest version
    
.EXAMPLE
    # One-line install (PowerShell)
    irm https://vstats.zsoft.cc/cli.ps1 | iex
    
.EXAMPLE
    # Install with specific method
    . { irm https://vstats.zsoft.cc/cli.ps1 } | iex; Install-VStatsCLI -Method scoop
    
.EXAMPLE
    # Install specific version
    . { irm https://vstats.zsoft.cc/cli.ps1 } | iex; Install-VStatsCLI -Version "1.0.0"
#>

param(
    [ValidateSet("auto", "scoop", "choco", "binary")]
    [string]$Method = "auto",
    [string]$Version,
    [string]$InstallDir,
    [switch]$Uninstall,
    [switch]$Upgrade,
    [switch]$Help
)

# Configuration
# Allow VSTATS_API to be overridden via environment variable (optional for CLI)
if (-not $env:VSTATS_API) {
    $env:VSTATS_API = "https://vstats.zsoft.cc"
}
$VSTATS_API = $env:VSTATS_API
$VSTATS_DOWNLOAD = "$VSTATS_API/download"
$GITHUB_REPO = "zsai001/vstats"
$GITHUB_API = "https://api.github.com/repos/$GITHUB_REPO/releases/latest"
$GITHUB_DOWNLOAD = "https://github.com/$GITHUB_REPO/releases/download"
$DEFAULT_INSTALL_DIR = "$env:LOCALAPPDATA\vstats"

# Colors and output helpers
function Write-Banner {
    Write-Host ""
    Write-Host "+=================================================+" -ForegroundColor Cyan
    Write-Host "|        vStats CLI - Installation Script         |" -ForegroundColor Cyan
    Write-Host "+=================================================+" -ForegroundColor Cyan
    Write-Host ""
}

function Write-Info($msg) { Write-Host "[INFO] " -ForegroundColor Blue -NoNewline; Write-Host $msg }
function Write-Success($msg) { Write-Host "[OK] " -ForegroundColor Green -NoNewline; Write-Host $msg }
function Write-Warn($msg) { Write-Host "[WARN] " -ForegroundColor Yellow -NoNewline; Write-Host $msg }
function Write-Error($msg) { Write-Host "[ERROR] " -ForegroundColor Red -NoNewline; Write-Host $msg }

function Show-Help {
    Write-Host @"
vStats CLI Installation Script for Windows

Usage: irm https://vstats.zsoft.cc/cli.ps1 | iex

Or download and run with options:
    .\cli.ps1 [-Method <method>] [-Version <version>] [-InstallDir <path>]

Options:
    -Method      Installation method: auto, scoop, choco, binary (default: auto)
    -Version     Specific version to install (default: latest)
    -InstallDir  Custom installation directory (binary method only)
    -Uninstall   Uninstall vStats CLI
    -Upgrade     Upgrade to latest version
    -Help        Show this help

Methods:
    auto    - Automatically detect best method (scoop > choco > binary)
    scoop   - Install via Scoop package manager
    choco   - Install via Chocolatey package manager
    binary  - Direct binary download

Examples:
    # One-line install
    irm https://vstats.zsoft.cc/cli.ps1 | iex

    # Install via Scoop
    . { irm https://vstats.zsoft.cc/cli.ps1 } | iex; Install-VStatsCLI -Method scoop

    # Install specific version
    . { irm https://vstats.zsoft.cc/cli.ps1 } | iex; Install-VStatsCLI -Version "1.0.0"
"@
}

function Get-Architecture {
    $arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        "x86"   { return "386" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    Write-Info "Fetching latest version..."
    
    try {
        $release = Invoke-RestMethod -Uri $GITHUB_API -ErrorAction Stop
        $version = $release.tag_name -replace '^v', ''
        Write-Success "Latest version: $version"
        return $version
    } catch {
        Write-Warn "Could not fetch latest version from GitHub"
        throw "Failed to get latest version: $_"
    }
}

function Test-ScoopInstalled {
    return (Get-Command scoop -ErrorAction SilentlyContinue) -ne $null
}

function Test-ChocolateyInstalled {
    return (Get-Command choco -ErrorAction SilentlyContinue) -ne $null
}

function Install-ViaScoop {
    Write-Info "Installing vStats CLI via Scoop..."
    
    if (-not (Test-ScoopInstalled)) {
        Write-Info "Scoop not found. Installing Scoop first..."
        Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
        Invoke-RestMethod -Uri get.scoop.sh | Invoke-Expression
    }
    
    # Add vstats bucket
    Write-Info "Adding vstats bucket..."
    scoop bucket add vstats https://github.com/zsai001/scoop-vstats 2>$null
    
    # Install
    Write-Info "Installing vstats..."
    scoop install vstats
    
    Write-Success "vStats CLI installed via Scoop"
}

function Install-ViaChocolatey {
    Write-Info "Installing vStats CLI via Chocolatey..."
    
    if (-not (Test-ChocolateyInstalled)) {
        Write-Error "Chocolatey not found. Please install Chocolatey first or use another method."
        Write-Host "Install Chocolatey: https://chocolatey.org/install"
        throw "Chocolatey not installed"
    }
    
    # Check for admin rights
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    if (-not $isAdmin) {
        Write-Error "Chocolatey requires administrator privileges."
        Write-Host "Please run PowerShell as Administrator."
        throw "Admin required for Chocolatey"
    }
    
    # Install
    Write-Info "Installing vstats..."
    choco install vstats -y
    
    Write-Success "vStats CLI installed via Chocolatey"
}

function Install-ViaBinary {
    param([string]$TargetVersion, [string]$TargetDir)
    
    Write-Info "Installing vStats CLI binary..."
    
    $arch = Get-Architecture
    $installDir = if ($TargetDir) { $TargetDir } else { $DEFAULT_INSTALL_DIR }
    $version = if ($TargetVersion) { $TargetVersion } else { Get-LatestVersion }
    
    # Create install directory
    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }
    
    # Construct download URL
    $binaryName = "vstats-cli-windows-$arch.exe"
    $downloadUrl = "https://github.com/$GITHUB_REPO/releases/download/v$version/$binaryName"
    $targetPath = Join-Path $installDir "vstats.exe"
    
    Write-Info "Version: $version"
    Write-Info "Architecture: $arch"
    Write-Info "Downloading from: $downloadUrl"
    
    # Download
    try {
        $ProgressPreference = 'SilentlyContinue'
        Invoke-WebRequest -Uri $downloadUrl -OutFile $targetPath -ErrorAction Stop
        $ProgressPreference = 'Continue'
    } catch {
        Write-Error "Failed to download binary"
        throw $_
    }
    
    Write-Success "Binary installed to: $targetPath"
    
    # Add to PATH
    $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($userPath -notlike "*$installDir*") {
        Write-Info "Adding $installDir to PATH..."
        [Environment]::SetEnvironmentVariable(
            "PATH",
            "$userPath;$installDir",
            "User"
        )
        $env:PATH = "$env:PATH;$installDir"
        Write-Success "Added to PATH (restart terminal to apply globally)"
    }
    
    Write-Success "vStats CLI installed via binary"
}

function Uninstall-VStatsCLI {
    Write-Info "Uninstalling vStats CLI..."
    
    # Try Scoop
    if (Test-ScoopInstalled) {
        $scoopApps = scoop list 2>$null | Select-String "vstats"
        if ($scoopApps) {
            Write-Info "Removing via Scoop..."
            scoop uninstall vstats
            Write-Success "Uninstalled via Scoop"
            return
        }
    }
    
    # Try Chocolatey
    if (Test-ChocolateyInstalled) {
        $chocoApps = choco list --local-only 2>$null | Select-String "vstats"
        if ($chocoApps) {
            Write-Info "Removing via Chocolatey..."
            choco uninstall vstats -y
            Write-Success "Uninstalled via Chocolatey"
            return
        }
    }
    
    # Remove binary
    $installDir = $DEFAULT_INSTALL_DIR
    if (Test-Path $installDir) {
        Write-Info "Removing binary installation..."
        Remove-Item -Path $installDir -Recurse -Force
        
        # Remove from PATH
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -like "*$installDir*") {
            $newPath = ($userPath -split ';' | Where-Object { $_ -ne $installDir }) -join ';'
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        }
        
        Write-Success "Uninstalled binary version"
    } else {
        Write-Warn "vStats CLI not found"
    }
}

function Upgrade-VStatsCLI {
    Write-Info "Upgrading vStats CLI..."
    
    # Try Scoop
    if (Test-ScoopInstalled) {
        $scoopApps = scoop list 2>$null | Select-String "vstats"
        if ($scoopApps) {
            Write-Info "Upgrading via Scoop..."
            scoop update vstats
            Write-Success "Upgraded via Scoop"
            return
        }
    }
    
    # Try Chocolatey
    if (Test-ChocolateyInstalled) {
        $chocoApps = choco list --local-only 2>$null | Select-String "vstats"
        if ($chocoApps) {
            Write-Info "Upgrading via Chocolatey..."
            choco upgrade vstats -y
            Write-Success "Upgraded via Chocolatey"
            return
        }
    }
    
    # Upgrade binary
    Write-Info "Upgrading binary installation..."
    Install-ViaBinary
}

function Show-Completion {
    Write-Host ""
    Write-Host "+=================================================+" -ForegroundColor Green
    Write-Host "|      vStats CLI Installation Complete!          |" -ForegroundColor Green
    Write-Host "+=================================================+" -ForegroundColor Green
    Write-Host ""
    
    # Show version
    try {
        $versionOutput = & vstats version 2>&1
        Write-Host "  Version: $versionOutput" -ForegroundColor Cyan
    } catch {
        Write-Host "  Version: (restart terminal to verify)" -ForegroundColor Yellow
    }
    
    Write-Host ""
    Write-Host "  Quick Start:" -ForegroundColor Cyan
    Write-Host "    vstats login              # Login to vStats Cloud"
    Write-Host "    vstats server list        # List your servers"
    Write-Host "    vstats server create web1 # Create a new server"
    Write-Host ""
    Write-Host "  Documentation: https://vstats.zsoft.cc/docs/cli"
    Write-Host ""
}

# Exported function for pipeline install
function Install-VStatsCLI {
    param(
        [ValidateSet("auto", "scoop", "choco", "binary")]
        [string]$Method = "auto",
        [string]$Version,
        [string]$InstallDir
    )
    
    Write-Banner
    
    # Detect OS
    Write-Info "Detected: Windows/$((Get-Architecture))"
    
    # Determine installation method
    $selectedMethod = $Method
    if ($Method -eq "auto") {
        if (Test-ScoopInstalled) {
            $selectedMethod = "scoop"
        } elseif (Test-ChocolateyInstalled) {
            $selectedMethod = "choco"
        } else {
            $selectedMethod = "binary"
        }
        Write-Info "Auto-selected method: $selectedMethod"
    }
    
    Write-Host ""
    
    # Install
    switch ($selectedMethod) {
        "scoop" { Install-ViaScoop }
        "choco" { Install-ViaChocolatey }
        "binary" { Install-ViaBinary -TargetVersion $Version -TargetDir $InstallDir }
    }
    
    Show-Completion
}

# Main execution
function Main {
    Write-Banner
    
    if ($Help) {
        Show-Help
        return
    }
    
    if ($Uninstall) {
        Uninstall-VStatsCLI
        return
    }
    
    if ($Upgrade) {
        Upgrade-VStatsCLI
        return
    }
    
    Install-VStatsCLI -Method $Method -Version $Version -InstallDir $InstallDir
}

# Handle pipeline execution vs script execution
if ($MyInvocation.InvocationName -eq '.' -or $MyInvocation.InvocationName -eq '') {
    # Dot-sourced or pipeline - just load functions
    # User will call Install-VStatsCLI manually
    return
}

# Direct execution
Main
