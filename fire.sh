#!/bin/bash
# F.I.R.E. GUI - Flexible launcher script

# Default to quick build
BUILD_MODE=""

# Parse arguments
case "$1" in
    "")
        # Default - quick build
        BUILD_MODE=""
        echo "Quick build and launch..."
        ;;
    "restart"|"r")
        BUILD_MODE="-NoBuild"
        echo "Restarting (no build)..."
        ;;
    "full"|"f")
        BUILD_MODE="-FullBuild"
        echo "Full rebuild and launch..."
        ;;
    "help"|"h"|"-h"|"--help")
        echo "F.I.R.E. GUI Launcher"
        echo "===================="
        echo ""
        echo "Usage: ./fire.sh [option]"
        echo ""
        echo "Options:"
        echo "  (none)     Quick build and launch (default)"
        echo "  restart|r  Just restart without building"
        echo "  full|f     Full rebuild without cache"
        echo "  help|h     Show this help"
        echo ""
        echo "Examples:"
        echo "  ./fire.sh           # Quick build and launch"
        echo "  ./fire.sh restart   # Just restart the GUI"
        echo "  ./fire.sh full      # Clean rebuild everything"
        echo ""
        exit 0
        ;;
    *)
        echo "Unknown option: $1"
        echo "Use './fire.sh help' for usage"
        exit 1
        ;;
esac

# Kill any existing tmux session
tmux kill-session -t firegui 2>/dev/null || true

# Run in tmux so we can see output
echo "Starting in tmux session 'firegui'..."
tmux new-session -d -s firegui "/mnt/d/Projects/DevProxy/devctl.exe \
    -token 4064d8d901b152758feb320719cd3c059849dafe922919b7d9733e6beb2271b3 \
    -cwd D:\\Projects\\project_fire \
    powershell -ExecutionPolicy Bypass -File fire_gui_runner.ps1 $BUILD_MODE"

echo ""
echo "Use 'tmux attach -t firegui' to see live output"
echo ""

# Wait and show initial output
sleep 3
echo "Status:"
echo "======="
tmux capture-pane -t firegui -p | tail -30 | grep -E "Building|Launching|Error|FAIL|completed|GUI" || echo "Building..."