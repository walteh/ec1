#! /bin/bash

set -e

wrk_dir="$HOME/Developer/tmp/ksa2/wrk"

mkdir -p "$wrk_dir/var/lib/containerd"
mkdir -p "$wrk_dir/var/run/containerd"

# # Ensure proper permissions
sudo chown -R "$(whoami)" "$wrk_dir"

sudo ./ctr.sh image pull "docker.io/library/busybox:latest" --all-platforms --platform=linux/arm64

sudo ./ctr.sh content fetch --platform=linux/arm64 "docker.io/library/busybox:latest"
