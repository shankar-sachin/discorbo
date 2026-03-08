# Build the Discorbo Go bot for both Windows architectures
Write-Host "Building Discorbo Go bot..." -ForegroundColor Cyan

Set-Location $PSScriptRoot

$env:GOOS = "windows"

# Build AMD64
Write-Host "  Building windows/amd64..." -ForegroundColor Gray
$env:GOARCH = "amd64"
go build -o discorbo-amd64.exe .
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed (amd64)!" -ForegroundColor Red; exit 1 }

# Build ARM64
Write-Host "  Building windows/arm64..." -ForegroundColor Gray
$env:GOARCH = "arm64"
go build -o discorbo-arm64.exe .
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed (arm64)!" -ForegroundColor Red; exit 1 }

Write-Host "Build successful! discorbo-amd64.exe + discorbo-arm64.exe" -ForegroundColor Green
