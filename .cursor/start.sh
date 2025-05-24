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

# Function to update specific local dependencies from go.mod
update_local_dep() {
	local repo_name="$1"
	local expected_branch="$2"
	local target_dir="$3"

	if [ -d "$target_dir" ]; then
		echo "🔄 Updating $repo_name..."
		cd "$target_dir"
		current_branch=$(git branch --show-current 2> /dev/null || echo "unknown")
		if [ "$current_branch" = "$expected_branch" ]; then
			git fetch origin 2> /dev/null || echo "ℹ️ Could not fetch $repo_name"
			git pull origin "$expected_branch" 2> /dev/null || echo "ℹ️ Could not pull $repo_name"
			echo "✅ $repo_name updated (branch: $current_branch)"
		else
			echo "⚠️ $repo_name branch mismatch (expected: $expected_branch, actual: $current_branch)"
		fi
		cd "/workspace"
	else
		echo "⚠️ $repo_name not found at $target_dir - run install.sh to set up dependencies"
	fi
}

# Update the 5 local dependencies from go.mod replace directives
echo "📂 Updating local dependencies from go.mod..."
PARENT_DIR="$(dirname "/workspace")"

update_local_dep "Apple VZ Fork" "feat/vm-console-devices" "$PARENT_DIR/vz"
update_local_dep "Containerd" "main" "$PARENT_DIR/containerd"
update_local_dep "Gvisor Tap VSock" "main" "$PARENT_DIR/gvisor-tap-vsock"
update_local_dep "Kata Containers" "vf" "$PARENT_DIR/kata-containers"

# Update Go module dependencies
echo "📦 Updating Go module dependencies..."
./gow mod download
./gow mod tidy

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

# Verify local dependencies status
echo "🔍 Verifying local dependencies from go.mod replace directives..."
deps=(
	"vz:feat/vm-console-devices"
	"containerd:main"
	"gvisor-tap-vsock:main"
	"kata-containers:vf"
)

all_deps_ok=true
for dep_info in "${deps[@]}"; do
	dep_name="${dep_info%:*}"
	expected_branch="${dep_info#*:}"
	dep_path="$PARENT_DIR/$dep_name"

	if [ -d "$dep_path" ]; then
		cd "$dep_path"
		current_branch=$(git branch --show-current 2> /dev/null || echo "unknown")
		if [ "$current_branch" = "$expected_branch" ]; then
			echo "✅ $dep_name (branch: $current_branch)"
		else
			echo "⚠️ $dep_name (expected: $expected_branch, actual: $current_branch)"
			all_deps_ok=false
		fi
		cd "/workspace"
	else
		echo "⚠️ Missing: $dep_name - go mod will use fallback"
		all_deps_ok=false
	fi
done

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

# Determine dependency status
if $all_deps_ok; then
	dep_status="ALL_LOCAL"
else
	dep_status="MIXED/FALLBACK"
fi

# Print current status
echo ""
echo "🎯 EC1 MICROVM DEVELOPMENT STATUS:"
echo "================================="
echo "📁 Workspace: /workspace"
echo "🔧 GOW ready: $(./gow version 2> /dev/null > /dev/null && echo 'YES' || echo 'NO')"
echo "🧪 Tests ready: $(./gow test ./... > /dev/null 2>&1 && echo 'YES' || echo 'PARTIAL/NO')"
echo "💻 Tmux sessions: $(tmux list-sessions 2> /dev/null | wc -l || echo '0')"
echo "📦 Dependencies: $dep_status"
echo ""
echo "🔥 Key Development Areas:"
echo "   • Firecracker API (pkg/firecracker/)"
echo "   • VMM Abstraction (pkg/vmm/)"
echo "   • Init Injection (pkg/bootloader/)"
echo "   • Performance Testing (pkg/testing/tstream/)"
echo ""
echo "📂 Local Dependencies Status (go.mod replace directives):"
for dep_info in "${deps[@]}"; do
	dep_name="${dep_info%:*}"
	expected_branch="${dep_info#*:}"
	dep_path="$PARENT_DIR/$dep_name"

	if [ -d "$dep_path" ]; then
		cd "$dep_path" > /dev/null 2>&1
		current_branch=$(git branch --show-current 2> /dev/null || echo "unknown")
		if [ "$current_branch" = "$expected_branch" ]; then
			echo "   ✅ $dep_name ($current_branch)"
		else
			echo "   ⚠️ $dep_name (expected: $expected_branch, actual: $current_branch)"
		fi
		cd "/workspace" > /dev/null 2>&1
	else
		echo "   ❌ $dep_name (missing - go mod fallback)"
	fi
done
echo ""
echo "🚀 Ready for development! Use aliases like 'firecracker', 'vmm', 'gowtest'"
echo "================================="
