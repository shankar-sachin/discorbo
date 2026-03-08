# Run the Discorbo Go bot
Write-Host "Starting Discorbo Go bot..." -ForegroundColor Cyan

Set-Location $PSScriptRoot

# Detect current architecture
$arch = (Get-CimInstance Win32_Processor).Architecture
# 0=x86, 9=x64(AMD64), 12=ARM64
$exe = if ($arch -eq 12) { "discorbo-arm64.exe" } else { "discorbo-amd64.exe" }

Write-Host "Detected architecture: $(if ($arch -eq 12) { 'ARM64' } else { 'AMD64' })" -ForegroundColor Gray

# Always rebuild to ensure binaries are up to date
Write-Host "Building for current platform..." -ForegroundColor Yellow
& .\build.ps1
if ($LASTEXITCODE -ne 0) {
    exit 1
}

# Run the correct binary
Write-Host "Running bot ($exe)..." -ForegroundColor Green
& ".\$exe"
