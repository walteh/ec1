#!/bin/bash
set -e

echo "🔥 Starting EC1 MicroVM development environment..."

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
[ -f "./gow" ] && echo "✅ GOW tool available"
[ -f ".cursor/README.md" ] && echo "✅ Documentation available" || echo "ℹ️ Documentation in .cursor/README.md"

# Quick test to ensure everything works
echo "🧪 Running quick smoke test..."
if [ -d "tools/cmd/gow" ]; then
	./gow test -run TestNewGowConfig ./tools/cmd/gow/ && echo "✅ Test framework working"
else
	echo "ℹ️ GOW source tests will be available when tools/cmd/gow/ exists"
fi

# Set up tmux session for background work
echo "📺 Setting up tmux session for development work..."
tmux new-session -d -s ec1-dev 2> /dev/null || echo "ℹ️ Tmux session already exists or tmux not available"

# Create development windows in tmux if tmux is available
if command -v tmux > /dev/null 2>&1; then
	# Create windows for different development areas
	tmux new-window -t ec1-dev:1 -n 'firecracker' -c '/workspace/pkg/firecracker' 2> /dev/null || true
	tmux new-window -t ec1-dev:2 -n 'vmm' -c '/workspace/pkg/vmm' 2> /dev/null || true
	tmux new-window -t ec1-dev:3 -n 'bootloader' -c '/workspace/pkg/bootloader' 2> /dev/null || true
	tmux new-window -t ec1-dev:4 -n 'performance' -c '/workspace/pkg/testing/tstream' 2> /dev/null || true
fi

# Print current status
echo ""
echo "🎯 EC1 MICROVM DEVELOPMENT STATUS:"
echo "================================="
echo "📁 Workspace: /workspace"
echo "🔧 GOW ready: $(./gow version 2> /dev/null > /dev/null && echo 'YES' || echo 'NO')"
echo "🧪 Tests ready: $(./gow test ./... > /dev/null 2>&1 && echo 'YES' || echo 'PARTIAL/NO')"
echo "💻 Tmux sessions: $(tmux list-sessions 2> /dev/null | wc -l || echo '0')"
echo ""
echo "🔥 Key Development Areas:"
echo "   • Firecracker API (pkg/firecracker/)"
echo "   • VMM Abstraction (pkg/vmm/)"
echo "   • Init Injection (pkg/bootloader/)"
echo "   • Performance Testing (pkg/testing/tstream/)"
echo ""
echo "🚀 Ready for development! Use aliases like 'firecracker', 'vmm', 'gowtest'"
echo "================================="
