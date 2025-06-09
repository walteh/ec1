package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Code-Hex/vz/v3"
	"github.com/mholt/archives"

	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/ext/osx"
)

const (
	logFile     = "/tmp/proc-demo/proc-demo.log"
	vmlinuzFile = "/tmp/proc-demo/harpoon_vmlinux_arm64"
)

func init() {
	if len(os.Args) != 1 {
		if os.Args[1] == "orphaner" {

			executable, err := os.Executable()
			if err != nil {
				log("orphan_executable_error=%v\n", err)
				os.Exit(1)
			}

			// exec myself with the "orphan" argument
			cmd := exec.Command(executable, "orphan")
			cmd.Start()
			os.Exit(0)
		} else if os.Args[1] == "orphan" {
			// run the test
			runOrphan()
			os.Exit(0)
		}
	}
}

func log(format string, a ...any) {
	lfile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer lfile.Close()

	fmt.Fprintf(lfile, format, a...)
}

func parseLog(key string) (string, error) {

	lines, err := os.ReadFile(logFile)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(lines), "\n") {
		if strings.HasPrefix(line, key) {
			return strings.TrimPrefix(line, key+"="), nil
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

func runOrphan() error {

	log("orphan_pid=%d\n", os.Getpid())
	log("orphan_ppid=%d\n", os.Getppid())

	bootloader, err := vz.NewLinuxBootLoader(vmlinuzFile)
	if err != nil {
		log("orphan_bootloader_error=%v\n", err)
		return err
	}

	log("orphan_bootloader_created=true\n")

	config, err := vz.NewVirtualMachineConfiguration(bootloader, 1, 64*1024*1024)
	if err != nil {
		log("orphan_config_error=%v\n", err)
		return err
	}

	log("orphan_config_created=true\n")

	ok, err := config.Validate()
	if err != nil {
		log("orphan_config_validate_error=%v\n", err)
		return err
	}

	if !ok {
		log("orphan_config_validate_error=config is not valid\n")
		return fmt.Errorf("config is not valid")
	}

	log("orphan_config_validate_ok=true\n")

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		log("orphan_vm_error=%v\n", err)
		return err
	}

	log("orphan_vm_created=true\n")

	err = vm.Start()
	if err != nil {
		log("orphan_vm_start_error=%v\n", err)
		return err
	}

	log("orphan_vm_started=true\n")

	return nil
}

func main() {

	var mode string
	if os.Args[1] == "-main" {
		mode = "main"
	} else if os.Args[1] == "-orphan" {
		mode = "orphan"
	} else if os.Args[1] == "-orphaner" {
		mode = "orphaner"
	} else {
		fmt.Printf("main_error=invalid argument: %s\n", os.Args[1])
		os.Exit(1)
	}

	ctx := context.Background()

	os.RemoveAll(filepath.Dir(logFile))
	os.MkdirAll(filepath.Dir(vmlinuzFile), 0755)

	// create the log file
	f, err := os.Create(logFile)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("creating log file: %w", err))
		os.Exit(1)
	}
	f.Close()

	fmt.Printf("main_pid=%d\n", os.Getpid())

	xzr, err := (archives.Xz{}).OpenReader(bytes.NewReader(harpoon_vmlinux_arm64.BinaryXZ))
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("creating xz reader: %w", err))
		os.Exit(1)
	}

	err = osx.WriteFromReaderToFilePaths(ctx, xzr, vmlinuzFile)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("writing xz reader to file: %w", err))
		os.Exit(1)
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("getting executable: %w", err))
		os.Exit(1)
	}

	if mode == "main" {
		go runOrphan()
	} else {

		// runtime.LockOSThread()
		// defer runtime.UnlockOSThread()

		// libdispatch.DispatchMain()

		pid, err := syscall.ForkExec(executable, []string{mode}, &syscall.ProcAttr{
			Env: os.Environ(),
			Sys: &syscall.SysProcAttr{
				// Setpgid: true,
				// Set:  true,
				// Ptrace:     true,
				// Foreground: true,

				// Pgid: os.Getpid(),
			},
		})

		// cmd := exec.Command(executable, mode)
		// // cmd.SysProcAttr = &syscall.SysProcAttr{
		// // 	Setpgid: true,
		// // 	// Set:  true,
		// // 	// Ptrace:     true,
		// // 	Foreground: true,

		// // 	Pgid: os.Getpid(),
		// // }

		// err = cmd.Start()
		go func() {
			for {
				fmt.Printf("orphan_process_state=%d\n", pid)
				time.Sleep(1 * time.Second)
			}
		}()

		if err != nil {

			// dig deeper into the error
			fmt.Printf("main_error_type=%d\n", uintptr(any(err).(syscall.Errno)))
			os.Exit(1)
		}
	}

	orphanPid, err := waitOnLogWithTimeout("orphan_pid", 10*time.Second)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("waiting for orphan_pid: %w", err))
		os.Exit(1)
	}

	orphanCreated, err := waitOnLogWithTimeout("orphan_vm_created", 10*time.Second)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("waiting for orphan_vm_created: %w", err))
		os.Exit(1)
	}

	if orphanCreated != "true" {
		fmt.Printf("main_error=orphan_vm_created is not true\n")
		os.Exit(1)
	}

	orphanStarted, err := waitOnLogWithTimeout("orphan_vm_started", 10*time.Second)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("waiting for orphan_vm_started: %w", err))
		os.Exit(1)
	}

	if orphanStarted != "true" {
		fmt.Printf("main_error=orphan_vm_started is not true\n")
		os.Exit(1)
	}

	fmt.Printf("main_orphan_pid=%s\n", orphanPid)
	fmt.Printf("main_orphan_created=%s\n", orphanCreated)
	fmt.Printf("main_orphan_started=%s\n", orphanStarted)

	os.Exit(0)
}
