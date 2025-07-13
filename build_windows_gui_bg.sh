#!/bin/bash
# Shell wrapper to run Windows GUI build in background

echo "Starting Windows GUI build..."

# Create log file
LOG_FILE="/mnt/d/Projects/project_fire/build_gui.log"

# Run the PowerShell build script in background
cd /mnt/d/Projects/project_fire
powershell.exe -ExecutionPolicy Bypass -File 'D:\Projects\project_fire\rebuild_and_run.ps1' > "$LOG_FILE" 2>&1 &

# Get the PID
BUILD_PID=$!
echo "Build started with PID: $BUILD_PID"
echo "Log file: $LOG_FILE"

# Function to check build status
check_status() {
    if ps -p $BUILD_PID > /dev/null 2>&1; then
        echo "Build is still running..."
        echo "Last 10 lines of log:"
        tail -10 "$LOG_FILE"
    else
        echo "Build completed!"
        echo "Final log output:"
        tail -20 "$LOG_FILE"
    fi
}

echo ""
echo "To check status, run: tail -f $LOG_FILE"
echo "Build PID: $BUILD_PID"

# Wait a bit and show initial output
sleep 3
check_status