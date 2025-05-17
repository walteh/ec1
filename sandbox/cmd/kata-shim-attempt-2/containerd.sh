#!/bin/bash

set -e

# This script is a shim for the containerd binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

wrk_dir="$HOME/Developer/tmp/ksa2/wrk"

# echo ""
# echo "command:     sudo go run ./containerd/main.go --config=${config_path} -l=trace $*"
# echo "directory:   $(pwd)"
# echo ""

sed "s|INJECT_PWD|$wrk_dir|g" "${this_files_dir}/config.toml" > "$wrk_dir/config.toml"

go build -o "$wrk_dir/containerd" ./containerd/main.go

sudo "$wrk_dir/containerd" "$@" --config="${wrk_dir}/config.toml" -l=trace
