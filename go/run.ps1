# Run the Discorbo Go bot
Write-Host "Starting Discorbo Go bot..." -ForegroundColor Cyan

Set-Location $PSScriptRoot

# Check if executable exists, build if not
if (-not (Test-Path "discorbo.exe")) {
    Write-Host "Executable not found. Building..." -ForegroundColor Yellow
    & .\build.ps1
    if ($LASTEXITCODE -ne 0) {
        exit 1
    }
}

# Run the bot
Write-Host "Running bot..." -ForegroundColor Green
& .\discorbo.exe
