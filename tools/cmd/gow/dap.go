package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func addSelfAsGoToPath() (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "gow-go-injected-*")
	if err != nil {
		return "", nil, fmt.Errorf("create temp directory: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return "", nil, fmt.Errorf("get executable: %w", err)
	}

	// create a symlink to the current binary named 'go' and add it to the PATH
	err = os.Symlink(executable, filepath.Join(tmpDir, "go"))
	if err != nil {
		return "", nil, fmt.Errorf("create symlink: %w", err)
	}
	path := os.Getenv("PATH")
	pathDirs := strings.Split(path, string(os.PathListSeparator))
	pathDirs = append([]string{tmpDir}, pathDirs...)

	updatedPath := strings.Join(pathDirs, string(os.PathListSeparator))

	err = os.Setenv("PATH", updatedPath)
	if err != nil {
		return "", nil, fmt.Errorf("set PATH: %w", err)
	}
	fmt.Printf("I am %s, added %s to PATH as a symlink\n", executable, updatedPath)

	return updatedPath, func() {
		os.Setenv("PATH", strings.ReplaceAll(os.Getenv("PATH"), tmpDir+string(os.PathListSeparator), ""))
		os.RemoveAll(tmpDir)
	}, nil
}

// handleDap processes dap commands
func (cfg *GowConfig) handleDap(args []string) error {

	var root bool

	argz := []string{}

	for _, arg := range args[1:] {
		if arg == "-root" {
			root = true
		} else {
			argz = append(argz, arg)
		}
	}

	if root && os.Geteuid() != 0 {
		fmt.Println("debug: root is required for -root flag")
		return fmt.Errorf("root is required for -root flag")
	}

	updatedPath, cleanup, err := addSelfAsGoToPath()
	if err != nil {
		return fmt.Errorf("add self as go to path: %w", err)
	}

	defer cleanup()


	dlvCmd := exec.Command("dlv", append([]string{"dap"}, argz...)...)
	dlvCmd.Env = append([]string{"PATH=" + updatedPath}, os.Environ()...)
	dlvCmd.Stdout = stdout
	dlvCmd.Stderr = stderr
	dlvCmd.Stdin = stdin

	return dlvCmd.Run()
}
