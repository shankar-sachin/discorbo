# Start just the AI Chat Service
Write-Host "Starting AI Chat Service..." -ForegroundColor Cyan

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
Write-Host "Installing dependencies..." -ForegroundColor Yellow
& .\venv\Scripts\Activate.ps1
pip install -q -r requirements.txt

# Start AI service
Write-Host "AI Chat Service running on http://localhost:5000" -ForegroundColor Green
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
python ai_chat.py
