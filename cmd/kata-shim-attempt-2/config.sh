#! /bin/bash

set -e

# This script is a shim for the containerd binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

# # Create required directory structure
mkdir -p "/Users/dub6ix/Developer/tmp/ksa2/wrk/var/lib/containerd"
mkdir -p "/Users/dub6ix/Developer/tmp/ksa2/wrk/var/run/containerd"

# # Ensure proper permissions
sudo chown -R "$(whoami)" "/Users/dub6ix/Developer/tmp/ksa2/wrk"

# replace INJECT_PWD in config.toml with the current directory and write to out/config.toml
sed "s|INJECT_PWD|/Users/dub6ix/Developer/tmp/ksa2/wrk|g" "${this_files_dir}/config.toml" > "/Users/dub6ix/Developer/tmp/ksa2/wrk/config.toml"

echo "/Users/dub6ix/Developer/tmp/ksa2/wrk/config.toml"
