//go:build linux

// File: init.go
// Build with: CGO_ENABLED=0 go build -o init init.go

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
)

func main() {
	// 1. Parse kernel cmdline
	tag, mnt := "shared", "/mnt/shared"
	if f, err := os.Open("/proc/cmdline"); err == nil {
		defer f.Close()
		reader := bufio.NewReader(f)
		if line, err := reader.ReadString('\n'); err == nil || err == io.EOF {
			for _, kv := range strings.Fields(line) {
				if parts := strings.SplitN(kv, "=", 2); len(parts) == 2 {
					switch parts[0] {
					case "virtiofs.tag":
						tag = parts[1]
					case "virtiofs.mnt":
						mnt = parts[1]
					}
				}
			}
		}
	}

	// 2. Create mount point
	if err := os.MkdirAll(mnt, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create mount dir %s: %v\n", mnt, err)
	}

	// 3. Perform the mount
	if err := syscall.Mount(tag, mnt, "virtiofs", 0, ""); err != nil {
		fmt.Fprintf(os.Stderr, "virtiofs mount failed: %v\n", err)
	}

	// 4. Exec the real init
	realInit := "/init.real"

	// Replace this process with the real init
	if err := syscall.Exec(realInit, []string{realInit}, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec %s failed: %v\n", realInit, err)
		os.Exit(1)
	}
}
