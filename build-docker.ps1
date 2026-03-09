# Local Docker build script for Windows
# This script prepares the build context for Docker with the go-audible dependency

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BuildDir = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "docker-build-$(Get-Random)")

Write-Host "Preparing Docker build context in $BuildDir..." -ForegroundColor Cyan

# Copy audible-plex-downloader
Write-Host "Copying audible-plex-downloader..." -ForegroundColor Cyan
Copy-Item -Recurse -Path $ScriptDir -Destination (Join-Path $BuildDir "audible-plex-downloader")

# Copy or clone go-audible
$GoAudiblePath = Join-Path (Split-Path -Parent $ScriptDir) "go-audible"
if (Test-Path $GoAudiblePath) {
    Write-Host "Copying local go-audible from ../go-audible..." -ForegroundColor Cyan
    Copy-Item -Recurse -Path $GoAudiblePath -Destination (Join-Path $BuildDir "go-audible")
} else {
    Write-Host "Cloning go-audible from GitHub..." -ForegroundColor Cyan
    Push-Location $BuildDir
    git clone https://github.com/mstrhakr/go-audible.git go-audible
    Pop-Location
}

# Build the Docker image
Write-Host "Building Docker image..." -ForegroundColor Cyan
Push-Location $BuildDir
docker build -f audible-plex-downloader/Dockerfile -t audible-plex-downloader:local .
Pop-Location

Write-Host "Cleaning up build context..." -ForegroundColor Cyan
Remove-Item -Recurse -Force $BuildDir

Write-Host ""
Write-Host "✅ Docker image built successfully as 'audible-plex-downloader:local'" -ForegroundColor Green
Write-Host ""
Write-Host "To run:" -ForegroundColor Cyan
Write-Host "  docker run -d -p 8080:8080 -v ${PWD}/config:/config -v ${PWD}/audiobooks:/audiobooks audible-plex-downloader:local"
