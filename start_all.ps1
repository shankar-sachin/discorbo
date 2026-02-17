# Start both AI Chat Service and Discord Bot
Write-Host "Starting Discorbo with AI Chat..." -ForegroundColor Cyan

# Check if Python is installed
if (-not (Get-Command python -ErrorAction SilentlyContinue)) {
    Write-Host "Python not found! Please install Python 3.8+ first." -ForegroundColor Red
    exit 1
}

# Check if virtual environment exists
if (-not (Test-Path "venv")) {
    Write-Host "Creating Python virtual environment..." -ForegroundColor Yellow
    python -m venv venv
}

# Activate virtual environment and install dependencies
Write-Host "Installing Python dependencies..." -ForegroundColor Yellow
& .\venv\Scripts\Activate.ps1
pip install -q -r requirements.txt

# Start AI Chat Service in background
Write-Host "Starting AI Chat Service..." -ForegroundColor Green
$aiProcess = Start-Process python -ArgumentList "ai_chat.py" -PassThru -WindowStyle Hidden

# Wait a moment for AI service to start
Start-Sleep -Seconds 2

# Start Discord Bot
Write-Host "Starting Discord Bot..." -ForegroundColor Green
Set-Location go
& .\run.ps1

# Cleanup on exit
if ($aiProcess -and !$aiProcess.HasExited) {
    Write-Host "Stopping AI Chat Service..." -ForegroundColor Yellow
    Stop-Process -Id $aiProcess.Id -Force
}
