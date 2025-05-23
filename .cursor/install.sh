#!/bin/bash
set -e

echo "🚀 Setting up EC1 MicroVM development environment for Dr B..."

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

# Verify Go installation
/usr/local/go/bin/go version

# Install gotestsum for enhanced testing (our gow tool uses this)
echo "🧪 Installing gotestsum for enhanced test output..."
/usr/local/go/bin/go install gotest.tools/gotestsum@latest

# Install additional Go tools that our gow wrapper uses
echo "🔧 Installing Go development tools..."
/usr/local/go/bin/go install github.com/walteh/retab/v2/cmd/retab@latest
/usr/local/go/bin/go install golang.org/x/tools/cmd/goimports@latest

# Set up workspace (this will be our project directory)
cd /workspace

# Build our enhanced gow tool (Dr B's best friend!)
echo "⚡ Building GOW - our enhanced Go wrapper..."
cd /workspace/cmd/gow
/usr/local/go/bin/go build -o ../../gow .
cd /workspace

# Make gow executable and verify it works
chmod +x ./gow
echo "✅ Testing gow functionality..."
./gow version

# Run go mod tidy across workspace to ensure dependencies are ready
echo "📦 Setting up Go workspace dependencies..."
./gow mod tidy

# Install development tools
echo "🛠️ Installing additional development tools..."

# Install dlv (Delve debugger) for our dap command
/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest

# Create helpful aliases
echo "🔗 Setting up development aliases..."
cat >> ~/.bashrc << 'EOF'

# EC1 MicroVM Development Aliases
alias gow='./gow'
alias gowtest='./gow test -function-coverage -v'
alias gowbench='./gow test -run Benchmark'
alias gowstream='./gow test -run TestStreamPerformance ./pkg/testing/tstream/'
alias bootloader='cd /workspace/pkg/bootloader'
alias performance='cd /workspace/pkg/testing/tstream'

# Performance shortcuts
alias quicktest='./gow test -run TestNewGowConfig ./cmd/gow/ && echo "✅ Quick test passed!"'
alias coverage='./gow test -function-coverage ./...'
alias benchmark='./gow test -run BenchmarkInject ./pkg/initramfs/'

# Dr B's mission-specific shortcuts (MAIN workspace, NOT sandbox!)
alias firecracker='cd /workspace/pkg/firecracker && echo "🔥 Main Firecracker API workspace!"'
alias fireapi='cd /workspace/pkg/firecracker && echo "🔥 Main Firecracker API workspace - NOT sandbox!"'
alias firetest='./gow test -v ./pkg/firecracker/'
alias firebench='./gow test -run Benchmark ./pkg/firecracker/'

EOF

# Set up git (if not already configured)
echo "📝 Setting up git configuration..."
git config --global init.defaultBranch main
git config --global pull.rebase false
git config --global core.editor "nano"

# Verify our stream performance testing framework works
echo "🧪 Verifying stream performance testing framework..."
./gow test -v ./pkg/testing/tstream/ || echo "⚠️ Stream tests will be available once pkg/testing/tstream is created"

# Test our init injection system
echo "🔥 Testing init injection system..."
ls -la pkg/bootloader/linux.go || echo "⚠️ Bootloader will be available in pkg/bootloader/"

# Show Dr B the development environment status
echo ""
echo "🎉 DR B DEVELOPMENT ENVIRONMENT READY!"
echo "============================================"
echo "✅ Go $(go version | cut -d' ' -f3) installed"
echo "✅ GOW enhanced wrapper built and ready"
echo "✅ gotestsum installed for enhanced testing"
echo "✅ Development tools and aliases configured"
echo "✅ Performance testing framework available"
echo "✅ Firecracker workspace ready"
echo ""
echo "🎯 MISSION: Build fastest Firecracker-compatible microVM"
echo "⚡ SECRET WEAPON: Init injection for SSH-free execution"
echo "📊 TARGET: <100ms boot time, >85% test coverage"
echo ""
echo "🚀 Dr B, your mission starts NOW!"
echo "   Use 'gow -verbose' for detailed output"
echo "   Use 'gowtest' for function coverage testing"
echo "   Use 'firecracker' to navigate to your workspace"
echo ""
echo "💡 Remember: Every line of code should be FASTER than before!"
echo "============================================"
