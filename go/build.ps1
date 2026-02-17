# Build the Discorbo Go bot
Write-Host "Building Discorbo Go bot..." -ForegroundColor Cyan

Set-Location $PSScriptRoot

# Build the executable
go build -o discorbo.exe

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Executable: discorbo.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
