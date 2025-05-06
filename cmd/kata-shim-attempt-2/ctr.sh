#!/bin/bash

set -e

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

address="/tmp/kata-shim-attempt-2/var/run/containerd/containerd.sock"

echo ""
echo "command:     sudo go run ./ctr/main.go $* --address=${address} --debug"
echo "directory:   $(pwd)"
echo ""

go build -o out/ctr ./ctr/main.go

sudo ./out/ctr --address=${address} --debug "$@"
