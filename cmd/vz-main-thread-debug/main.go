package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/mholt/archives"
	"github.com/nxadm/tail"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/ext/osx"
	"github.com/walteh/ec1/pkg/logging"
)

const (
	logFile                          = "/tmp/proc-demo/proc-demo.log"
	vmlinuzFile                      = "/tmp/proc-demo/harpoon_vmlinux_arm64"
	CHILD_STARTED                    = "CHILD_STARTED"
	CHILD_CONFIG_CREATED             = "CHILD_CONFIG_CREATED"
	CHILD_BOOTLOADER_CREATED         = "CHILD_BOOTLOADER_CREATED"
	CHILD_CREATED                    = "CHILD_CREATED"
	CHILD_CONFIG_VALIDATED           = "CHILD_CONFIG_VALIDATED"
	CHILD_CONFIG_VALIDATION_COMPLETE = "CHILD_CONFIG_VALIDATION_COMPLETE"
)

var executable string

func main() {

}

func init() {
	runtime.LockOSThread()
	realMain()
}

func realMain() {

	var arga, argb string
	if len(os.Args) < 3 {
		panic("call requires at least three arguments")
	}

	arga = os.Args[1]
	argb = os.Args[2]

	e, err := os.Executable()
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("getting executable: %w", err))
		os.Exit(1)
	}
	executable = e

	if arga == "START" {
		err = runStart(argb)
		if err != nil {
			fmt.Printf("main_error=%v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	err = runSwitch(arga, argb)
	if err != nil {
		fmt.Printf("main_error=%v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func runSwitch(arg string, argb string) error {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigchan
		for _, cleanup := range cleanups {
			go cleanup()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	defer func() {
		for _, cleanup := range cleanups {
			go cleanup()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	log("running switch with arg: %s, argb: %s", arg, argb)

	arg = strings.ToUpper(arg)
	var err error
	switch arg {
	case "START":
		panic("START should have been handled above")
	case "CHILD":
		if argb == "" {
			log("CHILD requires another argument")
			panic("CHILD requires another argument")
		}
		// run in this process as a child
		err = runChild(argb)
	case "CMD-EXEC":
		// run as orphaner ()
		err = runChildInCmdExec()
	case "BASH-C":
		err = runChildInCmdExecBashC()
	case "POSIX-SPAWN":
		err = runChildInCmdExecPosixSpawn()
	default:
		err = fmt.Errorf("invalid argument: %s", arg)
	}

	if err != nil {
		log("error in runSwitch: %s %s: %v\n", arg, argb, err)
		return err
	}

	return nil
}

func runChildInCmdExec() error {

	log("running child in cmd exec")

	cmd := exec.Command(executable, "CHILD", "CMD-EXEC")
	err := cmd.Start()
	if err != nil {
		errw := errors.Wrap(err, "starting child")
		log("err running child: %v\n", errw)
		return errw
	}

	return cmd.Wait()
}

func runChildInCmdExecBashC() error {

	cmd := exec.Command("bash", "-c", fmt.Sprintf("%s CHILD BASH-C", executable))
	cmd.Env = append(os.Environ(), "EC1_INITIALIZED=true")
	err := cmd.Start()
	if err != nil {
		errw := errors.Wrap(err, "starting child")
		log("err running child: %v\n", errw)
		return errw
	}

	return cmd.Wait()
}

func runChildInCmdExecPosixSpawn() error {

	_, err := Spawn(executable, []string{"CHILD", "POSIX-SPAWN"})
	if err != nil {
		errw := errors.Wrap(err, "starting child")
		log("err running child: %v\n", errw)
		return errw
	}

	return nil
}

var cleanups []func()
var cleanupsmu sync.Mutex = sync.Mutex{}

func addCleanup(cleanup func()) {
	cleanupsmu.Lock()
	defer cleanupsmu.Unlock()
	cleanups = append(cleanups, cleanup)
}

func runChild(caller string) error {

	bootloader, err := vz.NewLinuxBootLoader(vmlinuzFile)
	if err != nil {
		log("child bootloader error: %v\n", err)
		return err
	}

	log(CHILD_BOOTLOADER_CREATED)

	config, err := vz.NewVirtualMachineConfiguration(bootloader, 1, 64*1024*1024)
	if err != nil {
		log("child config error: %v\n", err)
		return err
	}

	log(CHILD_CONFIG_CREATED)

	ok, err := config.Validate()
	if err != nil {
		log("child config validate error: %v\n", err)
		return err
	}

	log(CHILD_CONFIG_VALIDATION_COMPLETE)

	if !ok {
		log("child config validate error: config is not valid\n")
		return fmt.Errorf("config is not valid")
	}

	log(CHILD_CONFIG_VALIDATED)

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		log("orphan_vm_error=%v\n", err)
		return err
	}

	log(CHILD_CREATED)
	//

	addCleanup(func() {
		err = vm.Stop()
		if err != nil {
			log("orphan_vm_stop_error=%v\n", err)
		}
	})

	err = vm.Start()
	if err != nil {
		log("orphan_vm_start_error=%v\n", err)
		return err
	}

	log(CHILD_STARTED)

	return nil
}

func runStart(mode string) error {

	ctx := context.Background()

	os.RemoveAll(filepath.Dir(logFile))
	os.MkdirAll(filepath.Dir(vmlinuzFile), 0755)

	// create the log file
	f, err := os.Create(logFile)
	if err != nil {
		return errors.Errorf("creating log file: %w", err)
	}
	f.Close()

	go func() {
		t, err := tail.TailFile(logFile, tail.Config{Follow: true})
		if err != nil {
			slog.ErrorContext(ctx, "error tailing log file", "error", err)
			return
		}
		for line := range t.Lines {
			fmt.Fprintf(logging.GetDefaultLogWriter(), "%s\n", line.Text)
		}
	}()

	log("running start with mode: %s", mode)

	xzr, err := (archives.Xz{}).OpenReader(bytes.NewReader(harpoon_vmlinux_arm64.BinaryXZ))
	if err != nil {
		return errors.Errorf("creating xz reader: %w", err)
	}

	err = osx.WriteFromReaderToFilePaths(ctx, xzr, vmlinuzFile)
	if err != nil {
		return errors.Errorf("writing xz reader to file: %w", err)
	}

	go func() {
		err = runSwitch(mode, "START")
		if err != nil {
			log("err running switch: %v\n", err)
		}
	}()
	waiters := []string{
		CHILD_STARTED,
		CHILD_CONFIG_CREATED,
		CHILD_BOOTLOADER_CREATED,
		CHILD_CREATED,
		CHILD_CONFIG_VALIDATED,
		CHILD_CONFIG_VALIDATION_COMPLETE,
	}

	group := sync.WaitGroup{}
	group.Add(len(waiters))
	type result struct {
		waiter string
		err    error
	}
	results := make(chan result, len(waiters))

	for _, waiter := range waiters {
		go func(waiter string) {
			defer group.Done()
			_, err := waitOnLogWithTimeout(waiter, 5*time.Second)
			results <- result{waiter: waiter, err: err}
		}(waiter)
	}

	go func() {
		group.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			log("error waiting for %s: %v", result.waiter, result.err)
			return result.err
		}
	}

	return nil
}

var logmu sync.Mutex = sync.Mutex{}

func log(format string, a ...any) {
	logmu.Lock()
	defer logmu.Unlock()

	lfile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer lfile.Close()

	uri := logging.GetCurrentCallerURIOffset(1)

	format = strings.ReplaceAll(format, "|", "---")
	format += "\n"

	fmt.Fprintf(lfile, "[pid=%d,ppid=%d] [func=%s] [line=%d] | ", os.Getpid(), os.Getppid(), uri.Function, uri.Line)
	fmt.Fprintf(lfile, format, a...)
}

func parseLog(key string) (string, error) {

	lines, err := os.ReadFile(logFile)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(lines), "\n") {
		if strings.Contains(line, key) {
			return line, nil
		}
	}

	return "", fmt.Errorf("key not found")
}

func waitOnLogWithTimeout(key string, timeout time.Duration) (string, error) {
	start := time.Now()
	for {
		pid, err := parseLog(key)
		if err == nil {
			return pid, nil
		}

		if time.Since(start) > timeout {
			return "", fmt.Errorf("timeout waiting for %s", key)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
