#!/bin/bash

set -exuo pipefail

# This script makes debugging with sudo easier

CONFIG_PATH="${PWD}/cmd/kata-containerd-macos/config.toml"

# # Create required directory structure
mkdir -p "${PWD}/tmp/var/lib/containerd"
mkdir -p "${PWD}/tmp/var/run/containerd"

# # Ensure proper permissions
sudo chown -R $(whoami) "${PWD}/tmp"

# Run dlv with sudo if we want to debug
if [ "$1" = "debug" ]; then
	shift
	sudo -E /Users/dub6ix/go/bin/dlv debug \
		--headless \
		--listen=:2345 \
		--api-version=2 \
		--accept-multiclient \
		./cmd/kata-containerd-macos/containerd \
		-- --config="${CONFIG_PATH}" "$@"
else
	# Normal execution with sudo
	sudo -E go run ./cmd/kata-containerd-macos/containerd --config="${CONFIG_PATH}" "$@"
fi
