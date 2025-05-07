#!/bin/bash

set -e

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

wrk_dir="$HOME/Developer/tmp/ksa2/wrk"

address="$wrk_dir/var/run/containerd/containerd.sock"

# echo ""
# echo "command:     sudo go run ./ctr/main.go $* --address=${address} --debug"
# echo "directory:   $(pwd)"
# echo ""

go build -o "$wrk_dir/ctr" ./ctr/main.go

export LC_RPATH=/usr/local/lib
export fuse_CFLAGS="-I/usr/local/include/fuse -D_FILE_OFFSET_BITS=64 -D_DARWIN_C_SOURCE"
export fuse_LIBS="-L/usr/local/lib -lfuse-t -pthread"

CGO_ENABLED=1 go build -buildvcs=false -o "$wrk_dir/shim" ./shim3

go run "../codesign" "$wrk_dir/shim" || true

sed "s|INJECT_PWD|$wrk_dir|g" "${this_files_dir}/kata.toml" > "$wrk_dir/kata.toml"

sudo "$wrk_dir/ctr" --address="${address}" --debug "$@"
