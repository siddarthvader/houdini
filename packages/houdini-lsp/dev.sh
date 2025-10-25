#!/bin/bash
# Development helper - watches build AND LSP logs

echo "🚀 Houdini LSP Development Mode"
echo ""
echo "Terminal 1: Build watcher"
echo "Terminal 2: LSP log watcher (showing below)"
echo ""
echo "Press Ctrl+C to stop"
echo "================================================"
echo ""

# Start build watcher in background
pnpm dev &
BUILD_PID=$!

# Wait a moment for build to start
sleep 1

# Watch LSP logs (filter for Houdini and errors)
echo "📝 Watching LSP logs..."
tail -f ~/.local/state/nvim/lsp.log 2>/dev/null | grep --line-buffered -E "(Houdini|houdini_lsp|ERROR)" &
TAIL_PID=$!

# Cleanup on exit
trap "kill $BUILD_PID $TAIL_PID 2>/dev/null; exit" INT TERM

# Wait
wait
