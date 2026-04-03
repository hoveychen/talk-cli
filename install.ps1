# Works on Windows. Downloads the right binary. You're welcome.
#
# Usage:
#   powershell -c "irm https://raw.githubusercontent.com/hoveychen/speak-cli/main/install.ps1 | iex"
#
# Options (env vars):
#   $env:SPEAK_VERSION    = "v0.2.0"         install a specific version (default: latest)
#   $env:SPEAK_INSTALL_DIR = "C:\Tools"      install location (default: $env:LOCALAPPDATA\speak-cli\bin)

$ErrorActionPreference = "Stop"

$Repo       = "hoveychen/speak-cli"
$BinaryName = "speak.exe"
$Asset      = "speak-windows-amd64.exe"
$InstallDir = if ($env:SPEAK_INSTALL_DIR) { $env:SPEAK_INSTALL_DIR } `
              else { Join-Path $env:LOCALAPPDATA "speak-cli\bin" }
$Version    = if ($env:SPEAK_VERSION) { $env:SPEAK_VERSION } else { "" }

function Write-Info  { param($msg) Write-Host "  -> $msg" -ForegroundColor Cyan }
function Write-Ok    { param($msg) Write-Host "  v $msg"  -ForegroundColor Green }
function Write-Warn  { param($msg) Write-Host "  ! $msg"  -ForegroundColor Yellow }
function Write-Fail  { param($msg) Write-Host "  x $msg"  -ForegroundColor Red; exit 1 }

# ── detect arch ───────────────────────────────────────────────────────────────
$Arch = $env:PROCESSOR_ARCHITECTURE
if ($Arch -ne "AMD64") {
    Write-Fail "Unsupported architecture: $Arch. Only AMD64 is supported."
}

# ── resolve version ───────────────────────────────────────────────────────────
if (-not $Version) {
    Write-Info "Fetching latest release..."
    try {
        $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
        $Version = $Release.tag_name
    } catch {
        Write-Fail "Could not fetch latest version: $_"
    }
}

$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$Asset"

# ── download ──────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  Installing speak $Version (Windows/AMD64)" -ForegroundColor White
Write-Host ""

$TmpFile = Join-Path $env:TEMP "speak-install-$([System.IO.Path]::GetRandomFileName()).exe"

Write-Info "Downloading $Asset..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpFile -UseBasicParsing
} catch {
    Write-Fail "Download failed: $_"
}

# ── install ───────────────────────────────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$Dest = Join-Path $InstallDir $BinaryName
Move-Item -Force $TmpFile $Dest
Write-Ok "Installed to $Dest"

# ── PATH check ────────────────────────────────────────────────────────────────
$UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Warn "$InstallDir is not in your PATH."
    Write-Info "Adding it now for your user account..."
    [System.Environment]::SetEnvironmentVariable(
        "PATH", "$UserPath;$InstallDir", "User"
    )
    $env:PATH = "$env:PATH;$InstallDir"
    Write-Ok "PATH updated. Restart your terminal to apply."
}

# ── done ──────────────────────────────────────────────────────────────────────
Write-Host ""
Write-Host "  All done! Run: speak ""Hello, world!""" -ForegroundColor Green
Write-Host ""
