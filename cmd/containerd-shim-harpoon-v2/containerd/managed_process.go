package containerd

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	taskt "github.com/containerd/containerd/api/types/task"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"
)

type managedProcess struct {
	id      string
	execID  string
	spec    *specs.Process
	io      stdio
	console *os.File
	mu      sync.Mutex

	// VM-specific fields
	container *container
	pid       int

	// waitblock  chan struct{}
	// status     taskt.Status
	// exitStatus uint32
	// exitedAt   time.Time

	// For tracking the running command
	commandCtx    context.Context
	commandCancel context.CancelFunc

	runningCmd *signalRunner
}

func NewManagedProcess(execID string, container *container, spec *specs.Process, sio stdio) *managedProcess {
	id := execID
	if id == "" {
		id = "primary"
	}

	return &managedProcess{
		id:        id,
		execID:    execID,
		container: container,
		spec:      spec,
		io:        sio,
	}
}

type ManagedProcessState struct {
	Status   taskt.Status
	ExitedAt time.Time
	ExitCode int32
}

func (p *managedProcess) getStatus() ManagedProcessState {
	if p.runningCmd == nil {
		return ManagedProcessState{
			Status:   taskt.Status_CREATED,
			ExitedAt: time.Time{},
			ExitCode: 0,
		}
	}

	return ManagedProcessState{
		Status:   p.runningCmd.status,
		ExitedAt: p.runningCmd.exitedAt,
		ExitCode: p.runningCmd.exitCode,
	}
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

	if p.runningCmd != nil {
		if err := p.runningCmd.SendSignal(syscall.SIGKILL); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	if err := p.io.Close(); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	if p.console != nil {
		if err := p.console.Close(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	return
}

func (p *managedProcess) StartSignalRunner(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	guestService, err := p.container.vm.GuestService(ctx)
	if err != nil {
		return errors.Errorf("getting guest service: %w", err)
	}

	signalClient, err := guestService.RunSpecSignal(ctx)
	if err != nil {
		return errors.Errorf("running process in VM: %w", err)
	}

	p.runningCmd = &signalRunner{
		done:         make(chan struct{}),
		status:       taskt.Status_RUNNING,
		signalClient: signalClient,
		error:        nil,
		exitCode:     0,
	}

	return nil
}

func (p *managedProcess) SendSignalToRunningCmd(signal syscall.Signal) error {

	if p.runningCmd == nil {
		return errors.Errorf("no running command")
	}

	return p.runningCmd.SendSignal(signal)
}

// func (p *managedProcess) setup(ctx context.Context, stdin string, stdout string, stderr string) error {
// 	var err error

// 	p.io, err = setupIO(ctx, stdin, stdout, stderr)
// 	if err != nil {
// 		return errors.Errorf("setting up IO: %w", err)
// 	}

// 	if len(p.spec.Args) <= 0 {
// 		// Default to shell if no args provided
// 		p.spec.Args = []string{"/bin/sh"}
// 	}

// 	return nil
// }

// func (p *managedProcess) startIO(ctx context.Context, vm *vmm.RunningVM[*vf.VirtualMachine]) (err error) {
// 	slog.InfoContext(ctx, "managedProcess.startIO: starting process in VM")
// 	defer func() {
// 		if r := recover(); r != nil {
// 			slog.ErrorContext(ctx, "panic in startIO", "error", r)
// 		}
// 		slog.InfoContext(ctx, "startIO: finished")
// 	}()

// 	p.mu.Lock()
// 	defer p.mu.Unlock()

// 	exitCode, err := vm.RunWithStdio(ctx, nil, p.io.stdin, p.io.stdout, p.io.stderr)
// 	if err != nil {
// 		return errors.Errorf("running with stdio: %w", err)
// 	}

// 	slog.InfoContext(ctx, "managedProcess.startIO: starting process in VM", "exitCode", exitCode)

// 	p.exitStatus = uint32(exitCode)

// 	p.status = taskt.Status_STOPPED
// 	p.exitedAt = time.Now()

// 	return nil
// }

// 	slog.Info("managedProcess.start: Starting process in VM")

// 	// Wait for VM to be ready for exec
// 	slog.Info("managedProcess.start: Waiting for VM to be ready for exec")
// 	select {
// 	case <-p.vm.WaitOnVMReadyToExec():
// 		// VM is ready
// 		slog.Info("managedProcess.start: VM is ready for exec")
// 	case <-time.After(30 * time.Second):
// 		slog.Error("managedProcess.start: Timeout waiting for VM to be ready for exec")
// 		return fmt.Errorf("timeout waiting for VM to be ready for exec")
// 	}

// 	// // Build the command to execute
// 	var cmdParts []string

// 	// Handle working directory
// 	if p.spec.Cwd != "" {
// 		cmdParts = append(cmdParts, "cd", p.spec.Cwd, "&&")
// 	}

// 	// Handle environment variables
// 	for _, env := range p.spec.Env {
// 		cmdParts = append(cmdParts, "export", env, "&&")
// 	}

// 	// Add the actual command
// 	cmdParts = append(cmdParts, p.spec.Args...)

// 	command := strings.Join(cmdParts, " ")
// 	slog.Info("managedProcess.start: Executing command in VM", "command", command)

// 	// Create context for the command
// 	p.commandCtx, p.commandCancel = context.WithCancel(context.Background())

// 	p.status = taskt.Status_RUNNING

// 	// Execute the command in the VM
// 	go func() {
// 		defer func() {
// 			slog.Info("managedProcess.start: Command execution finished, updating process state")
// 			p.mu.Lock()
// 			p.status = taskt.Status_STOPPED
// 			p.exitedAt = time.Now()
// 			close(p.waitblock)
// 			p.mu.Unlock()
// 			slog.Info("managedProcess.start: Process state updated and waitblock closed")
// 		}()

// 		slog.Info("managedProcess.start: Calling vm.Exec")
// 		stdout, stderr, exitCode, err := p.vm.Exec(p.commandCtx, command)

// 		slog.Info("managedProcess.start: vm.Exec completed", "stdout_len", len(stdout), "stderr_len", len(stderr), "exit_code", string(exitCode), "error", err)

// 		// Write output to the configured I/O
// 		if len(stdout) > 0 {
// 			slog.Debug("managedProcess.start: Writing stdout", "length", len(stdout))
// 			p.io.stdout.Write(stdout)
// 		}
// 		if len(stderr) > 0 {
// 			slog.Debug("managedProcess.start: Writing stderr", "length", len(stderr))
// 			p.io.stderr.Write(stderr)
// 		}

// 		// Parse exit code
// 		if len(exitCode) > 0 {
// 			if code, parseErr := strconv.Atoi(strings.TrimSpace(string(exitCode))); parseErr == nil {
// 				slog.Info("managedProcess.start: Setting exit status", "exit_code", code)
// 				p.mu.Lock()
// 				p.exitStatus = uint32(code)
// 				p.mu.Unlock()
// 			} else {
// 				slog.Warn("managedProcess.start: Failed to parse exit code", "exit_code_raw", string(exitCode), "error", parseErr)
// 			}
// 		}

// 		if err != nil {
// 			slog.Error("managedProcess.start: Command execution failed", "error", err, "command", command)
// 			p.mu.Lock()
// 			p.exitStatus = 1
// 			p.mu.Unlock()
// 		}
// 	}()

// 	slog.Info("managedProcess.start: Process started successfully")
// 	return nil
// }
