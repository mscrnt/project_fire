# Quick test to see if GUI launches

Write-Host "Quick GUI Test" -ForegroundColor Cyan
Write-Host "=============" -ForegroundColor Cyan

# Kill existing
Stop-Process -Name "fire-gui" -Force -ErrorAction SilentlyContinue

# Check if exe exists
if (Test-Path ".\fire-gui.exe") {
    Write-Host "fire-gui.exe found, launching..." -ForegroundColor Green
    
    try {
        & ".\fire-gui.exe"
    } catch {
        Write-Host "Error: $_" -ForegroundColor Red
    }
} else {
    Write-Host "fire-gui.exe not found!" -ForegroundColor Red
    Write-Host "Current directory: $(Get-Location)" -ForegroundColor Yellow
    Write-Host "Files:" -ForegroundColor Yellow
    Get-ChildItem *.exe
}

Read-Host "Press Enter to exit"