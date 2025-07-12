# F.I.R.E. GUI Runner - Flexible build/launch script
param(
    [switch]$NoBuild,      # Skip building, just launch
    [switch]$QuickBuild,   # Build with cache (fast)
    [switch]$FullBuild,    # Build without cache (slow, thorough)
    [switch]$Help          # Show help
)

if ($Help) {
    Write-Host ""
    Write-Host "F.I.R.E. GUI Runner" -ForegroundColor Cyan
    Write-Host "==================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\fire_gui_runner.ps1 [options]" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "Options:" -ForegroundColor Green
    Write-Host "  -NoBuild      Skip building, just kill existing and launch"
    Write-Host "  -QuickBuild   Build with cache (default, fast)"
    Write-Host "  -FullBuild    Build without cache (slow, thorough)"
    Write-Host "  -Help         Show this help"
    Write-Host ""
    Write-Host "Examples:" -ForegroundColor Green
    Write-Host "  .\fire_gui_runner.ps1                  # Quick build and launch"
    Write-Host "  .\fire_gui_runner.ps1 -NoBuild         # Just restart"
    Write-Host "  .\fire_gui_runner.ps1 -FullBuild       # Clean rebuild"
    Write-Host ""
    exit 0
}

Write-Host "F.I.R.E. GUI Runner" -ForegroundColor Cyan
Write-Host "==================" -ForegroundColor Cyan
Write-Host ""

# 1. Always kill existing processes
Write-Host "Killing any existing fire-gui.exe processes..." -ForegroundColor Yellow
Stop-Process -Name "fire-gui" -Force -ErrorAction SilentlyContinue
Start-Sleep -Milliseconds 500

# 2. Clear logs
Write-Host "Clearing old logs..." -ForegroundColor Yellow
Remove-Item "gui_debug.log" -ErrorAction SilentlyContinue
Remove-Item "fire-gui.log" -ErrorAction SilentlyContinue
Remove-Item "perf.log" -ErrorAction SilentlyContinue

# 3. Build if requested
if (-not $NoBuild) {
    # Set environment
    $env:PATH = "C:\ProgramData\mingw64\mingw64\bin;C:\Program Files\Go\bin;" + $env:PATH
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "1"
    
    if ($FullBuild) {
        Write-Host "`nPerforming FULL rebuild (no cache)..." -ForegroundColor Yellow
        Write-Host "This may take a while..." -ForegroundColor DarkGray
        & go clean -cache
        $buildArgs = @("build", "-a", "-v", "-o", "fire-gui.exe", "./cmd/fire-gui")
    } else {
        Write-Host "`nPerforming quick build..." -ForegroundColor Yellow
        $buildArgs = @("build", "-v", "-o", "fire-gui.exe", "./cmd/fire-gui")
    }
    
    $buildStart = Get-Date
    $buildResult = & go $buildArgs 2>&1
    $buildSuccess = $LASTEXITCODE -eq 0
    $buildEnd = Get-Date
    $buildTime = $buildEnd - $buildStart
    
    if ($buildSuccess) {
        Write-Host "Build completed in $($buildTime.TotalSeconds.ToString('F1')) seconds!" -ForegroundColor Green
    } else {
        Write-Host "Build FAILED!" -ForegroundColor Red
        Write-Host $buildResult -ForegroundColor Red
        exit 1
    }
} else {
    Write-Host "`nSkipping build (using existing fire-gui.exe)" -ForegroundColor Yellow
}

# 4. Check if exe exists
if (-not (Test-Path ".\fire-gui.exe")) {
    Write-Host "Error: fire-gui.exe not found!" -ForegroundColor Red
    Write-Host "Run without -NoBuild to compile first." -ForegroundColor Yellow
    exit 1
}

# 5. Launch
Write-Host "`nLaunching F.I.R.E. GUI..." -ForegroundColor Green
Write-Host "========================" -ForegroundColor Green
Write-Host ""

try {
    & ".\fire-gui.exe"
    $exitCode = $LASTEXITCODE
    if ($exitCode -eq 0) {
        Write-Host "`nF.I.R.E. GUI exited normally." -ForegroundColor Green
    } else {
        Write-Host "`nF.I.R.E. GUI exited with code: $exitCode" -ForegroundColor Yellow
    }
} catch {
    Write-Host "`nError running F.I.R.E. GUI: $_" -ForegroundColor Red
}

# 6. Show debug log if available
if (Test-Path "gui_debug.log") {
    $logSize = (Get-Item "gui_debug.log").Length
    if ($logSize -gt 0) {
        Write-Host "`nDebug log tail:" -ForegroundColor Yellow
        Get-Content "gui_debug.log" -Tail 20
    }
}

Write-Host "`nPress any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")