package containerd

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/api/runtime/task/v3"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/pkg/errgrpc"
	"github.com/containerd/typeurl/v2"
	"github.com/containers/common/pkg/strongunits"
	"github.com/hashicorp/go-multierror"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

const unmountFlags = unix.MNT_FORCE

type container struct {
	// These fields are readonly and filled when container is created
	spec       *oci.Spec
	bundlePath string
	request    *task.CreateTaskRequest
	pid        int
	// VMM-specific fields
	vm         *vmm.RunningVM[*vf.VirtualMachine]
	hypervisor vmm.Hypervisor[*vf.VirtualMachine]

	processesMu sync.Mutex

	// the "" key is the primary process.
	// Containerd will pass an empty string for the execID to signify the primary process.
	processes map[string]*managedProcess
}

func (c *container) getAllProcesses() []*managedProcess {
	c.processesMu.Lock()
	defer c.processesMu.Unlock()

	processes := make([]*managedProcess, 0, len(c.processes))
	for _, p := range c.processes {
		processes = append(processes, p)
	}

	return processes
}

func (c *container) getProcess(ctx context.Context, processID string) (*managedProcess, error) {
	c.processesMu.Lock()
	defer c.processesMu.Unlock()

	if processID == "" {
		processID = "primary"
	}

	p, ok := c.processes[processID]
	if !ok {
		return nil, errgrpc.ToGRPCf(errdefs.ErrNotFound, "process not found: %s", processID)
	}

	return p, nil
}

func (c *container) setProcess(ctx context.Context, p *managedProcess) error {
	c.processesMu.Lock()
	defer c.processesMu.Unlock()

	if _, ok := c.processes[p.id]; ok {
		return errgrpc.ToGRPCf(errdefs.ErrAlreadyExists, "process already exists: %s", p.id)
	}

	c.processes[p.id] = p
	return nil
}

func (c *container) destroy() (retErr error) {

	// Stop all auxiliary processes first
	for _, p := range c.processes {
		if err := p.destroy(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	// Stop the VM if it's running
	if c.vm != nil {
		ctx := context.Background()
		if c.vm.VM().CurrentState() == vmm.VirtualMachineStateTypeRunning {
			if err := c.vm.VM().HardStop(ctx); err != nil {
				slog.WarnContext(ctx, "failed to stop VM", "error", err)
				retErr = multierror.Append(retErr, err)
			}
		}

		// Wait for VM to stop
		if err := c.vm.WaitOnVmStopped(); err != nil {
			slog.WarnContext(ctx, "error waiting for VM to stop", "error", err)
			retErr = multierror.Append(retErr, err)
		}
	}

	// // Remove socket file to avoid continuity "failed to create irregular file" error during multiple Dockerfile  `RUN` steps
	// _ = os.Remove(c.dnsSocketPath)

	// // if err := mount.UnmountRecursive(c.rootfs, unmountFlags); err != nil {
	// // 	retErr = multierror.Append(retErr, err)
	// // }

	return retErr
}

// createContainerizedVM creates and starts a new microVM for this container using the already-prepared rootfs
func createContainerizedVM[H vmm.VirtualMachine](ctx context.Context, hypervisor vmm.Hypervisor[H], spec *oci.Spec, createRequest *task.CreateTaskRequest, stdio stdio) (vm *vmm.RunningVM[H], retErr error) {

	// Add panic recovery for VM creation
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "FATAL: createVM panic", "panic", r, "id", createRequest.ID)
			retErr = errors.Errorf("VM creation panicked: %v", r)
			panic(r)
		}
	}()

	// Extract configuration from the OCI spec
	memory := strongunits.MiB(128).ToBytes() // Use 512MB minimum for VZ compatibility
	vcpus := uint64(1)                       // Default, TODO: Extract from spec.Process or other location

	var platform units.Platform

	if spec.Annotations["nerdctl/platform"] != "" {
		platform = units.Platform(spec.Annotations["nerdctl/platform"])
	}

	if platform == "" {
		if spec.Linux != nil {
			platform = units.PlatformLinuxARM64
		} else {
			platform = units.PlatformDarwinARM64
		}
	}

	// slog.InfoContext(ctx, "createVM: VM configuration", "memory", memory, "vcpus", vcpus, "spec", valuelog.NewPrettyValue(spec), "platform", platform)

	vm, err := vmm.NewContainerizedVirtualMachineFromRootfs(ctx, hypervisor, vmm.ContainerizedVMConfig{
		ID:           createRequest.ID,
		RootfsMounts: createRequest.Rootfs,
		StderrWriter: stdio.stderr,
		StdoutWriter: stdio.stdout,
		StdinReader:  stdio.stdin,
		Spec:         spec,
		Platform:     platform,
		Memory:       memory,
		VCPUs:        vcpus,
	})

	if err != nil {
		return nil, errors.Errorf("creating VM from rootfs: %w", err)
	}

	// to := time.NewTimer(10 * time.Second)
	// defer to.Stop()

	// if err := vmm.WaitForVMState(ctx, vm.VM(), vmm.VirtualMachineStateTypeRunning, to.C); err != nil {
	// 	return errors.Errorf("timeout waiting for VM to start: %w", err)
	// }

	slog.InfoContext(ctx, "createVM: VM created successfully")

	return vm, nil
}

func NewContainer(ctx context.Context, hypervisor vmm.Hypervisor[*vf.VirtualMachine], spec *oci.Spec, createRequest *task.CreateTaskRequest) (*container, *managedProcess, error) {

	iod, err := setupIO(ctx, createRequest.Stdin, createRequest.Stdout, createRequest.Stderr)
	if err != nil {
		return nil, nil, errors.Errorf("setting up IO: %w", err)
	}
	vm, err := createContainerizedVM(ctx, hypervisor, spec, createRequest, iod)
	if err != nil {
		return nil, nil, errors.Errorf("creating vm: %w", err)
	}

	c := &container{
		request:    createRequest,
		pid:        os.Getpid(),
		vm:         vm,
		spec:       spec,
		bundlePath: createRequest.Bundle,
		hypervisor: hypervisor,
	}

	primary := NewManagedProcess("primary", c, spec.Process, iod)
	primary.pid = 0 //this is the primary process, so it doesn't have a pid
	c.processes = map[string]*managedProcess{"primary": primary}

	return c, primary, err
}

func (c *container) AddProcess(ctx context.Context, execRequest *task.ExecProcessRequest) (*managedProcess, error) {

	iod, err := setupIO(ctx, execRequest.Stdin, execRequest.Stdout, execRequest.Stderr)
	if err != nil {
		return nil, errors.Errorf("setting up IO: %w", err)
	}

	specAny, err := typeurl.UnmarshalAny(execRequest.Spec)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal spec: %w", err)
	}

	spec, ok := specAny.(*specs.Process)
	if !ok {
		return nil, errors.Errorf("invalid spec type '%T', expected *specs.Process", specAny)
	}

	process := NewManagedProcess(execRequest.ExecID, c, spec, iod)

	if err := c.setProcess(ctx, process); err != nil {
		return nil, errors.Errorf("setting process: %w", err)
	}

	return process, nil
}
