#!/bin/bash

set -e

# This script is a shim for the shim binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

export SHIM_LOG_FILE="${this_files_dir}/out/kata-shim-$(date +%s).log"
export CONTAINERD_SHIM_LOG_LEVEL=debug
export RUST_LOG=trace

echo -e "\n\nStarting shim with args: $*\n\n" > "$SHIM_LOG_FILE"
echo -e "\n\nlog file: $SHIM_LOG_FILE\n\n" >> "$SHIM_LOG_FILE"

CGO_ENABLED=1 go build -buildvcs=false -o out/shim ./shim3

./out/shim "$@"
