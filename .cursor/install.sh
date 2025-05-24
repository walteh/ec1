#!/bin/bash
set -e

echo "🚀 Setting up EC1 MicroVM development environment..."

# Update system and install essential packages
sudo apt-get update
sudo apt-get install -y \
	build-essential \
	git \
	curl \
	wget \
	jq \
	tmux \
	htop \
	watch \
	tree \
	unzip \
	ca-certificates \
	gnupg \
	lsb-release

# Install Go 1.24+ (latest)
echo "📦 Installing Go..."
cd /tmp
GO_VERSION="1.24.3"
wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc
export PATH=$PATH:/usr/local/go/bin:~/go/bin

# Set up private module access for github.com/walteh
echo 'export GOPRIVATE=github.com/walteh' >> ~/.bashrc
export GOPRIVATE=github.com/walteh

# Verify Go installation
/usr/local/go/bin/go version

# Install gotestsum for enhanced testing (our gow tool uses this)
echo "🧪 Installing gotestsum for enhanced test output..."
/usr/local/go/bin/go install gotest.tools/gotestsum@latest

# Install additional Go tools that our gow wrapper uses
echo "🔧 Installing Go development tools..."
/usr/local/go/bin/go install github.com/walteh/retab/v2/cmd/retab@latest
/usr/local/go/bin/go install golang.org/x/tools/cmd/goimports@latest
/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest
/usr/local/go/bin/go install github.com/vektra/mockery/v2@latest

# Set up workspace (this will be our project directory)
cd /workspace

# Verify gow tool is available (should be pre-built)
echo "⚡ Verifying GOW tool availability..."
if [ -f "./gow" ]; then
	chmod +x ./gow
	./gow version
	echo "✅ GOW wrapper is ready"
else
	echo "⚠️ GOW wrapper not found - will be built automatically when needed"
fi

# Run go mod tidy across workspace to ensure dependencies are ready
echo "📦 Setting up Go workspace dependencies..."
./gow mod tidy

# Create helpful aliases for development
echo "🔗 Setting up development aliases..."
cat >> ~/.bashrc << 'EOF'

# EC1 MicroVM Development Environment
export PATH="/workspace:$PATH"

# Core development aliases
alias gow='./gow'
alias gowtest='./gow test -function-coverage -v'
alias gowbench='./gow test -bench=.'
alias coverage='./gow test -function-coverage ./...'

# Package-specific navigation and testing
alias firecracker='cd /workspace/pkg/firecracker && echo "🔥 Firecracker API workspace"'
alias vmm='cd /workspace/pkg/vmm && echo "🖥️ VMM abstraction layer"'
alias bootloader='cd /workspace/pkg/bootloader && echo "🔧 Init injection system"'
alias performance='cd /workspace/pkg/testing/tstream && echo "📊 Performance testing tools"'

# Testing shortcuts
alias firetest='./gow test -v ./pkg/firecracker/'
alias vmmtest='./gow test -v ./pkg/vmm/'
alias streamtest='./gow test -v ./pkg/testing/tstream/'
alias fulltest='./gow test -function-coverage ./...'

# Performance monitoring
alias benchmark-firecracker='./gow test -bench=. ./pkg/firecracker/'
alias benchmark-vmm='./gow test -bench=. ./pkg/vmm/'
alias benchmark-inject='./gow test -run BenchmarkInject ./pkg/initramfs/'

EOF

# Set up git configuration
echo "📝 Setting up git configuration..."
git config --global init.defaultBranch main
git config --global pull.rebase false
git config --global core.editor "nano"

# Verify key project components
echo "🧪 Verifying project components..."

# Check stream performance testing framework
if [ -d "pkg/testing/tstream" ]; then
	./gow test -v ./pkg/testing/tstream/ && echo "✅ Stream performance tools working"
else
	echo "⚠️ Stream performance tools not found - will be available in pkg/testing/tstream/"
fi

# Check VMM abstraction layer
if [ -d "pkg/vmm" ]; then
	echo "✅ VMM abstraction layer available"
else
	echo "⚠️ VMM layer not found - will be available in pkg/vmm/"
fi

# Check Firecracker API implementation
if [ -d "pkg/firecracker" ]; then
	echo "✅ Firecracker API implementation available"
else
	echo "⚠️ Firecracker API not found - will be available in pkg/firecracker/"
fi

# Check init injection system
if [ -d "pkg/bootloader" ]; then
	echo "✅ Init injection system available"
else
	echo "⚠️ Init injection system not found - will be available in pkg/bootloader/"
fi

# Show development environment status
echo ""
echo "🎉 EC1 MICROVM DEVELOPMENT ENVIRONMENT READY!"
echo "============================================"
echo "✅ Go $(go version | cut -d' ' -f3) installed"
echo "✅ Development tools and aliases configured"
echo "✅ Private module access configured (GOPRIVATE)"
echo "✅ GOW wrapper available for enhanced development"
echo "✅ Testing framework ready (gotestsum, mockery)"
echo ""
echo "🎯 MISSION: Make VMs as easy and fast as Docker containers"
echo "⚡ SECRET WEAPON: Init injection for SSH-free execution"
echo "🏁 TARGETS: <100ms boot time, >85% test coverage, <50MB memory"
echo "🍎 FOCUS: macOS first with Apple Virtualization Framework"
echo ""
echo "🔥 Key Development Areas:"
echo "   • pkg/firecracker/ - Firecracker API compatibility"
echo "   • pkg/vmm/ - Virtual machine abstraction"
echo "   • pkg/bootloader/ - Init injection system"
echo "   • pkg/testing/tstream/ - Performance testing tools"
echo ""
echo "🚀 Quick Start Commands:"
echo "   ./gow test -function-coverage ./...  # Run all tests with coverage"
echo "   firecracker                         # Navigate to Firecracker workspace"
echo "   gowtest                            # Quick test with coverage"
echo "   benchmark-firecracker              # Performance benchmarks"
echo ""
echo "💡 Remember: Every feature must be faster and more efficient!"
echo "============================================"
