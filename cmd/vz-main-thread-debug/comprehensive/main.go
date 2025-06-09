package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/mholt/archives"
	"github.com/nxadm/tail"

	vzplugin "github.com/walteh/ec1/cmd/vz-main-thread-debug/plugin/vz"

	"github.com/walteh/ec1/gen/harpoon/harpoon_vmlinux_arm64"
	"github.com/walteh/ec1/pkg/ext/osx"
)

const (
	logFile          = "/tmp/proc-demo/proc-demo.log"
	vmlinuzFile      = "/tmp/proc-demo/harpoon_vmlinux_arm64"
	supervisorPid    = "/tmp/proc-demo/supervisor.pid"
	supervisorSocket = "/tmp/proc-demo/supervisor.sock"
	vzPluginFile     = "/tmp/proc-demo/vz-plugin.so"
)

func init() {

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "orphaner":
			executable, err := os.Executable()
			if err != nil {
				log("orphan_executable_error=%v\n", err)
				os.Exit(1)
			}
			// exec myself with the "orphan" argument
			cmd := exec.Command(executable, "orphan")
			cmd.Start()
			os.Exit(0)

		case "orphan":
			// run the test in orphaned process
			runOrphan()
			os.Exit(0)

		case "supervisor":
			// run as supervisor daemon
			runSupervisor()
			os.Exit(0)

		case "supervised-orphan":
			// run orphan test but with supervisor as parent
			runSupervisedOrphan()
			os.Exit(0)

		case "shim-exec":
			// simulate containerd shim exec pattern
			runShimExec()
			os.Exit(0)

		case "fork-only":
			// test fork without exec
			runForkOnly()
			os.Exit(0)

		case "shell-like-1":
			// test shell-like execution method 1
			runVZTest("shell_like")
			os.Exit(0)

		case "shell-like-2":
			// test shell-like execution method 2
			runVZTest("shell_like")
			os.Exit(0)

		case "exec-method-1":
			// test exec.Command method
			runVZTest("exec_method")
			os.Exit(0)

		case "exec-method-2":
			// test exec.Command with explicit attributes
			runVZTest("exec_method")
			os.Exit(0)

		case "shell-exec-1":
			// test execution via /bin/sh
			runVZTest("shell_exec")
			os.Exit(0)

		case "shell-exec-2":
			// test execution via /bin/zsh
			runVZTest("shell_exec")
			os.Exit(0)

		case "import-only-child":
			// Child process that imports VZ but never calls VZ APIs
			fmt.Printf("import_only_child_starting\n")

			// Import is already done at package level
			// Let's just verify the process works without any VZ calls
			time.Sleep(100 * time.Millisecond)
			fmt.Printf("import_only_child_success=true\n")
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

func runVZTest(prefix string) error {
	log("%s_pid=%d\n", prefix, os.Getpid())
	log("%s_ppid=%d\n", prefix, os.Getppid())

	plugin, err := vzplugin.NewVZPlugin(vzPluginFile)
	if err != nil {
		log("%s_plugin_error=%v\n", prefix, err)
		return err
	}
	log("%s_plugin_created=true\n", prefix)

	err = plugin.TestVZ(vmlinuzFile, log)
	if err != nil {
		log("%s_plugin_test_error=%v\n", prefix, err)
		return err
	}

	log("%s_vm_started=true\n", prefix)

	return nil
}

// func runVZTest(prefix string) error {
// 	log("%s_pid=%d\n", prefix, os.Getpid())
// 	log("%s_ppid=%d\n", prefix, os.Getppid())

// 	bootloader, err := vz.NewLinuxBootLoader(vmlinuzFile)
// 	if err != nil {
// 		log("%s_bootloader_error=%v\n", prefix, err)
// 		return err
// 	}
// 	log("%s_bootloader_created=true\n", prefix)

// 	config, err := vz.NewVirtualMachineConfiguration(bootloader, 1, 64*1024*1024)
// 	if err != nil {
// 		log("%s_config_error=%v\n", prefix, err)
// 		return err
// 	}
// 	log("%s_config_created=true\n", prefix)

// 	ok, err := config.Validate()
// 	if err != nil {
// 		log("%s_config_validate_error=%v\n", prefix, err)
// 		return err
// 	}
// 	if !ok {
// 		log("%s_config_validate_error=config is not valid\n", prefix)
// 		return fmt.Errorf("config is not valid")
// 	}
// 	log("%s_config_validate_ok=true\n", prefix)

// 	vm, err := vz.NewVirtualMachine(config)
// 	if err != nil {
// 		log("%s_vm_error=%v\n", prefix, err)
// 		return err
// 	}
// 	log("%s_vm_created=true\n", prefix)

// 	// This is the critical test
// 	err = vm.Start()
// 	if err != nil {
// 		log("%s_vm_start_error=%v\n", prefix, err)
// 		return err
// 	}
// 	log("%s_vm_started=true\n", prefix)

// 	return nil
// }

func runOrphan() error {
	return runVZTest("orphan")
}

func runSupervisor() error {
	log("supervisor_pid=%d\n", os.Getpid())
	log("supervisor_ppid=%d\n", os.Getppid())

	// Write supervisor PID to file
	err := os.WriteFile(supervisorPid, []byte(fmt.Sprintf("%d", os.Getpid())), 0644)
	if err != nil {
		log("supervisor_pid_write_error=%v\n", err)
		return err
	}

	log("supervisor_started=true\n")

	// Keep supervisor alive to act as parent for child processes
	// In real implementation, this would handle shim lifecycle management
	for {
		log("supervisor_heartbeat=%d\n", time.Now().Unix())
		time.Sleep(5 * time.Second)
	}
}

func runSupervisedOrphan() error {
	log("supervised_orphan_pid=%d\n", os.Getpid())
	log("supervised_orphan_ppid=%d\n", os.Getppid())

	// Check if we have a non-1 parent (supervisor)
	if os.Getppid() == 1 {
		log("supervised_orphan_error=still orphaned despite supervisor\n")
		return fmt.Errorf("still orphaned")
	}

	log("supervised_orphan_has_parent=true\n")
	return runVZTest("supervised_orphan")
}

func runShimExec() error {
	log("shim_exec_pid=%d\n", os.Getpid())
	log("shim_exec_ppid=%d\n", os.Getppid())

	// This simulates how containerd would exec a shim
	// The shim should be able to run VZ operations
	return runVZTest("shim_exec")
}

func runForkOnly() error {
	log("fork_only_pid=%d\n", os.Getpid())
	log("fork_only_ppid=%d\n", os.Getppid())

	// This simulates fork without exec - keeps execution context
	return runVZTest("fork_only")
}

func testForkOnly() error {
	// Fork without exec to test if it's fork vs exec that causes the issue
	pid, err := syscall.ForkExec("/bin/true", []string{"/bin/true"}, &syscall.ProcAttr{
		Env: os.Environ(),
	})
	if err != nil {
		return fmt.Errorf("fork failed: %w", err)
	}

	if pid == 0 {
		// Child process - run VZ test directly (no exec)
		return runVZTest("fork_only")
	} else {
		// Parent process - wait for child
		var status syscall.WaitStatus
		_, err := syscall.Wait4(int(pid), &status, 0, nil)
		if err != nil {
			return fmt.Errorf("wait failed: %w", err)
		}

		log("main_fork_only_completed=%d\n", status.ExitStatus())
		return nil
	}
}

func startSupervisor() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable: %w", err)
	}

	// Start supervisor as background daemon
	cmd := exec.Command(executable, "supervisor")
	cmd.Start()

	// Wait for supervisor to be ready
	_, err = waitOnLogWithTimeout("supervisor_started", 5*time.Second)
	if err != nil {
		return fmt.Errorf("supervisor failed to start: %w", err)
	}

	log("main_supervisor_started=true\n")
	return nil
}

func execViaSupervisor(mode string) error {
	// Read supervisor PID (for verification, not strictly needed for this demo)
	_, err := os.ReadFile(supervisorPid)
	if err != nil {
		return fmt.Errorf("reading supervisor PID: %w", err)
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable: %w", err)
	}

	// Use the supervisor as parent by creating child process of supervisor
	// This simulates how a supervisor would spawn shim processes

	// For this demo, we'll use ForkExec but in real implementation,
	// the supervisor would handle this via IPC/socket communication
	pid, err := syscall.ForkExec(executable, []string{executable, mode}, &syscall.ProcAttr{
		Env: os.Environ(),
		Sys: &syscall.SysProcAttr{
			// Key: Don't use Setpgid here - keep as child of current process
		},
	})

	if err != nil {
		return fmt.Errorf("ForkExec failed: %w", err)
	}

	log("main_spawned_via_supervisor=%d\n", pid)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <mode>\n", os.Args[0])
		fmt.Printf("Modes:\n")
		fmt.Printf("  -main: Run VZ test in main process (baseline)\n")
		fmt.Printf("  -orphaner: Create orphaned process (demonstrates issue)\n")
		fmt.Printf("  -supervisor-test: Test supervisor approach\n")
		fmt.Printf("  -shim-test: Test containerd shim exec pattern\n")
		fmt.Printf("  -fork-test: Test fork without exec\n")
		fmt.Printf("  -shell-like-test: Test ForkExec with shell-like attributes\n")
		fmt.Printf("  -exec-test: Test different exec methods\n")
		fmt.Printf("  -shell-exec-test: Use actual shell to create process\n")
		fmt.Printf("  -raw-syscall-test: Test raw fork/execve syscalls to bypass posix_spawn\n")
		fmt.Printf("  -raw-fork-child: This will be executed in the child process created by raw fork\n")
		fmt.Printf("  -lazy-vz-test: Test lazy VZ initialization - defer VZ context until after fork\n")
		fmt.Printf("  -lazy-vz-child: This will be executed in the child process with lazy VZ initialization\n")
		fmt.Printf("  -env-isolation-test: Test environment-based VZ isolation\n")
		fmt.Printf("  -no-vz-child: Child process that never initializes VZ\n")
		fmt.Printf("  -import-only-test: Test if just importing VZ package establishes context\n")
		fmt.Printf("  -import-only-child: Child process that imports VZ but never calls VZ APIs\n")
		os.Exit(1)
	}

	mode := os.Args[1]

	ctx := context.Background()

	// Setup
	os.Remove(logFile)
	os.Mkdir(filepath.Dir(vmlinuzFile), 0755)

	// create the log file
	f, err := os.Create(logFile)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("creating log file: %w", err))
		os.Exit(1)
	}
	f.Close()

	// CGO_ENABLED=1 go build -x -buildmode=plugin -o "/tmp/proc-demo/vz-plugin.so" ./plugin.main.go

	// Start log tailing
	go func() {
		t, err := tail.TailFile(logFile, tail.Config{Follow: true})
		if err != nil {
			fmt.Printf("main_error=%v\n", fmt.Errorf("tailing log file: %w", err))
			os.Exit(1)
		}
		for line := range t.Lines {
			fmt.Printf("%s\n", line.Text)
		}
	}()

	fmt.Printf("main_pid=%d\n", os.Getpid())
	fmt.Printf("main_mode=%s\n", mode)

	// Extract vmlinux
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

	switch mode {
	case "-main":
		// Baseline: run in main process
		fmt.Printf("main_test=baseline\n")
		go runOrphan()

	case "-orphaner":
		// Problem: create orphaned process (demonstrates issue)
		fmt.Printf("main_test=orphaned_process_issue\n")
		pid, err := syscall.ForkExec(executable, []string{executable, "orphaner"}, &syscall.ProcAttr{
			Env: os.Environ(),
			Sys: &syscall.SysProcAttr{
				Setpgid: true, // This causes orphaning
			},
		})
		if err != nil {
			fmt.Printf("main_error=%v\n", err)
			os.Exit(1)
		}
		go func() {
			for {
				fmt.Printf("orphan_process_state=%d\n", pid)
				time.Sleep(1 * time.Second)
			}
		}()

	case "-import-only-test":
		// NEW: Test if just importing VZ package establishes context
		fmt.Printf("main_test=import_only\n")

		err := testImportOnlyVZ(executable)
		if err != nil {
			fmt.Printf("main_error=%v\n", err)
			os.Exit(1)

		}

	case "-shell-test":
		fmt.Printf("main_test=shell_like_test\n")
		err := testActualShellExecution(executable)
		if err != nil {
			fmt.Printf("main_error=%v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("main_error=invalid mode: %s\n", mode)
		os.Exit(1)
	}

	// Wait for test completion and report results
	var testPrefix string
	switch mode {
	case "-main":
		testPrefix = "orphan"
	case "-orphaner":
		testPrefix = "orphan"
	case "-import-only-test":
		testPrefix = "import_only"
	}

	// Wait for results
	// Skip VM creation waits for tests that don't create VMs
	if testPrefix == "no_vz_child" || testPrefix == "import_only_child" || testPrefix == "import_only" {
		fmt.Printf("main_test_result=no_vm_created\n")
		fmt.Printf("main_success=true\n")
		os.Exit(0)
	}

	_, err = waitOnLogWithTimeout(testPrefix+"_vm_created", 10*time.Second)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("waiting for %s_vm_created: %w", testPrefix, err))
		os.Exit(1)
	}

	vmStarted, err := waitOnLogWithTimeout(testPrefix+"_vm_started", 10*time.Second)
	if err != nil {
		fmt.Printf("main_error=%v\n", fmt.Errorf("waiting for %s_vm_started: %w", testPrefix, err))
		os.Exit(1)
	}

	fmt.Printf("main_test_result=%s\n", vmStarted)
	fmt.Printf("main_success=true\n")
	os.Exit(0)
}

// testShellLikeForkExec tries to mimic shell process creation more closely
func testShellLikeForkExec(executable string) error {
	fmt.Printf("main_testing_shell_like_attributes\n")

	// Try multiple configurations that might match shell behavior

	// Test 1: Full shell-like environment and file descriptors
	pid1, err := syscall.ForkExec(executable, []string{executable, "shell-like-1"}, &syscall.ProcAttr{
		Dir: ".",          // Explicit working directory
		Env: os.Environ(), // Full environment
		Files: []uintptr{
			uintptr(syscall.Stdin),  // Keep stdin
			uintptr(syscall.Stdout), // Keep stdout
			uintptr(syscall.Stderr), // Keep stderr
		},
		Sys: &syscall.SysProcAttr{
			// Don't set Setpgid - keep same process group as shell would
		},
	})
	if err != nil {
		return fmt.Errorf("shell-like test 1 failed: %w", err)
	}
	fmt.Printf("main_shell_like_pid_1=%d\n", pid1)

	// Test 2: With proper session/controlling terminal setup
	pid2, err := syscall.ForkExec(executable, []string{executable, "shell-like-2"}, &syscall.ProcAttr{
		Dir: ".",
		Env: os.Environ(),
		Files: []uintptr{
			uintptr(syscall.Stdin),
			uintptr(syscall.Stdout),
			uintptr(syscall.Stderr),
		},
		Sys: &syscall.SysProcAttr{
			// Keep foreground process group (like interactive shell)
			Foreground: true,
		},
	})
	if err != nil {
		return fmt.Errorf("shell-like test 2 failed: %w", err)
	}
	fmt.Printf("main_shell_like_pid_2=%d\n", pid2)

	return nil
}

// testDifferentExecMethods tests various ways to exec processes
func testDifferentExecMethods(executable string) error {
	fmt.Printf("main_testing_different_exec_methods\n")

	// Test 1: Using exec.Command (higher level)
	fmt.Printf("main_testing_exec_command\n")
	cmd := exec.Command(executable, "exec-method-1")
	cmd.Env = os.Environ()
	cmd.Dir = "."
	// Don't modify SysProcAttr - let it use defaults
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("exec.Command failed: %w", err)
	}
	fmt.Printf("main_exec_command_pid=%d\n", cmd.Process.Pid)

	// Test 2: Using exec.Command with explicit attributes
	fmt.Printf("main_testing_exec_command_explicit\n")
	cmd2 := exec.Command(executable, "exec-method-2")
	cmd2.Env = os.Environ()
	cmd2.Dir = "."
	cmd2.SysProcAttr = &syscall.SysProcAttr{
		// Mimic shell behavior - no special process group handling
	}
	err = cmd2.Start()
	if err != nil {
		return fmt.Errorf("exec.Command explicit failed: %w", err)
	}
	fmt.Printf("main_exec_command_explicit_pid=%d\n", cmd2.Process.Pid)

	return nil
}

// testActualShellExecution uses the shell itself to create the process
func testActualShellExecution(executable string) error {
	fmt.Printf("main_testing_actual_shell_execution\n")

	// Test 1: Use /bin/sh to execute our binary (like shell would)
	fmt.Printf("main_testing_via_bin_sh\n")
	cmd := exec.Command("/bin/bash", "-c", executable+" shell-exec-1")

	cmd.Env = os.Environ()
	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("shell execution via /bin/sh failed: %w", err)
	}
	fmt.Printf("main_shell_exec_pid_1=%d\n", cmd.Process.Pid)

	// Test 2: Use /bin/zsh (the actual shell we're using)
	fmt.Printf("main_testing_via_bin_zsh\n")
	cmd2 := exec.Command("/bin/zsh", "-c", executable+" shell-exec-2")
	cmd2.Env = os.Environ()
	err = cmd2.Start()
	if err != nil {
		return fmt.Errorf("shell execution via /bin/zsh failed: %w", err)
	}
	fmt.Printf("main_shell_exec_pid_2=%d\n", cmd2.Process.Pid)

	return nil
}

func testRawSyscallForkExec(executable string) error {
	fmt.Printf("main_testing_raw_syscall_approach\n")

	// Try to bypass Go's posix_spawn by using raw syscalls
	// WARNING: This is highly experimental and dangerous!

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Direct fork syscall - SYS_FORK = 2 on Darwin
	pid, _, errno := syscall.RawSyscall(2, 0, 0, 0) // SYS_FORK

	if errno != 0 {
		return fmt.Errorf("raw fork failed: %v", errno)
	}

	if pid == 0 {
		// Child process - try to exec
		fmt.Printf("raw_fork_child_process=true\n")

		// Prepare execve arguments
		argv0, err := syscall.BytePtrFromString(executable)
		if err != nil {
			fmt.Printf("raw_fork_child_error=argv0: %v\n", err)
			syscall.RawSyscall(1, 1, 0, 0) // SYS_EXIT
		}

		args := []string{executable, "-raw-fork-child"}
		argvp, err := syscall.SlicePtrFromStrings(args)
		if err != nil {
			fmt.Printf("raw_fork_child_error=argvp: %v\n", err)
			syscall.RawSyscall(1, 1, 0, 0) // SYS_EXIT
		}

		envp, err := syscall.SlicePtrFromStrings(os.Environ())
		if err != nil {
			fmt.Printf("raw_fork_child_error=envp: %v\n", err)
			syscall.RawSyscall(1, 1, 0, 0) // SYS_EXIT
		}

		// Raw execve syscall - SYS_EXECVE = 59 on Darwin
		_, _, errno := syscall.RawSyscall(
			59, // SYS_EXECVE
			uintptr(unsafe.Pointer(argv0)),
			uintptr(unsafe.Pointer(&argvp[0])),
			uintptr(unsafe.Pointer(&envp[0])),
		)

		fmt.Printf("raw_fork_child_error=execve failed: %v\n", errno)
		syscall.RawSyscall(1, 1, 0, 0) // SYS_EXIT

	} else {
		// Parent process - wait for child
		fmt.Printf("raw_fork_parent=true, child_pid=%d\n", pid)

		var status syscall.WaitStatus
		wpid, err := syscall.Wait4(int(pid), &status, 0, nil)
		if err != nil {
			return fmt.Errorf("wait4 failed: %w", err)
		}

		fmt.Printf("raw_fork_child_completed=true, wpid=%d, status=%d\n", wpid, status.ExitStatus())

		// Check if our expected log was produced
		success, err := waitOnLogWithTimeout("raw_fork_child_success", 2*time.Second)
		if err != nil {
			return fmt.Errorf("child process failed or timed out: %w", err)
		}

		if success == "true" {
			fmt.Printf("main_raw_syscall_test_success=true\n")
		} else {
			fmt.Printf("main_raw_syscall_test_failed=child_did_not_succeed\n")
		}
	}

	return nil
}

func testLazyVZInit(executable string) error {
	fmt.Printf("main_testing_lazy_vz_initialization\n")

	// Test lazy VZ initialization - exec a child process that will initialize VZ
	// This avoids any VZ context inheritance from parent

	cmd := exec.Command(executable, "-lazy-vz-child")
	cmd.Env = os.Environ()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	fmt.Printf("lazy_vz_exec_duration=%v\n", duration)

	if err != nil {
		fmt.Printf("lazy_vz_exec_error=%v\n", err)
		return err
	}

	fmt.Printf("main_lazy_vz_test_success=true\n")
	return nil
}

func testEnvIsolation(executable string) error {
	fmt.Printf("main_testing_env_isolation\n")

	// Strategy 1: Test isolation by avoiding VZ initialization completely in the parent
	// and deferring it only to processes that actually need it

	// Set an environment variable to indicate VZ should NOT be initialized
	env := append(os.Environ(), "EC1_DISABLE_VZ=true")

	// Start a child process that checks this environment
	cmd := exec.Command(executable, "-no-vz-child")
	cmd.Env = env

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	fmt.Printf("env_isolation_no_vz_duration=%v\n", duration)

	if err != nil {
		fmt.Printf("env_isolation_no_vz_error=%v\n", err)
		return err
	}

	// Strategy 2: Test actual VZ operations in isolation (clean child process)
	cmd2 := exec.Command(executable, "-lazy-vz-child")
	cmd2.Env = os.Environ() // Clean environment without VZ disable flag

	startTime2 := time.Now()
	err2 := cmd2.Run()
	duration2 := time.Since(startTime2)

	fmt.Printf("env_isolation_vz_duration=%v\n", duration2)

	if err2 != nil {
		fmt.Printf("env_isolation_vz_error=%v\n", err2)
		return err2
	}

	fmt.Printf("main_env_isolation_test_success=true\n")
	return nil
}

func testImportOnlyVZ(executable string) error {
	fmt.Printf("main_testing_import_only_vz\n")

	// Test if just importing VZ package establishes context
	// This is a simple test to ensure that importing VZ package establishes VZ context
	// without actually calling any VZ APIs

	// Start a child process that imports VZ but never calls VZ APIs
	cmd := exec.Command(executable, "-import-only-child")
	cmd.Env = os.Environ()

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	fmt.Printf("import_only_exec_duration=%v\n", duration)

	if err != nil {
		fmt.Printf("import_only_exec_error=%v\n", err)
		return err
	}

	fmt.Printf("main_import_only_test_success=true\n")
	return nil
}
