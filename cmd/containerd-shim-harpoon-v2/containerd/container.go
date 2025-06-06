package containerd

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"sync"

	"golang.org/x/sys/unix"

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

func (c *container) getProcessL(execID string) (*managedProcess, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.getProcess(execID)
}

func (c *container) getProcess(execID string) (*managedProcess, error) {
	if execID == "" {
		return &c.primary, nil
	}

	p := c.auxiliary[execID]

	if p == nil {
		return nil, errgrpc.ToGRPCf(errdefs.ErrNotFound, "exec not found: %s", execID)
	}

	return p, nil
}

// createVM creates and starts a new microVM for this container using the already-prepared rootfs
func (c *container) createVM(ctx context.Context, spec *oci.Spec, rootfs string, stdio stdio) error {
	// containerd has already prepared the rootfs for us at c.rootfs
	// We just need to create a VM that uses this existing rootfs directory

	// Extract configuration from the OCI spec
	memory := strongunits.MiB(128).ToBytes() // Default, TODO: Extract from spec.Linux.Resources.Memory
	vcpus := uint64(1)                       // Default, TODO: Extract from spec.Process or other location

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
	}

	vm, err := vmm.NewContainerizedVirtualMachineFromRootfs(ctx, c.hypervisor, vmm.ContainerizedVMConfig{
		RootfsPath: c.rootfs,
		StderrFD:   stdio.stderrFD,
		StdoutFD:   stdio.stdoutFD,
		StdinFD:    stdio.stdinFD,
		Spec:       spec,
		Platform:   platform,
		Memory:     memory,
		VCPUs:      vcpus,
	})
	if err != nil {
		return errors.Errorf("creating VM from rootfs: %w", err)
	}

	c.vm = vm
	return nil
}
