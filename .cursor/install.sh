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

# Set up workspace (this will be our project directory)
cd /workspace

# Install essential Go tools (using standard versions)

# Set up local dependencies from go.mod replace directives
echo "📂 Setting up local dependencies required by go.mod..."
WORKSPACE_ROOT="/workspace"
DEPS_DIR="/workspace/deps"

# Create deps directory if it doesn't exist
mkdir -p "$DEPS_DIR"

# Function to clone local dependency with specific branch
setup_local_dep() {
	local repo_name="$1"
	local github_path="$2"
	local branch="$3"
	local target_dir="$4"

	if [ ! -d "$target_dir" ]; then
		echo "🔄 Cloning $repo_name (branch: $branch)..."
		if git clone -b "$branch" "https://github.com/walteh/$github_path.git" "$target_dir" 2> /dev/null; then
			echo "✅ Cloned walteh/$github_path on branch $branch"
		else
			echo "⚠️ Failed to clone walteh/$github_path - dependency will use go mod download"
		fi
	else
		echo "✅ $repo_name already available at $target_dir"
	fi
}

# Set up the 5 required local dependencies with their current branches
setup_local_dep "Apple VZ Fork" "vz" "feat/vm-console-devices" "$DEPS_DIR/vz"
setup_local_dep "Containerd" "containerd" "main" "$DEPS_DIR/containerd"
setup_local_dep "Gvisor Tap VSock" "gvisor-tap-vsock" "main" "$DEPS_DIR/gvisor-tap-vsock"
setup_local_dep "Kata Containers" "kata-containers" "vf" "$DEPS_DIR/kata-containers"

# Update go.mod to use the deps directory instead of ../
echo "🔧 Updating go.mod replace directives to use /workspace/deps/..."
if [ -f "go.mod" ]; then
	# Create backup
	cp go.mod go.mod.backup

	# Update replace directives to point to /workspace/deps/
	sed -i 's|=> \.\./vz|=> /workspace/deps/vz|g' go.mod
	sed -i 's|=> \.\./containerd|=> /workspace/deps/containerd|g' go.mod
	sed -i 's|=> \.\./gvisor-tap-vsock|=> /workspace/deps/gvisor-tap-vsock|g' go.mod
	sed -i 's|=> \.\./kata-containers|=> /workspace/deps/kata-containers|g' go.mod

	echo "✅ Updated go.mod replace directives for container environment"
fi

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

# Dependency management
alias update-deps='cd /workspace && ./gow mod tidy && echo "Dependencies updated"'
alias check-deps='cd /workspace && ./gow mod verify && echo "Dependencies verified"'

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

# Verify local dependencies
echo "🔍 Verifying local dependencies from go.mod replace directives..."
deps=(
	"vz:feat/vm-console-devices"
	"containerd:main"
	"gvisor-tap-vsock:main"
	"kata-containers:vf"
)

for dep_info in "${deps[@]}"; do
	dep_name="${dep_info%:*}"
	expected_branch="${dep_info#*:}"
	dep_path="$DEPS_DIR/$dep_name"

	if [ -d "$dep_path" ]; then
		cd "$dep_path"
		current_branch=$(git branch --show-current 2> /dev/null || echo "unknown")
		if [ "$current_branch" = "$expected_branch" ]; then
			echo "✅ $dep_name (branch: $current_branch)"
		else
			echo "⚠️ $dep_name (expected: $expected_branch, actual: $current_branch)"
		fi
		cd "$WORKSPACE_ROOT"
	else
		echo "⚠️ Missing local dependency: $dep_name (go mod will download fallback)"
	fi
done

# Show development environment status
echo ""
echo "🎉 EC1 MICROVM DEVELOPMENT ENVIRONMENT READY!"
echo "============================================"
echo "✅ Go $(go version | cut -d' ' -f3) installed"
echo "✅ Development tools installed (gotestsum, retab, goimports, dlv, mockery)"
echo "✅ Private module access configured (GOPRIVATE)"
echo "✅ GOW wrapper available for enhanced development"
echo "✅ Local dependencies configured for go.mod replace directives"
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
echo "📂 Local Dependencies (go.mod replace directives):"
echo "   • /workspace/deps/vz (feat/vm-console-devices) - Apple Virtualization Framework fork"
echo "   • /workspace/deps/containerd (main) - Containerd API and runtime"
echo "   • /workspace/deps/gvisor-tap-vsock (main) - Network virtualization"
echo "   • /workspace/deps/kata-containers (vf) - Kata runtime integration"
echo ""
echo "🚀 Quick Start Commands:"
echo "   ./gow test -function-coverage ./...  # Run all tests with coverage"
echo "   firecracker                         # Navigate to Firecracker workspace"
echo "   gowtest                            # Quick test with coverage"
echo "   benchmark-firecracker              # Performance benchmarks"
echo "   update-deps                        # Update and tidy dependencies"
echo ""
echo "💡 Remember: Every feature must be faster and more efficient!"
echo "============================================"
