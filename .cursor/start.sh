#!/bin/bash
set -e

echo "🔥 Starting EC1 MicroVM development environment for Dr B..."

# Ensure we're in the workspace directory
cd /workspace

# Export essential environment variables
export PATH=$PATH:/usr/local/go/bin:~/go/bin
export GOPROXY='https://proxy.golang.org'
export GOSUMDB='sum.golang.org'
export GOPRIVATE='github.com/walteh'

# Set up any environment variables our gow tool might need
export GOW_WORKSPACE_ROOT="/workspace"
export GOW_VERBOSE="false"

# Ensure gow is executable and ready
chmod +x ./gow

# Quick health check of our development environment
echo "🔍 Running development environment health check..."

# Check Go installation
go version && echo "✅ Go is ready"

# Check gow functionality
./gow version && echo "✅ GOW wrapper is ready"

# Verify workspace structure
ls -la cmd/gow/main.go && echo "✅ GOW source code available"
ls -la .cursor/README.md && echo "✅ Documentation available"

# Quick test to ensure everything works
echo "🧪 Running quick smoke test..."
./gow test -run TestNewGowConfig ./cmd/gow/ && echo "✅ Test framework working"

# Set up tmux session for Dr B's work
echo "📺 Setting up tmux session for background work..."
tmux new-session -d -s drb-session

# Create development windows in tmux
tmux new-window -t drb-session:1 -n 'firecracker-main' -c '/workspace/pkg/firecracker'
tmux new-window -t drb-session:2 -n 'performance-testing' -c '/workspace/pkg/testing/tstream'
tmux new-window -t drb-session:3 -n 'bootloader' -c '/workspace/pkg/bootloader'

# Print current status
echo ""
echo "🎯 DR B ENVIRONMENT STATUS:"
echo "=========================="
echo "📁 Workspace: /workspace"
echo "🔧 GOW ready: $(./gow version 2> /dev/null && echo 'YES' || echo 'NO')"
echo "🧪 Tests ready: $(./gow test -run TestNewGowConfig ./cmd/gow/ > /dev/null 2>&1 && echo 'YES' || echo 'NO')"
echo "💻 Tmux sessions available: $(tmux list-sessions | wc -l)"
echo ""
echo "🚀 Ready for Firecracker integration development!"
echo "=========================="
