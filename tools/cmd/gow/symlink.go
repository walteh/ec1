package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Command string

const (
	CommandDap  Command = "dap"
	CommandTest Command = "test"
)

func isNestedBy(command Command) bool {
	env := os.Getenv("GOW_GO_CALLED_BY")
	return env == string(command)
}

func addSelfAsGoToPath(command Command) ([]string, string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "gow-go-injected-*")
	if err != nil {
		return nil, "", nil, fmt.Errorf("create temp directory: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return nil, "", nil, fmt.Errorf("get executable: %w", err)
	}

	// create a symlink to the current binary named 'go' and add it to the PATH
	err = os.Symlink(executable, filepath.Join(tmpDir, "go"))
	if err != nil {
		return nil, "", nil, fmt.Errorf("create symlink: %w", err)
	}
	path := os.Getenv("PATH")
	pathDirs := strings.Split(path, string(os.PathListSeparator))
	pathDirs = append([]string{tmpDir}, pathDirs...)

	updatedPath := strings.Join(pathDirs, string(os.PathListSeparator))

	err = os.Setenv("PATH", updatedPath)
	if err != nil {
		return nil, "", nil, fmt.Errorf("set PATH: %w", err)
	}
	fmt.Printf("I am %s, added %s to PATH as a symlink\n", executable, updatedPath)

	updatedEnv := append([]string{"PATH=" + updatedPath, "GOW_GO_CALLED_BY=" + string(command)}, os.Environ()...)

	return updatedEnv, updatedPath, func() {
		os.Setenv("PATH", strings.ReplaceAll(os.Getenv("PATH"), tmpDir+string(os.PathListSeparator), ""))
		os.RemoveAll(tmpDir)
	}, nil
}
