package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
)

var supportedPaths = []string{
	"cat",
	"grep",
}

var (
	buildPathProxy = flag.Bool("build-path-proxy", false, "build path proxy scripts")
	useGoRun       = flag.Bool("use-go-run", false, "use go run")
	proxyDir       = flag.String("proxy-dir", "/tmp/macos-linux-shim", "proxy dir")

	logfileName = flag.String("logfile", "./macos-linux-shim.log", "logfile")
)

func removeSelfFromPath() string {
	path := os.Getenv("PATH")
	paths := strings.Split(path, ":")
	for _, p := range paths {
		if !strings.Contains(p, "macos-linux-shim") {
			paths = append(paths, p)
		}
	}
	return strings.Join(paths, ":")
}

var logFile *os.File

func init() {
	os.Setenv("PATH", removeSelfFromPath())
}

func main() {
	flag.Parse()

	var err error

	fmt.Println("PROXY INVOKED", os.Args)
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "usage: shim <path> [args...]")
		os.Exit(1)
	}

	logFile, err = os.OpenFile(*logfileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logFile.WriteString(fmt.Sprintf("error: %v\n", err))
		os.Exit(1)
	}

	realStdout := os.Stdout
	realStderr := os.Stderr

	os.Stdout = logFile
	os.Stderr = logFile

	defer logFile.Close()

	logFile.WriteString(fmt.Sprintf("PROXY INVOKED %v\n", os.Args))

	if *buildPathProxy {
		// create a new path
		uCache, err := os.UserCacheDir()
		if err != nil {
			logFile.WriteString(fmt.Sprintf("error: %v\n", err))
			panic(err)
		}
		if *proxyDir == "" {
			*proxyDir = filepath.Join(uCache, "macos-linux-shim", "path-proxy")
		}
		os.RemoveAll(*proxyDir)
		os.MkdirAll(*proxyDir, 0755)
		for _, path := range supportedPaths {
			realPath, err := exec.LookPath(path)
			if err != nil {
				logFile.WriteString(fmt.Sprintf("error: %v\n", err))
				panic(err)
			}
			scriptPath := filepath.Join(*proxyDir, path)
			if *useGoRun {

				_, file, _, _ := runtime.Caller(0)
				dir := filepath.Dir(filepath.Dir(filepath.Dir(file)))
				script := fmt.Sprintf(`#!/bin/bash
if [ -n "$IGNORE_PATH_PROXY" ]; then
	exec %[1]s "$@"
fi
cd %[2]s && go run %[3]s %[4]s "$@" < /dev/stdin > /dev/stdout
`, realPath, dir, "./cmd/macos-linux-shim", path)
				os.WriteFile(scriptPath, []byte(script), 0755)
			} else {
				script := fmt.Sprintf(`#!/bin/bash
if [ -n "$IGNORE_PATH_PROXY" ]; then
	exec %[1]s "$@" 
fi	
%[2]s "$@" < /dev/stdin > /dev/stdout
`, realPath, path)
				os.WriteFile(scriptPath, []byte(script), 0755)
			}
		}
		fmt.Println(*proxyDir)
		return
	}

	// first arg is the linux path we need to emulate
	if len(os.Args) < 2 {
		logFile.WriteString(fmt.Sprintf("usage: shim <path> [args...]\n"))
		logFile.Close()
		panic("usage: shim <path> [args...]")
	}

	mapd, err := commandMap(os.Args[1:])
	if err != nil {
		logFile.WriteString(fmt.Sprintf("fallback: %v %v\n", os.Args[1], os.Args[2:]))
		// fallback: print usage or error
		cmd := exec.Command(os.Args[1], os.Args[2:]...)
		cmd.Stdout = realStdout
		cmd.Stderr = realStderr
		err = cmd.Run()
		if err != nil {
			logFile.WriteString(fmt.Sprintf("error: %v\n", err))
			logFile.Close()
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			panic(err)
		}
	}
	fmt.Fprintf(realStdout, "%s", mapd)
	fmt.Fprintf(realStderr, "%s", "")
}

func commandMap(args []string) (string, error) {
	logFile.WriteString(fmt.Sprintf("commandMap: %v\n", args))

	switch args[0] {
	case "cat":
		switch args[1] {
		case "/proc/cpuinfo":
			return procCpuInfo(), nil
		case "/proc/meminfo":
			return procMemInfo(), nil
		}
	case "grep":
		switch args[1] {
		case "microcode":
			switch args[2] {
			case "/proc/cpuinfo":
				return "hello", nil
			case "/proc/meminfo":
				return "hello", nil
			}
		}
	}
	return "", fmt.Errorf("command not found")
}

func grep(filedata, pattern string) (string, error) {
	logFile.WriteString(fmt.Sprintf("grep %s %s\n", pattern, filedata))
	tmpFile, err := os.CreateTemp("", "macos-linux-shim-grep-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	logFile.WriteString(fmt.Sprintf("write to tmp file %s\n", tmpFile.Name()))

	tmpFile.WriteString(filedata)
	tmpFile.Close()

	var buf bytes.Buffer
	var ebuf bytes.Buffer

	logFile.WriteString(fmt.Sprintf("grep %s %s\n", pattern, tmpFile.Name()))

	cmd := exec.Command("grep", pattern, tmpFile.Name())
	cmd.Stdout = &buf
	cmd.Stderr = &ebuf
	cmd.Env = append(os.Environ(), "IGNORE_PATH_PROXY=1")
	err = cmd.Run()
	if err != nil {
		logFile.WriteString(fmt.Sprintf("error: %v\n", err))
		logFile.WriteString(fmt.Sprintf("stdout: %s\n", buf.String()))
		logFile.WriteString(fmt.Sprintf("stderr: %s\n", ebuf.String()))
		return "", err
	}

	logFile.WriteString(fmt.Sprintf("grep %s %s:\n%s\n", pattern, tmpFile.Name(), buf.String()))
	return buf.String(), nil
}

func procCpuInfo() string {
	cpuCount := runtime.NumCPU()
	var buf bytes.Buffer
	for i := 0; i < cpuCount; i++ {
		buf.WriteString(fmt.Sprintf("processor\t: %d\n", i))
	}
	return buf.String()
}

func procMemInfo() string {
	// retrieve total memory in bytes
	totalBytes, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		// fallback to zero
		totalBytes = 0
	}
	totalKB := totalBytes / 1024
	freeKB := totalKB / 10
	availKB := totalKB * 8 / 10

	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("MemTotal:       %d kB\n", totalKB))
	buf.WriteString(fmt.Sprintf("MemFree:        %d kB\n", freeKB))
	buf.WriteString(fmt.Sprintf("MemAvailable:   %d kB\n", availKB))
	return buf.String()
}
