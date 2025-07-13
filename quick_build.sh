#!/bin/bash

# Quick build of FIRE GUI using devctl

echo "Quick building FIRE GUI..."

/mnt/d/Projects/DevProxy/devctl.exe \
    -token 4064d8d901b152758feb320719cd3c059849dafe922919b7d9733e6beb2271b3 \
    -cwd D:\\Projects\\project_fire \
    go build -v -o fire-gui-latest.exe ./cmd/fire-gui

echo "Build command sent"