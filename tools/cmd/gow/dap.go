package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// handleDap processes dap commands
func (cfg *GowConfig) handleDap(args []string) error {

	var root bool
	var listenAddress string

	argz := []string{}

	for _, arg := range args[1:] {
		if arg == "-root" {
			root = true
		} else if strings.HasPrefix(arg, "--listen=") {
			listenAddress = strings.TrimPrefix(arg, "--listen=")
			argz = append(argz, arg)
		} else {
			argz = append(argz, arg)
		}
	}

	// var addr string = "127.0.0.1:22345"



	// argz = append(argz, "--client-addr="+addr)

	if root && os.Geteuid() != 0 {
		fmt.Println("debug: root is required for -root flag")
		return fmt.Errorf("root is required for -root flag")
	}

	updatedEnv, updatedPath, cleanup, err := addSelfAsGoToPath(CommandDap)
	if err != nil {
		return fmt.Errorf("add self as go to path: %w", err)
	}
	defer cleanup()

	os.Setenv("PATH", updatedPath)
	os.Setenv("GOW_DAP_WRAP_ADDRESS", listenAddress)
	os.Setenv("DAP_LISTEN_ADDRESS", listenAddress)

	dlvCmd := exec.Command("dlv", append([]string{"dap"}, argz...)...)
	dlvCmd.Env = append(updatedEnv, "GOW_DAP_WRAP_ADDRESS="+listenAddress, "DAP_LISTEN_ADDRESS="+listenAddress)
	dlvCmd.Stdout = stdout
	dlvCmd.Stderr = stderr
	dlvCmd.Stdin = stdin

	return dlvCmd.Run()
}
