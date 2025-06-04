package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	pattern := flag.String("pattern", "containerd-shim-harpoon-v1", "pgrep pattern for shim")
	port := flag.Int("port", 2345, "port the DAP server should listen on")
	delve := flag.String("dlv", "go", "dlv wrapper: 'go' for 'go tool dlv' or absolute path to dlv")
	clean := flag.Bool("clean", true, "kill orphan shim/dlv processes first")
	flag.Parse()

	// 0️⃣ tidy up
	if *clean {
		_ = exec.Command("pkill", "-f",
			fmt.Sprintf("%s tool dlv dap .*%d", *delve, *port)).Run()
		_ = exec.Command("pkill", "-f", *pattern).Run()
	}

	// 1️⃣ wait for newest shim PID
	var pid string
	for {
		out, _ := exec.Command("pgrep", "-fn", *pattern).Output()
		pid = strings.TrimSpace(string(out))
		if pid != "" {
			fmt.Printf("🚀 shim PID %s\n", pid)
			break
		}
		fmt.Println("⌛ waiting for shim")
		time.Sleep(50 * time.Millisecond)
	}

	// 2️⃣ build & exec the dlv dap command — different layout depending on wrapper
	var args []string
	listenFlag := fmt.Sprintf("--listen=127.0.0.1:%d", *port)

	if *delve == "go" {
		// use `go tool dlv ...`
		args = []string{
			"tool", "dlv", "dap",
			listenFlag,
			"--accept-multiclient",
			"--log",
			"attach", pid,
		}
	} else {
		// absolute/path/to/dlv ...
		args = []string{
			*delve, "dap",
			listenFlag,
			"--accept-multiclient",
			"--log",
			"attach", pid,
		}
	}

	fmt.Printf("🛠  exec %s %s\n", *delve, strings.Join(args, " "))

	cmd := exec.Command(*delve, args...)
	// mirror stdio so VS Code gets logs
	cmd.Stdout, cmd.Stderr, cmd.Stdin = os.Stdout, os.Stderr, os.Stdin

	// replace the current helper with dlv (never returns on success)
	if err := cmd.Run(); err != nil {
		log.Fatalf("dlv: %v", err)
	}
}
