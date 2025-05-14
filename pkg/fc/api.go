package fc

import (
	"context"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/restapi/operations"
	"github.com/walteh/ec1/pkg/hypervisors"
)

func NewEC1FirecrackerAPI[V hypervisors.VirtualMachine](hpv hypervisors.Hypervisor[V]) *EC1FirecrackerAPI[V] {
	return &EC1FirecrackerAPI[V]{
		hpv: hpv,
	}
}

var _ operations.FirecrackerAPI = &EC1FirecrackerAPI[hypervisors.VirtualMachine]{}

type EC1FirecrackerAPI[V hypervisors.VirtualMachine] struct {
	hpv hypervisors.Hypervisor[V]
}

// CreateSnapshot implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) CreateSnapshot(ctx context.Context, params operations.CreateSnapshotParams) operations.CreateSnapshotResponder {
	return operations.CreateSnapshotNotImplemented()
}

// CreateSyncAction implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) CreateSyncAction(ctx context.Context, params operations.CreateSyncActionParams) operations.CreateSyncActionResponder {
	return operations.CreateSyncActionNotImplemented()
}

// DescribeBalloonConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) DescribeBalloonConfig(ctx context.Context, params operations.DescribeBalloonConfigParams) operations.DescribeBalloonConfigResponder {
	return operations.DescribeBalloonConfigNotImplemented()
}

// DescribeBalloonStats implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) DescribeBalloonStats(ctx context.Context, params operations.DescribeBalloonStatsParams) operations.DescribeBalloonStatsResponder {
	return operations.DescribeBalloonStatsNotImplemented()
}

// DescribeInstance implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) DescribeInstance(ctx context.Context, params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {
	return operations.DescribeInstanceNotImplemented()
}

// GetExportVMConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) GetExportVMConfig(ctx context.Context, params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
	return operations.GetExportVMConfigNotImplemented()
}

// GetFirecrackerVersion implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) GetFirecrackerVersion(ctx context.Context, params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {
	return operations.GetFirecrackerVersionNotImplemented()
}

// GetMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) GetMachineConfiguration(ctx context.Context, params operations.GetMachineConfigurationParams) operations.GetMachineConfigurationResponder {
	return operations.GetMachineConfigurationNotImplemented()
}

// GetMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) GetMmds(ctx context.Context, params operations.GetMmdsParams) operations.GetMmdsResponder {
	return operations.GetMmdsNotImplemented()
}

// LoadSnapshot implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) LoadSnapshot(ctx context.Context, params operations.LoadSnapshotParams) operations.LoadSnapshotResponder {
	return operations.LoadSnapshotNotImplemented()
}

// PatchBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchBalloon(ctx context.Context, params operations.PatchBalloonParams) operations.PatchBalloonResponder {
	return operations.PatchBalloonNotImplemented()
}

// PatchBalloonStatsInterval implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchBalloonStatsInterval(ctx context.Context, params operations.PatchBalloonStatsIntervalParams) operations.PatchBalloonStatsIntervalResponder {
	return operations.PatchBalloonStatsIntervalNotImplemented()
}

// PatchGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchGuestDriveByID(ctx context.Context, params operations.PatchGuestDriveByIDParams) operations.PatchGuestDriveByIDResponder {
	return operations.PatchGuestDriveByIDNotImplemented()
}

// PatchGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchGuestNetworkInterfaceByID(ctx context.Context, params operations.PatchGuestNetworkInterfaceByIDParams) operations.PatchGuestNetworkInterfaceByIDResponder {
	return operations.PatchGuestNetworkInterfaceByIDNotImplemented()
}

// PatchMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchMachineConfiguration(ctx context.Context, params operations.PatchMachineConfigurationParams) operations.PatchMachineConfigurationResponder {
	return operations.PatchMachineConfigurationNotImplemented()
}

// PatchMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchMmds(ctx context.Context, params operations.PatchMmdsParams) operations.PatchMmdsResponder {
	return operations.PatchMmdsNotImplemented()
}

// PatchVM implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PatchVM(ctx context.Context, params operations.PatchVMParams) operations.PatchVMResponder {
	return operations.PatchVMNotImplemented()
}

// PutBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutBalloon(ctx context.Context, params operations.PutBalloonParams) operations.PutBalloonResponder {
	return operations.PutBalloonNotImplemented()
}

// PutCPUConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutCPUConfiguration(ctx context.Context, params operations.PutCPUConfigurationParams) operations.PutCPUConfigurationResponder {
	return operations.PutCPUConfigurationNotImplemented()
}

// PutEntropyDevice implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutEntropyDevice(ctx context.Context, params operations.PutEntropyDeviceParams) operations.PutEntropyDeviceResponder {
	return operations.PutEntropyDeviceNotImplemented()
}

// PutGuestBootSource implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutGuestBootSource(ctx context.Context, params operations.PutGuestBootSourceParams) operations.PutGuestBootSourceResponder {
	return operations.PutGuestBootSourceNotImplemented()
}

// PutGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutGuestDriveByID(ctx context.Context, params operations.PutGuestDriveByIDParams) operations.PutGuestDriveByIDResponder {
	return operations.PutGuestDriveByIDNotImplemented()
}

// PutGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutGuestNetworkInterfaceByID(ctx context.Context, params operations.PutGuestNetworkInterfaceByIDParams) operations.PutGuestNetworkInterfaceByIDResponder {
	return operations.PutGuestNetworkInterfaceByIDNotImplemented()
}

// PutGuestVsock implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutGuestVsock(ctx context.Context, params operations.PutGuestVsockParams) operations.PutGuestVsockResponder {
	return operations.PutGuestVsockNotImplemented()
}

// PutLogger implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutLogger(ctx context.Context, params operations.PutLoggerParams) operations.PutLoggerResponder {
	return operations.PutLoggerNotImplemented()
}

// PutMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutMachineConfiguration(ctx context.Context, params operations.PutMachineConfigurationParams) operations.PutMachineConfigurationResponder {
	return operations.PutMachineConfigurationNotImplemented()
}

// PutMetrics implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutMetrics(ctx context.Context, params operations.PutMetricsParams) operations.PutMetricsResponder {
	return operations.PutMetricsNotImplemented()
}

// PutMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutMmds(ctx context.Context, params operations.PutMmdsParams) operations.PutMmdsResponder {
	return operations.PutMmdsNotImplemented()
}

// PutMmdsConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) PutMmdsConfig(ctx context.Context, params operations.PutMmdsConfigParams) operations.PutMmdsConfigResponder {
	return operations.PutMmdsConfigNotImplemented()
}
