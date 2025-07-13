#!/bin/bash

# Build FIRE GUI from WSL using DevProxy
# Runs the rebuild_and_run.ps1 script on Windows side

echo "Building FIRE GUI on Windows..."

/mnt/d/Projects/DevProxy/devctl.exe \
    -token 4064d8d901b152758feb320719cd3c059849dafe922919b7d9733e6beb2271b3 \
    -cwd D:\\Projects\\project_fire \
    powershell -ExecutionPolicy Bypass -File ".\\rebuild_and_run.ps1"

echo "Build script launched"