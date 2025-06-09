package containerd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	taskt "github.com/containerd/containerd/api/types/task"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

type managedProcess struct {
	spec    *specs.Process
	io      stdio
	console *os.File
	mu      sync.Mutex

	// VM-specific fields
	vm  *vmm.RunningVM[*vf.VirtualMachine]
	pid int

	waitblock  chan struct{}
	status     taskt.Status
	exitStatus uint32
	exitedAt   time.Time

	// For tracking the running command
	commandCtx    context.Context
	commandCancel context.CancelFunc
}

func (p *managedProcess) getConsoleL() *os.File {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.console
}

func (p *managedProcess) destroy() (retErr error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Cancel any running command
	if p.commandCancel != nil {
		p.commandCancel()
	}

	if err := p.io.Close(); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	if p.console != nil {
		if err := p.console.Close(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	if p.status != taskt.Status_STOPPED {
		p.status = taskt.Status_STOPPED
		p.exitedAt = time.Now()
		p.exitStatus = uint32(syscall.SIGKILL)
	}

	return
}

func (p *managedProcess) kill(signal syscall.Signal) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// For VM-based processes, we need to send the signal via the VM
	if p.vm != nil && p.pid > 0 {
		// Execute kill command in the VM
		ctx := context.Background()
		killCmd := fmt.Sprintf("kill -%d %d", int(signal), p.pid)
		_, _, _, err := p.vm.Exec(ctx, killCmd)
		return err
	}

	// If we have a command context, cancel it
	if p.commandCancel != nil {
		p.commandCancel()
	}

	return nil
}

func (p *managedProcess) setup(ctx context.Context, rootfs string, stdin string, stdout string, stderr string) error {
	var err error

	p.io, err = setupIO(ctx, stdin, stdout, stderr)
	if err != nil {
		return errors.Errorf("setting up IO: %w", err)
	}

	if len(p.spec.Args) <= 0 {
		// Default to shell if no args provided
		p.spec.Args = []string{"/bin/sh"}
	}

	return nil
}

func (p *managedProcess) start(vm *vmm.RunningVM[*vf.VirtualMachine]) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	slog.Info("managedProcess.start: Starting process in VM")

	// Wait for VM to be ready for exec
	slog.Info("managedProcess.start: Waiting for VM to be ready for exec")
	select {
	case <-p.vm.WaitOnVMReadyToExec():
		// VM is ready
		slog.Info("managedProcess.start: VM is ready for exec")
	case <-time.After(30 * time.Second):
		slog.Error("managedProcess.start: Timeout waiting for VM to be ready for exec")
		return fmt.Errorf("timeout waiting for VM to be ready for exec")
	}

	// // Build the command to execute
	var cmdParts []string

	// Handle working directory
	if p.spec.Cwd != "" {
		cmdParts = append(cmdParts, "cd", p.spec.Cwd, "&&")
	}

	// Handle environment variables
	for _, env := range p.spec.Env {
		cmdParts = append(cmdParts, "export", env, "&&")
	}

	// Add the actual command
	cmdParts = append(cmdParts, p.spec.Args...)

	command := strings.Join(cmdParts, " ")
	slog.Info("managedProcess.start: Executing command in VM", "command", command)

	// Create context for the command
	p.commandCtx, p.commandCancel = context.WithCancel(context.Background())

	p.status = taskt.Status_RUNNING

	// Execute the command in the VM
	go func() {
		defer func() {
			slog.Info("managedProcess.start: Command execution finished, updating process state")
			p.mu.Lock()
			p.status = taskt.Status_STOPPED
			p.exitedAt = time.Now()
			close(p.waitblock)
			p.mu.Unlock()
			slog.Info("managedProcess.start: Process state updated and waitblock closed")
		}()

		slog.Info("managedProcess.start: Calling vm.Exec")
		stdout, stderr, exitCode, err := p.vm.Exec(p.commandCtx, command)

		slog.Info("managedProcess.start: vm.Exec completed", "stdout_len", len(stdout), "stderr_len", len(stderr), "exit_code", string(exitCode), "error", err)

		// Write output to the configured I/O
		if len(stdout) > 0 {
			slog.Debug("managedProcess.start: Writing stdout", "length", len(stdout))
			p.io.stdout.Write(stdout)
		}
		if len(stderr) > 0 {
			slog.Debug("managedProcess.start: Writing stderr", "length", len(stderr))
			p.io.stderr.Write(stderr)
		}

		// Parse exit code
		if len(exitCode) > 0 {
			if code, parseErr := strconv.Atoi(strings.TrimSpace(string(exitCode))); parseErr == nil {
				slog.Info("managedProcess.start: Setting exit status", "exit_code", code)
				p.mu.Lock()
				p.exitStatus = uint32(code)
				p.mu.Unlock()
			} else {
				slog.Warn("managedProcess.start: Failed to parse exit code", "exit_code_raw", string(exitCode), "error", parseErr)
			}
		}

		if err != nil {
			slog.Error("managedProcess.start: Command execution failed", "error", err, "command", command)
			p.mu.Lock()
			p.exitStatus = 1
			p.mu.Unlock()
		}
	}()

	slog.Info("managedProcess.start: Process started successfully")
	return nil
}
