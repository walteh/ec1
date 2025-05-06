#! /bin/bash

set -e

# This script is a shim for the containerd binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

# # Create required directory structure
mkdir -p "${this_files_dir}/out/var/lib/containerd"
mkdir -p "${this_files_dir}/out/var/run/containerd"

# # Ensure proper permissions
sudo chown -R "$(whoami)" "${this_files_dir}/out"

# replace INJECT_PWD in config.toml with the current directory and write to out/config.toml
sed "s|INJECT_PWD|${this_files_dir}|g" "${this_files_dir}/config.toml" > "${this_files_dir}/out/config.toml"

echo "${this_files_dir}/out/config.toml"
