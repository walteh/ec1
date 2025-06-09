package containerd

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/api/runtime/task/v3"
	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/pkg/errgrpc"
	"github.com/containers/common/pkg/strongunits"
	"github.com/hashicorp/go-multierror"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/units"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

const unmountFlags = unix.MNT_FORCE

type container struct {
	// These fields are readonly and filled when container is created
	spec          *oci.Spec
	bundlePath    string
	rootfs        string
	dnsSocketPath string
	request       *task.CreateTaskRequest

	// VMM-specific fields
	vm         *vmm.RunningVM[*vf.VirtualMachine]
	imageRef   string
	hypervisor vmm.Hypervisor[*vf.VirtualMachine]
	// cache      ec1oci.ImageFetchConverter

	mu sync.Mutex

	// primary is the primary process for the container.
	// The lifetime of the container is tied to this process.
	primary managedProcess

	// auxiliary is a map of additional processes that run in the jail.
	auxiliary map[string]*managedProcess
}

func (c *container) destroy() (retErr error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop all auxiliary processes first
	for _, p := range c.auxiliary {
		if err := p.destroy(); err != nil {
			retErr = multierror.Append(retErr, err)
		}
	}

	// Stop the primary process
	if err := c.primary.destroy(); err != nil {
		retErr = multierror.Append(retErr, err)
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
		}
	}

	// Remove socket file to avoid continuity "failed to create irregular file" error during multiple Dockerfile  `RUN` steps
	_ = os.Remove(c.dnsSocketPath)

	if err := mount.UnmountRecursive(c.rootfs, unmountFlags); err != nil {
		retErr = multierror.Append(retErr, err)
	}

	return
}

func (c *container) getProcessL(ctx context.Context, execID string) (*managedProcess, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.getProcess(ctx, execID)
}

func (c *container) getProcess(ctx context.Context, execID string) (*managedProcess, error) {
	if execID == "" {
		return &c.primary, nil
	}

	p := c.auxiliary[execID]

	if p == nil {
		slog.ErrorContext(ctx, "exec not found", "execID", execID)
		return nil, errgrpc.ToGRPCf(errdefs.ErrNotFound, "exec not found: %s", execID)
	}

	return p, nil
}

// createVM creates and starts a new microVM for this container using the already-prepared rootfs
func (c *container) createVM(ctx context.Context, spec *oci.Spec, id string, execID string, rootfs string, stdio stdio) (retErr error) {
	// containerd has already prepared the rootfs for us at c.rootfs
	// We just need to create a VM that uses this existing rootfs directory\

	// if false {

	// this is a sample with another set of cfode that works during a test but does not in the shim

	// 	hpv := vf.NewHypervisor()

	// 	dir, err := os.MkdirTemp("", "ec1-test-cache")
	// 	if err != nil {
	// 		return errors.Errorf("creating temp dir: %w", err)
	// 	}

	// 	memMapFetcher := ec1oci.NewMemoryMapFetcher(dir, toci.Registry())
	// 	fetcher := ec1oci.NewImageCache(dir, memMapFetcher, ec1oci.NewOCIFilesystemConverter())

	// 	slog.InfoContext(ctx, "createVM: Running VM", "id", id, "execID", execID, "rootfs", rootfs)

	// 	rvm, err := vmm.NewContainerizedVirtualMachine(ctx, hpv, fetcher, vmm.ConatinerImageConfig{
	// 		ImageRef: oci_image_cache.ALPINE_LATEST.String(),
	// 		Platform: units.PlatformLinuxARM64,
	// 		Memory:   strongunits.MiB(64).ToBytes(),
	// 		VCPUs:    1,
	// 	})
	// 	if err != nil {
	// 		return errors.Errorf("creating VM: %w", err)
	// 	}

	// 	go func() {
	// 		slog.DebugContext(ctx, "vm running, waiting for vm to stop")
	// 		err := rvm.WaitOnVmStopped()
	// 		if err != nil {
	// 			slog.ErrorContext(ctx, "error waiting for VM to stop", "error", err)
	// 		}
	// 	}()

	// 	defer func() {
	// 		slog.DebugContext(ctx, "stopping vm")
	// 		rvm.VM().HardStop(ctx)
	// 	}()

	// 	err = vmm.WaitForVMState(ctx, rvm.VM(), vmm.VirtualMachineStateTypeRunning, time.After(30*time.Second))
	// 	if err != nil {
	// 		return errors.Errorf("timeout waiting for vm to be running: %w", err)
	// 	}

	// 	select {
	// 	case <-rvm.WaitOnVMReadyToExec():
	// 	case <-time.After(3 * time.Second):
	// 		return errors.Errorf("timeout waiting for vm to be ready to exec")
	// 	}

	// 	c.vm = rvm
	// 	return nil
	// }

	// Add panic recovery for VM creation
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "FATAL: createVM panic", "panic", r, "id", id, "execID", execID)
			retErr = errors.Errorf("VM creation panicked: %v", r)
			panic(r)
		}
	}()

	slog.InfoContext(ctx, "createVM: Starting VM creation", "id", id, "execID", execID, "rootfs", rootfs)

	// Extract configuration from the OCI spec
	memory := strongunits.MiB(64).ToBytes() // Use 512MB minimum for VZ compatibility
	vcpus := uint64(1)                      // Default, TODO: Extract from spec.Process or other location

	slog.InfoContext(ctx, "createVM: VM configuration", "memory", memory, "vcpus", vcpus)

	// Determine platform based on the OCI spec content and runtime architecture
	var platform units.Platform
	arch := runtime.GOARCH

	// Determine OS based on which platform-specific section is populated
	var osType string
	if c.spec.Linux != nil {
		osType = "linux"
	} else if c.spec.Windows != nil {
		osType = "windows"
	} else {
		// Default to linux if no platform-specific section is found
		osType = "linux"
	}

	// Create platform string and parse it
	platformStr := osType + "/" + arch
	platform, err := units.ParsePlatform(platformStr)
	if err != nil {
		// Fallback to ARM64 Linux if parsing fails
		platform = units.PlatformLinuxARM64
		slog.WarnContext(ctx, "createVM: Failed to parse platform, using fallback", "platformStr", platformStr, "fallback", platform)
	}

	vm, err := vmm.NewContainerizedVirtualMachineFromRootfs(ctx, c.hypervisor, vmm.ContainerizedVMConfig{
		ID:           id,
		ExecID:       execID,
		RootfsPath:   c.rootfs,
		RootfsMounts: c.request.Rootfs,
		StderrFD:     stdio.stderrFD,
		StdoutFD:     stdio.stdoutFD,
		StdinFD:      stdio.stdinFD,
		DNSPath:      c.dnsSocketPath,
		StdinPath:    c.request.Stdin,
		StdoutPath:   c.request.Stdout,
		StderrPath:   c.request.Stderr,
		Spec:         spec,
		Platform:     platform,
		Memory:       memory,
		VCPUs:        vcpus,
	})

	if err != nil {
		return errors.Errorf("creating VM from rootfs: %w", err)
	}

	// to := time.NewTimer(10 * time.Second)
	// defer to.Stop()

	// if err := vmm.WaitForVMState(ctx, vm.VM(), vmm.VirtualMachineStateTypeRunning, to.C); err != nil {
	// 	return errors.Errorf("timeout waiting for VM to start: %w", err)
	// }

	slog.InfoContext(ctx, "createVM: VM created successfully")
	c.vm = vm
	return nil
}
