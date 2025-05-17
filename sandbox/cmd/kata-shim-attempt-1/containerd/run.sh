#!/bin/bash

image="docker.io/library/busybox:latest"

sudo go run main.go image pull "docker.io/library/busybox:latest" --config=./config.toml
sudo go run main.go run --config=./config.toml --runtime "io.containerd.kata.v2" --rm -t "$image" test-kata uname -r
