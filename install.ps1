# lamplight installer for Windows
# Usage: irm https://raw.githubusercontent.com/intransigent-iconoclast/lamplight-cli/main/install.ps1 | iex
#
# Options (set as env vars before running):
#   $env:LAMPLIGHT_VERSION     install a specific version (default: latest)
#   $env:LAMPLIGHT_INSTALL_DIR install to a specific directory (default: $env:LOCALAPPDATA\lamplight)
#
#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Repo   = "intransigent-iconoclast/lamplight-cli"
$Binary = "lamplight"

function Write-Info    { Write-Host "==> $args" -ForegroundColor Cyan }
function Write-Success { Write-Host "✓  $args" -ForegroundColor Green }
function Write-Warn    { Write-Host "warning: $args" -ForegroundColor Yellow }
function Write-Err     { Write-Host "error: $args" -ForegroundColor Red; exit 1 }

# --- resolve latest version ---
function Get-LatestVersion {
    $response = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    return $response.tag_name
}

# --- verify SHA256 checksum ---
function Test-Checksum {
    param($File, $ChecksumFile)

    $filename = Split-Path $File -Leaf
    $line = Get-Content $ChecksumFile | Where-Object { $_ -match $filename }
    if (-not $line) {
        Write-Warn "no checksum found for $filename — skipping verification"
        return
    }

    $expected = ($line -split '\s+')[0].ToLower()
    $actual   = (Get-FileHash $File -Algorithm SHA256).Hash.ToLower()

    if ($expected -ne $actual) {
        Write-Err "checksum mismatch for ${filename}`n  expected: $expected`n  actual:   $actual"
    }

    Write-Info "Checksum verified"
}

# --- pick install directory ---
function Get-InstallDir {
    if ($env:LAMPLIGHT_INSTALL_DIR) { return $env:LAMPLIGHT_INSTALL_DIR }
    return Join-Path $env:LOCALAPPDATA "lamplight"
}

# --- add directory to user PATH if not already there ---
function Add-ToPath {
    param($Dir)

    $current = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($current -split ";" -contains $Dir) { return }

    [Environment]::SetEnvironmentVariable("PATH", "$current;$Dir", "User")
    Write-Warn "$Dir was not in your PATH — added it. Restart your terminal for it to take effect."
}

# --- main ---
function Main {
    Write-Info "Resolving latest version..."
    $version = if ($env:LAMPLIGHT_VERSION) { $env:LAMPLIGHT_VERSION } else { Get-LatestVersion }
    if (-not $version) { Write-Err "could not resolve latest version — check your connection or set LAMPLIGHT_VERSION" }

    $archive    = "${Binary}_${version}_windows_amd64.zip"
    $baseUrl    = "https://github.com/$Repo/releases/download/$version"
    $installDir = Get-InstallDir

    Write-Info "Installing $Binary $version (windows/amd64)"

    $tmp = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path "$_.dir" }
    try {
        $archivePath   = Join-Path $tmp $archive
        $checksumPath  = Join-Path $tmp "checksums.txt"

        Write-Info "Downloading..."
        Invoke-WebRequest "$baseUrl/$archive"      -OutFile $archivePath  -UseBasicParsing
        Invoke-WebRequest "$baseUrl/checksums.txt" -OutFile $checksumPath -UseBasicParsing

        Test-Checksum $archivePath $checksumPath

        Write-Info "Extracting..."
        Expand-Archive $archivePath -DestinationPath $tmp -Force

        $exeName  = "${Binary}.exe"
        $exeSrc   = Join-Path $tmp $exeName
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $exeDest  = Join-Path $installDir $exeName

        # show what's being replaced if already installed
        $existing = Get-Command $Binary -ErrorAction SilentlyContinue
        if ($existing) {
            $current = & $Binary version 2>$null
            Write-Info "Replacing existing install ($current)"
        }

        Move-Item $exeSrc $exeDest -Force

        Write-Success "Installed $Binary $version → $exeDest"

        Add-ToPath $installDir

        Write-Host ""
        & $exeDest version
    }
    finally {
        Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Main
