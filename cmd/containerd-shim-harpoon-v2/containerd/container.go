package containerd

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/containerd/containerd/v2/core/mount"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/pkg/errgrpc"
	"github.com/containers/common/pkg/strongunits"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	ec1oci "github.com/walteh/ec1/pkg/oci"

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

// createVM creates and starts a new microVM for this container
func (c *container) createVM(ctx context.Context) error {
	// Determine the image reference from the rootfs mounts
	// For now, we'll use a default image - in a real implementation,
	// this would be extracted from the container spec or rootfs
	imageRef := "alpine:latest" // TODO: Extract from container spec

	imageConfig := vmm.ConatinerImageConfig{
		ImageRef: imageRef,
		Platform: units.PlatformLinuxARM64,       // TODO: Detect from spec
		Memory:   strongunits.MiB(128).ToBytes(), // TODO: Extract from spec
		VCPUs:    1,                              // TODO: Extract from spec
	}

	// we need to have created a ImageFetchConverter from the information passed via containerd
	// note the ext4 thing is not used so no need to create that

	var crf ec1oci.ImageFetchConverter
	if crf == nil {
		return errors.Errorf("no image fetch converter")
	}

	vm, err := vmm.NewContainerizedVirtualMachine(ctx, c.hypervisor, crf, imageConfig)
	if err != nil {
		return err
	}

	c.vm = vm
	return nil
}
