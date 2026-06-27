#Requires -Version 5.1
<#
.SYNOPSIS
    Install Vidian on Windows.
.DESCRIPTION
    Downloads the latest Vidian release from GitHub and installs it to
    $env:LOCALAPPDATA\Programs\Vidian, then adds that directory to the
    current user's PATH if it isn't already there.
.PARAMETER Version
    Pin a specific release tag, e.g. "v1.2.0". Defaults to the latest release.
.EXAMPLE
    irm https://raw.githubusercontent.com/Ucok23/vidian/main/install.ps1 | iex
.EXAMPLE
    .\install.ps1 -Version v1.0.0
#>
param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$REPO        = "Ucok23/vidian"
$BINARY      = "vidian.exe"
$INSTALL_DIR = "$env:LOCALAPPDATA\Programs\Vidian"

function Write-Step  { Write-Host "-> $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host "v  $args" -ForegroundColor Green }
function Write-Err   { Write-Host "x  $args" -ForegroundColor Red; exit 1 }

Write-Host ""
Write-Host "  Vidian Installer" -ForegroundColor White
Write-Host "  Lightweight read-only code viewer"
Write-Host ""

# --- Detect architecture ---
$ARCH = if ([Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }
} else {
    Write-Err "32-bit Windows is not supported."
}

# --- Resolve version ---
if ($Version -eq "") {
    Write-Step "Fetching latest release..."
    try {
        $release = Invoke-RestMethod "https://api.github.com/repos/$REPO/releases/latest"
        $Version = $release.tag_name
    } catch {
        Write-Err "Could not fetch latest release: $_`nSet -Version to override, e.g.: .\install.ps1 -Version v1.0.0"
    }
}

Write-Step "Installing Vidian $Version (windows/$ARCH)..."

# --- Download ---
$ASSET       = "vidian_windows_$ARCH"
$DOWNLOAD_URL = "https://github.com/$REPO/releases/download/$Version/$ASSET.zip"
$TMP_DIR     = Join-Path ([System.IO.Path]::GetTempPath()) "vidian_install_$(Get-Random)"
New-Item -ItemType Directory -Path $TMP_DIR -Force | Out-Null

Write-Step "Downloading from: $DOWNLOAD_URL"
try {
    Invoke-WebRequest -Uri $DOWNLOAD_URL -OutFile "$TMP_DIR\$ASSET.zip" -UseBasicParsing
} catch {
    Write-Err "Download failed: $_`nCheck https://github.com/$REPO/releases for available versions."
}

Write-Step "Extracting..."
Expand-Archive -Path "$TMP_DIR\$ASSET.zip" -DestinationPath $TMP_DIR -Force

# --- Install ---
Write-Step "Installing to $INSTALL_DIR\$BINARY..."
New-Item -ItemType Directory -Path $INSTALL_DIR -Force | Out-Null
Copy-Item "$TMP_DIR\$BINARY" "$INSTALL_DIR\$BINARY" -Force

# --- Add to PATH ---
$USER_PATH = [Environment]::GetEnvironmentVariable("Path", "User")
if ($USER_PATH -notlike "*$INSTALL_DIR*") {
    [Environment]::SetEnvironmentVariable("Path", "$USER_PATH;$INSTALL_DIR", "User")
    Write-Ok "Added $INSTALL_DIR to your PATH."
    Write-Host "  Restart your terminal (or run: `$env:Path += ';$INSTALL_DIR'`) to use vidian now."
} else {
    Write-Ok "$INSTALL_DIR is already in your PATH."
}

# --- Cleanup ---
Remove-Item -Recurse -Force $TMP_DIR

Write-Ok "Vidian $Version installed successfully!"
Write-Host ""
Write-Host "  Usage:" -ForegroundColor White
Write-Host "    vidian .                    # open current directory"
Write-Host "    vidian C:\projects\my-app   # open a specific folder"
Write-Host ""
Write-Host "  Note: Windows SmartScreen may warn on first launch — choose 'More info -> Run anyway'."
Write-Host ""
