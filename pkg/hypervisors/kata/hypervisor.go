//
// Copyright (c) 2023 Apple Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package kata

import (
	"context"
	"fmt"
	"runtime"

	hv "github.com/kata-containers/kata-containers/src/runtime/pkg/hypervisors"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers/types"
	"github.com/walteh/ec1/pkg/hypervisors"
)

var _ virtcontainers.Hypervisor = &kataHypervisor{}

type kataHypervisor struct {
	hypervisor hypervisors.Hypervisor
}

func unimplemented() error {
	callerInfo, _, _, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(callerInfo).Name()
	return fmt.Errorf("kataHypervisor hypervisor: %s is not implemented yet", funcName)
}

func (vfw *kataHypervisor) CreateVM(ctx context.Context, id string, network virtcontainers.Network, hypervisorConfig *virtcontainers.HypervisorConfig) error {
	return unimplemented()
}

func (vfw *kataHypervisor) StartVM(ctx context.Context, timeout int) error {
	return unimplemented()
}

// If wait is set, don't actively stop the sandbox:
// just perform cleanup.
func (vfw *kataHypervisor) StopVM(ctx context.Context, waitOnly bool) error {
	return unimplemented()
}

func (vfw *kataHypervisor) PauseVM(ctx context.Context) error {
	return unimplemented()
}

func (vfw *kataHypervisor) SaveVM() error {
	return unimplemented()
}

func (vfw *kataHypervisor) ResumeVM(ctx context.Context) error {
	return unimplemented()
}

func (vfw *kataHypervisor) AddDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) error {
	return unimplemented()
}

func (vfw *kataHypervisor) HotplugAddDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) (interface{}, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor) HotplugRemoveDevice(ctx context.Context, devInfo interface{}, devType virtcontainers.DeviceType) (interface{}, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor) ResizeMemory(ctx context.Context, memMB uint32, memoryBlockSizeMB uint32, probe bool) (uint32, virtcontainers.MemoryDevice, error) {
	return 0, virtcontainers.MemoryDevice{}, unimplemented()
}

func (vfw *kataHypervisor) ResizeVCPUs(ctx context.Context, vcpus uint32) (uint32, uint32, error) {
	return 0, 0, unimplemented()
}

func (vfw *kataHypervisor) GetVMConsole(ctx context.Context, sandboxID string) (string, string, error) {
	return "", "", unimplemented()
}

func (vfw *kataHypervisor) Disconnect(ctx context.Context) {
	panic(unimplemented())
}

func (vfw *kataHypervisor) Capabilities(ctx context.Context) types.Capabilities {
	panic(unimplemented())
	return types.Capabilities{}
}

func (vfw *kataHypervisor) HypervisorConfig() virtcontainers.HypervisorConfig {
	panic(unimplemented())
	return virtcontainers.HypervisorConfig{}
}

func (vfw *kataHypervisor) GetThreadIDs(ctx context.Context) (virtcontainers.VcpuThreadIDs, error) {
	vcpuInfo := virtcontainers.NewVcpuThreadIds(make(map[int]int))

	panic(unimplemented())
	return vcpuInfo, nil
}

func (vfw *kataHypervisor) Cleanup(ctx context.Context) error {
	panic(unimplemented())
}

func (vfw *kataHypervisor) GetTotalMemoryMB(ctx context.Context) uint32 {
	panic(unimplemented())
	return 0
}

func (vfw *kataHypervisor) SetConfig(config *virtcontainers.HypervisorConfig) error {
	panic(unimplemented())
}

func (vfw *kataHypervisor) GetPids() []int {
	panic(unimplemented())
}

func (vfw *kataHypervisor) GetVirtioFsPid() *int {
	panic(unimplemented())
}

func (vfw *kataHypervisor) FromGrpc(ctx context.Context, hypervisorConfig *virtcontainers.HypervisorConfig, j []byte) error {
	panic(unimplemented())
}

func (vfw *kataHypervisor) ToGrpc(ctx context.Context) ([]byte, error) {
	panic(unimplemented())
}

func (vfw *kataHypervisor) Check() error {
	panic(unimplemented())
}

func (vfw *kataHypervisor) Save() hv.HypervisorState {
	panic(unimplemented())
}

func (vfw *kataHypervisor) Load(hv.HypervisorState) {
	panic(unimplemented())
}

func (vfw *kataHypervisor) GenerateSocket(id string) (interface{}, error) {
	return nil, unimplemented()
}

func (vfw *kataHypervisor) IsRateLimiterBuiltin() bool {
	panic(unimplemented())
	return false
}
