//
// Copyright (c) 2023 Apple Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package kata

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/containers/common/pkg/strongunits"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/types"
	"gitlab.com/tozd/go/errors"

	hv "github.com/kata-containers/kata-containers/src/runtime/pkg/hypervisors"

	"github.com/walteh/ec1/pkg/machines/bootloader"
	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

// This line is a compile-time check. If this file compiles, kataHypervisor implements virtcontainers.Hypervisor.
var _ virtcontainers.Hypervisor = (*kataHypervisor[vmm.VirtualMachine])(nil)

type kataHypervisor[VM vmm.VirtualMachine] struct {
	hypervisor      vmm.Hypervisor[VM]
	managedVm       VM
	config          *virtcontainers.HypervisorConfig
	creationContext context.Context
	vsockProxy      *virtio.VirtioVsock
}

func WrapHypervisorForKata[VM vmm.VirtualMachine](ctx context.Context, parentHypervisor vmm.Hypervisor[VM]) (virtcontainers.Hypervisor, error) {
	if parentHypervisor == nil {
		return nil, fmt.Errorf("parentHypervisor cannot be nil")
	}
	return &kataHypervisor[VM]{hypervisor: parentHypervisor}, nil
}

func unimplemented() error {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("kataHypervisor: method not implemented (runtime.Caller failed)")
	}
	details := runtime.FuncForPC(pc)
	if details == nil {
		return fmt.Errorf("kataHypervisor: method not implemented (runtime.FuncForPC failed)")
	}

	parts := strings.Split(details.Name(), ".")
	funcName := parts[len(parts)-1]
	// Check if the second to last part is our struct name
	if len(parts) > 1 && strings.HasSuffix(parts[len(parts)-2], "kataHypervisor") {
		return fmt.Errorf("kataHypervisor.%s is not implemented yet", funcName)
	}
	return fmt.Errorf("method %s (within kataHypervisor) is not implemented yet", funcName)
}

// virtcontainers.Hypervisor interface implementation

func (vfw *kataHypervisor[VM]) CreateVM(ctx context.Context, id string, network virtcontainers.Network, hypervisorConfig *virtcontainers.HypervisorConfig) error {
	vfw.creationContext = ctx
	if vfw.hypervisor == nil {
		return fmt.Errorf("underlying hypervisor is not initialized")
	}
	vfw.config = hypervisorConfig

	var kernelParams []string
	for _, p := range hypervisorConfig.KernelParams {
		if p.Value != "" {
			kernelParams = append(kernelParams, p.Key+"="+p.Value)
		} else {
			kernelParams = append(kernelParams, p.Key)
		}
	}
	var vmDevices []virtio.VirtioDevice

	// vfw.vsockProxy = &virtio.VirtioVsock{
	// 	Port:      9994,
	// 	SocketURL: "kata.sock",
	// 	Direction: virtio.VirtioVsockDirectionGuestListensAsServer,
	// }
	// vmDevices = append(vmDevices, vfw.vsockProxy)
	// kernelCmdLine := strings.Join(kernelParams, " ")

	// decompress the initrd

	// bl := bootloader.NewLinuxBootloader(
	// 	hypervisorConfig.KernelPath,
	// 	kernelCmdLine,
	// 	hypervisorConfig.InitrdPath,
	// )

	workDir := filepath.Join(filepath.Dir(hypervisorConfig.KernelPath), "vf")

	bl := bootloader.NewEFIBootloader(workDir, true)

	memBytes := strongunits.B(uint64(hypervisorConfig.MemorySize) * 1024 * 1024)

	for _, devCfg := range hypervisorConfig.VFIODevices {
		// Type of device: c, b, u or p
		// c , u - character(unbuffered)
		// p - FIFO
		// b - block(buffered) special file
		// More info in mknod(1).
		switch devCfg.DevType {
		case "c":
			serialDev, err := virtio.VirtioSerialNewPty()
			if err == nil {
				vmDevices = append(vmDevices, serialDev)
			}
		case "b":
			if devCfg.HostPath != "" {
				virtioBlk, err := virtio.VirtioBlkNew(devCfg.HostPath)
				if err == nil {
					if devCfg.ID != "" {
						virtioBlk.SetDeviceIdentifier(devCfg.ID)
					}
					vmDevices = append(vmDevices, virtioBlk)
				}
			}
		}

	}

	foundRng := false
	for _, dev := range vmDevices {
		if _, ok := dev.(*virtio.VirtioRng); ok {
			foundRng = true
			break
		}
	}
	if !foundRng {
		rngDev, _ := virtio.VirtioRngNew()
		if rngDev != nil {
			vmDevices = append(vmDevices, rngDev)
		}
	}

	opts := vmm.NewVMOptions{
		Vcpus:   uint(hypervisorConfig.NumVCPUs()),
		Memory:  memBytes,
		Devices: vmDevices,
	}

	// go func() {
	// 	<-vfw.hypervisor
	// }()

	vm, err := vfw.hypervisor.NewVirtualMachine(ctx, id, opts, bl)
	if err != nil {
		return fmt.Errorf("failed to create VM using underlying hypervisor: %w", err)
	}
	vfw.managedVm = vm
	return nil
}

func (vfw *kataHypervisor[VM]) StartVM(ctx context.Context, timeout int) error {
	if any(vfw.managedVm) == nil {
		return fmt.Errorf("VM not created yet, cannot start")
	}
	return vfw.managedVm.Start(ctx)
}

func (vfw *kataHypervisor[VM]) StopVM(ctx context.Context, waitOnly bool) error {
	if any(vfw.managedVm) == nil {
		return fmt.Errorf("VM not created yet, cannot stop")
	}
	if waitOnly {
		_, err := vfw.managedVm.RequestStop(ctx)
		return err
	}
	return vfw.managedVm.HardStop(ctx)
}

func (vfw *kataHypervisor[VM]) PauseVM(ctx context.Context) error {
	if any(vfw.managedVm) == nil {
		return fmt.Errorf("VM not created yet, cannot pause")
	}
	if !vfw.managedVm.CanPause(ctx) {
		return fmt.Errorf("VM cannot be paused in its current state: %s", vfw.managedVm.CurrentState())
	}
	return vfw.managedVm.Pause(ctx)
}

func (vfw *kataHypervisor[VM]) SaveVM() error {
	return unimplemented()
}

func (vfw *kataHypervisor[VM]) ResumeVM(ctx context.Context) error {
	if any(vfw.managedVm) == nil {
		return fmt.Errorf("VM not created yet, cannot resume")
	}
	if !vfw.managedVm.CanResume(ctx) {
		return fmt.Errorf("VM cannot be resumed in its current state: %s", vfw.managedVm.CurrentState())
	}
	return vfw.managedVm.Resume(ctx)
}

func (vfw *kataHypervisor[VM]) AddDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) error {
	return unimplemented()
}

func (vfw *kataHypervisor[VM]) HotplugAddDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) (interface{}, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor[VM]) HotplugRemoveDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) (interface{}, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor[VM]) ResizeMemory(ctx context.Context, memMB uint32, memoryBlockSizeMB uint32, probe bool) (uint32, virtcontainers.MemoryDevice, error) {
	if any(vfw.managedVm) == nil {
		return 0, virtcontainers.MemoryDevice{}, fmt.Errorf("VM not created or running")
	}
	targetBytes := strongunits.B(uint64(memMB) * 1024 * 1024)
	err := vfw.managedVm.SetMemoryBalloonTargetSize(ctx, targetBytes)
	if err != nil {
		return 0, virtcontainers.MemoryDevice{}, err
	}
	currentSize, errGet := vfw.managedVm.GetMemoryBalloonTargetSize(ctx)
	actualSizeMB := memMB // Default to requested if query fails
	if errGet == nil {
		actualSizeMB = uint32(strongunits.ToMib(currentSize))
	}
	// slot := 0
	memDevice := virtcontainers.MemoryDevice{
		// Type:   virtcontainers.MemoryDeviceTypeDIMM,
		SizeMB: int(actualSizeMB),
		// Slot:   int(slot),

	}
	return actualSizeMB, memDevice, nil
}

func (vfw *kataHypervisor[VM]) ResizeVCPUs(ctx context.Context, vcpus uint32) (uint32, uint32, error) {
	return 0, 0, unimplemented()
}

func (vfw *kataHypervisor[VM]) GetVMConsole(ctx context.Context, sandboxID string) (string, string, error) {
	return "", "", unimplemented()
}

func (vfw *kataHypervisor[VM]) Disconnect(ctx context.Context) {
}

func (vfw *kataHypervisor[VM]) Capabilities(ctx context.Context) types.Capabilities {
	var caps types.Capabilities
	caps.SetMultiQueueSupport()
	return caps
}

func (vfw *kataHypervisor[VM]) HypervisorConfig() virtcontainers.HypervisorConfig {
	if vfw.config != nil {
		return *vfw.config
	}
	return virtcontainers.HypervisorConfig{}
}

func (vfw *kataHypervisor[VM]) GetThreadIDs(ctx context.Context) (virtcontainers.VcpuThreadIDs, error) {
	return virtcontainers.NewVcpuThreadIds(make(map[int]int)), unimplemented()
}

func (vfw *kataHypervisor[VM]) Cleanup(ctx context.Context) error {
	if any(vfw.managedVm) != nil {
		currentState := vfw.managedVm.CurrentState()
		if currentState == vmm.VirtualMachineStateTypeRunning || currentState == vmm.VirtualMachineStateTypePaused {
			vfw.managedVm.HardStop(ctx)
		}
	}
	var zero VM
	vfw.managedVm = zero
	vfw.config = nil
	return nil
}

func (vfw *kataHypervisor[VM]) GetTotalMemoryMB(ctx context.Context) uint32 {
	total, err := vfw.managedVm.GetMemoryBalloonTargetSize(ctx)
	if err != nil {
		return 0
	}
	return uint32(strongunits.ToMib(total))
}

func (vfw *kataHypervisor[VM]) SetConfig(config *virtcontainers.HypervisorConfig) error {
	vfw.config = config
	return nil
}

func (vfw *kataHypervisor[VM]) GetPids() []int {

	return []int{}
}

func (vfw *kataHypervisor[VM]) GetVirtioFsPid() *int {
	return nil
}

func (vfw *kataHypervisor[VM]) FromGrpc(ctx context.Context, hypervisorConfig *virtcontainers.HypervisorConfig, j []byte) error {
	return unimplemented()
}

func (vfw *kataHypervisor[VM]) ToGrpc(ctx context.Context) ([]byte, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor[VM]) Check() error {
	if vfw.hypervisor == nil {
		return fmt.Errorf("underlying hypervisor not set")
	}
	return nil
}

func (vfw *kataHypervisor[VM]) Save() hv.HypervisorState {
	return hv.HypervisorState{}
}

func (vfw *kataHypervisor[VM]) Load(s hv.HypervisorState) {

}

// this is the kata-agent vsock that needs created
func (vfw *kataHypervisor[VM]) GenerateSocket(id string) (interface{}, error) {

	// needs  to grab the socket fid somehow and create a new proxy to the it and return it

	// can the proxy be a fd iteself that we return?

	fd, closerFunc, err := vmm.NewVSockStreamFileProxy(vfw.creationContext, vfw.managedVm, 1024)
	if err != nil {
		return nil, errors.Errorf("failed to create vsock stream file proxy: %w", err)
	}
	defer closerFunc()

	return &types.VSock{
		VhostFd: fd,
		Port:    vfw.vsockProxy.Port,
		// 3 for guests, 2 for hosts
		// https://developer.apple.com/forums/thread/772288
		ContextID: 3,
	}, nil
}

func (vfw *kataHypervisor[VM]) IsRateLimiterBuiltin() bool {
	return false
}

// func (vfw *kataHypervisor[VM]) AddDeviceFtrace(ctx context.Context, ID string, eventStr string, path string) (string, error) {
// 	return "", unimplemented()
// }

// func (vfw *kataHypervisor[VM]) GetHypervisorMetrics(ctx context.Context, groupID string, metricsConfig interface{}) (string, error) {
// 	return "", unimplemented()
// }

// func (vfw *kataHypervisor[VM]) ListBlockDevices(ctx context.Context) ([]*types.BlockDrive, error) {
// 	return nil, unimplemented()
// }

// func (vfw *kataHypervisor[VM]) GetIPTables(ctx context.Context, isIPv6 bool) ([]byte, error) {
// 	return nil, unimplemented()
// }

// func (vfw *kataHypervisor[VM]) SetIPTables(ctx context.Context, isIPv6 bool, data []byte) error {
// 	return unimplemented()
// }
