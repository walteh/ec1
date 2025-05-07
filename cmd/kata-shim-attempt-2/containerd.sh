#!/bin/bash

set -e

# This script is a shim for the containerd binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

config_path="$(./config.sh)"

# echo ""
# echo "command:     sudo go run ./containerd/main.go --config=${config_path} -l=trace $*"
# echo "directory:   $(pwd)"
# echo ""

go build -o /Users/dub6ix/Developer/tmp/ksa2/wrk/containerd ./containerd/main.go

sudo /Users/dub6ix/Developer/tmp/ksa2/wrk/containerd "$@" --config="${config_path}" -l=trace
