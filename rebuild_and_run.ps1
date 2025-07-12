# F.I.R.E. GUI - Kill, Rebuild (no cache), and Launch Script

Write-Host "F.I.R.E. GUI - Full Rebuild and Launch" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan
Write-Host ""

# 1. Kill any existing fire-gui.exe processes
Write-Host "Killing any existing fire-gui.exe processes..." -ForegroundColor Yellow
Stop-Process -Name "fire-gui" -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500

# 2. Clear build cache
Write-Host "Clearing Go build cache..." -ForegroundColor Yellow
& go clean -cache

# 3. Clear old logs for fresh debugging
Write-Host "Clearing old logs..." -ForegroundColor Yellow
Remove-Item "gui_debug.log" -ErrorAction SilentlyContinue
Remove-Item "fire-gui.log" -ErrorAction SilentlyContinue
Remove-Item "perf.log" -ErrorAction SilentlyContinue
Remove-Item "build.log" -ErrorAction SilentlyContinue

# 4. Set environment variables
$env:PATH = "C:\ProgramData\mingw64\mingw64\bin;C:\Program Files\Go\bin;" + $env:PATH
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"

# 5. Build fire-gui.exe (force rebuild, no cache)
Write-Host "`nBuilding fire-gui.exe (no cache)..." -ForegroundColor Yellow
$buildStart = Get-Date

$buildProcess = Start-Process -FilePath "go" `
    -ArgumentList "build", "-a", "-v", "-o", "fire-gui.exe", "./cmd/fire-gui" `
    -NoNewWindow -PassThru -RedirectStandardError "build_error.log" `
    -RedirectStandardOutput "build_output.log"

$buildProcess.WaitForExit()

if ($buildProcess.ExitCode -ne 0) {
    Write-Host "Build FAILED!" -ForegroundColor Red
    Write-Host "`nBuild errors:" -ForegroundColor Red
    Get-Content "build_error.log" -ErrorAction SilentlyContinue
    exit 1
}

$buildEnd = Get-Date
$buildTime = $buildEnd - $buildStart
Write-Host "Build completed in $($buildTime.TotalSeconds) seconds!" -ForegroundColor Green

# 6. Check if fire-gui.exe was created
if (-not (Test-Path ".\fire-gui.exe")) {
    Write-Host "Error: fire-gui.exe was not created!" -ForegroundColor Red
    exit 1
}

# 7. Launch fire-gui.exe
Write-Host "`nLaunching F.I.R.E. GUI..." -ForegroundColor Green
Start-Sleep -Milliseconds 500

# Run directly to see console output
try {
    & ".\fire-gui.exe"
    Write-Host "`nF.I.R.E. GUI exited!" -ForegroundColor Green
} catch {
    Write-Host "Error running F.I.R.E. GUI: $_" -ForegroundColor Red
}

# 8. Show logs if there were issues
if (Test-Path "gui_debug.log") {
    $logSize = (Get-Item "gui_debug.log").Length
    if ($logSize -gt 0) {
        Write-Host "`nDebug log (last 20 lines):" -ForegroundColor Yellow
        Get-Content "gui_debug.log" -Tail 20
    }
}

Write-Host "`nPress any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")