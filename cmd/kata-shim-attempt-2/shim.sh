#!/bin/bash

set -e

# This script is a shim for the shim binary

this_files_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "${this_files_dir}/cmd/shim"

go run main.go "$@"
