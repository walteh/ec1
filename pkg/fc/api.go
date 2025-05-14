package fc

import (
	"context"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/restapi/operations"
)

func NewEC1FirecrackerAPI() *EC1FirecrackerAPI {
	return &EC1FirecrackerAPI{}
}

var _ operations.FirecrackerAPI = &EC1FirecrackerAPI{}

type EC1FirecrackerAPI struct {
}

// CreateSnapshot implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) CreateSnapshot(ctx context.Context, params operations.CreateSnapshotParams) operations.CreateSnapshotResponder {
	return operations.CreateSnapshotNotImplemented()
}

// CreateSyncAction implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) CreateSyncAction(ctx context.Context, params operations.CreateSyncActionParams) operations.CreateSyncActionResponder {
	return operations.CreateSyncActionNotImplemented()
}

// DescribeBalloonConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeBalloonConfig(ctx context.Context, params operations.DescribeBalloonConfigParams) operations.DescribeBalloonConfigResponder {
	return operations.DescribeBalloonConfigNotImplemented()
}

// DescribeBalloonStats implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeBalloonStats(ctx context.Context, params operations.DescribeBalloonStatsParams) operations.DescribeBalloonStatsResponder {
	return operations.DescribeBalloonStatsNotImplemented()
}

// DescribeInstance implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeInstance(ctx context.Context, params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {
	return operations.DescribeInstanceNotImplemented()
}

// GetExportVMConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetExportVMConfig(ctx context.Context, params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
	return operations.GetExportVMConfigNotImplemented()
}

// GetFirecrackerVersion implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetFirecrackerVersion(ctx context.Context, params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {
	return operations.GetFirecrackerVersionNotImplemented()
}

// GetMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetMachineConfiguration(ctx context.Context, params operations.GetMachineConfigurationParams) operations.GetMachineConfigurationResponder {
	return operations.GetMachineConfigurationNotImplemented()
}

// GetMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetMmds(ctx context.Context, params operations.GetMmdsParams) operations.GetMmdsResponder {
	return operations.GetMmdsNotImplemented()
}

// LoadSnapshot implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) LoadSnapshot(ctx context.Context, params operations.LoadSnapshotParams) operations.LoadSnapshotResponder {
	return operations.LoadSnapshotNotImplemented()
}

// PatchBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchBalloon(ctx context.Context, params operations.PatchBalloonParams) operations.PatchBalloonResponder {
	return operations.PatchBalloonNotImplemented()
}

// PatchBalloonStatsInterval implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchBalloonStatsInterval(ctx context.Context, params operations.PatchBalloonStatsIntervalParams) operations.PatchBalloonStatsIntervalResponder {
	return operations.PatchBalloonStatsIntervalNotImplemented()
}

// PatchGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchGuestDriveByID(ctx context.Context, params operations.PatchGuestDriveByIDParams) operations.PatchGuestDriveByIDResponder {
	return operations.PatchGuestDriveByIDNotImplemented()
}

// PatchGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchGuestNetworkInterfaceByID(ctx context.Context, params operations.PatchGuestNetworkInterfaceByIDParams) operations.PatchGuestNetworkInterfaceByIDResponder {
	return operations.PatchGuestNetworkInterfaceByIDNotImplemented()
}

// PatchMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchMachineConfiguration(ctx context.Context, params operations.PatchMachineConfigurationParams) operations.PatchMachineConfigurationResponder {
	return operations.PatchMachineConfigurationNotImplemented()
}

// PatchMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchMmds(ctx context.Context, params operations.PatchMmdsParams) operations.PatchMmdsResponder {
	return operations.PatchMmdsNotImplemented()
}

// PatchVM implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchVM(ctx context.Context, params operations.PatchVMParams) operations.PatchVMResponder {
	return operations.PatchVMNotImplemented()
}

// PutBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutBalloon(ctx context.Context, params operations.PutBalloonParams) operations.PutBalloonResponder {
	return operations.PutBalloonNotImplemented()
}

// PutCPUConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutCPUConfiguration(ctx context.Context, params operations.PutCPUConfigurationParams) operations.PutCPUConfigurationResponder {
	return operations.PutCPUConfigurationNotImplemented()
}

// PutEntropyDevice implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutEntropyDevice(ctx context.Context, params operations.PutEntropyDeviceParams) operations.PutEntropyDeviceResponder {
	return operations.PutEntropyDeviceNotImplemented()
}

// PutGuestBootSource implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestBootSource(ctx context.Context, params operations.PutGuestBootSourceParams) operations.PutGuestBootSourceResponder {
	return operations.PutGuestBootSourceNotImplemented()
}

// PutGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestDriveByID(ctx context.Context, params operations.PutGuestDriveByIDParams) operations.PutGuestDriveByIDResponder {
	return operations.PutGuestDriveByIDNotImplemented()
}

// PutGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestNetworkInterfaceByID(ctx context.Context, params operations.PutGuestNetworkInterfaceByIDParams) operations.PutGuestNetworkInterfaceByIDResponder {
	return operations.PutGuestNetworkInterfaceByIDNotImplemented()
}

// PutGuestVsock implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestVsock(ctx context.Context, params operations.PutGuestVsockParams) operations.PutGuestVsockResponder {
	return operations.PutGuestVsockNotImplemented()
}

// PutLogger implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutLogger(ctx context.Context, params operations.PutLoggerParams) operations.PutLoggerResponder {
	return operations.PutLoggerNotImplemented()
}

// PutMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMachineConfiguration(ctx context.Context, params operations.PutMachineConfigurationParams) operations.PutMachineConfigurationResponder {
	return operations.PutMachineConfigurationNotImplemented()
}

// PutMetrics implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMetrics(ctx context.Context, params operations.PutMetricsParams) operations.PutMetricsResponder {
	return operations.PutMetricsNotImplemented()
}

// PutMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMmds(ctx context.Context, params operations.PutMmdsParams) operations.PutMmdsResponder {
	return operations.PutMmdsNotImplemented()
}

// PutMmdsConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMmdsConfig(ctx context.Context, params operations.PutMmdsConfigParams) operations.PutMmdsConfigResponder {
	return operations.PutMmdsConfigNotImplemented()
}
