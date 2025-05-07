#!/bin/bash

set -e

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}"

address="/Users/dub6ix/Developer/tmp/ksa2/var/run/containerd/containerd.sock"

# echo ""
# echo "command:     sudo go run ./ctr/main.go $* --address=${address} --debug"
# echo "directory:   $(pwd)"
# echo ""

go build -o /Users/dub6ix/Developer/tmp/ksa2/wrk/ctr ./ctr/main.go

export LC_RPATH=/usr/local/lib
export fuse_CFLAGS="-I/usr/local/include/fuse -D_FILE_OFFSET_BITS=64 -D_DARWIN_C_SOURCE"
export fuse_LIBS="-L/usr/local/lib -lfuse-t -pthread"

CGO_ENABLED=1 go build -buildvcs=false -o /Users/dub6ix/Developer/tmp/ksa2/wrk/shim ./shim3

sudo /Users/dub6ix/Developer/tmp/ksa2/wrk/ctr --address=${address} --debug "$@"
