#!/bin/bash
set -euo pipefail

# Configuration
KERNEL_VERSION="${KERNEL_VERSION:-6.15-rc7}"
KERNEL_CONFIG="${KERNEL_CONFIG:-ec1-aarch64.config}"
OUTPUT_DIR="${OUTPUT_DIR:-./gen/vmlinux}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
	echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
	echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
	echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
	exit 1
}

# Check if config file exists
if [[ ! -f "docker/kernel/${KERNEL_CONFIG}" ]]; then
	error "Kernel config file docker/kernel/${KERNEL_CONFIG} not found!"
fi

# Create output directory
log "Creating output directory: ${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

# Build kernel and extract to local directory
log "Building kernel with config: ${KERNEL_CONFIG}, version: ${KERNEL_VERSION}"
docker buildx build \
	--build-arg KERNEL_VERSION="${KERNEL_VERSION}" \
	--build-arg KERNEL_CONFIG="${KERNEL_CONFIG}" \
	--target export \
	--output type=local,dest="${OUTPUT_DIR}" \
	docker/kernel

# Check if build was successful
if [[ -f "${OUTPUT_DIR}/vmlinux" ]]; then
	log "✅ Kernel build successful!"
	log "📁 Kernel image: ${OUTPUT_DIR}/vmlinux"
	log "📁 Config file: ${OUTPUT_DIR}/config-${KERNEL_VERSION}"

	# Show file sizes
	log "📊 File sizes:"
	ls -lh "${OUTPUT_DIR}/"
else
	error "❌ Kernel build failed - vmlinux not found in ${OUTPUT_DIR}"
fi
