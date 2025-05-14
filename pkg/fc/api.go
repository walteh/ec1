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
	panic("unimplemented")
}

// CreateSyncAction implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) CreateSyncAction(ctx context.Context, params operations.CreateSyncActionParams) operations.CreateSyncActionResponder {
	panic("unimplemented")
}

// DescribeBalloonConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeBalloonConfig(ctx context.Context, params operations.DescribeBalloonConfigParams) operations.DescribeBalloonConfigResponder {
	panic("unimplemented")
}

// DescribeBalloonStats implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeBalloonStats(ctx context.Context, params operations.DescribeBalloonStatsParams) operations.DescribeBalloonStatsResponder {
	panic("unimplemented")
}

// DescribeInstance implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) DescribeInstance(ctx context.Context, params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {

	panic("unimplemented")
}

// GetExportVMConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetExportVMConfig(ctx context.Context, params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
	panic("unimplemented")
}

// GetFirecrackerVersion implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetFirecrackerVersion(ctx context.Context, params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {
	panic("unimplemented")
}

// GetMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetMachineConfiguration(ctx context.Context, params operations.GetMachineConfigurationParams) operations.GetMachineConfigurationResponder {
	panic("unimplemented")
}

// GetMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) GetMmds(ctx context.Context, params operations.GetMmdsParams) operations.GetMmdsResponder {
	panic("unimplemented")
}

// LoadSnapshot implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) LoadSnapshot(ctx context.Context, params operations.LoadSnapshotParams) operations.LoadSnapshotResponder {
	panic("unimplemented")
}

// PatchBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchBalloon(ctx context.Context, params operations.PatchBalloonParams) operations.PatchBalloonResponder {
	panic("unimplemented")
}

// PatchBalloonStatsInterval implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchBalloonStatsInterval(ctx context.Context, params operations.PatchBalloonStatsIntervalParams) operations.PatchBalloonStatsIntervalResponder {
	panic("unimplemented")
}

// PatchGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchGuestDriveByID(ctx context.Context, params operations.PatchGuestDriveByIDParams) operations.PatchGuestDriveByIDResponder {
	panic("unimplemented")
}

// PatchGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchGuestNetworkInterfaceByID(ctx context.Context, params operations.PatchGuestNetworkInterfaceByIDParams) operations.PatchGuestNetworkInterfaceByIDResponder {
	panic("unimplemented")
}

// PatchMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchMachineConfiguration(ctx context.Context, params operations.PatchMachineConfigurationParams) operations.PatchMachineConfigurationResponder {
	panic("unimplemented")
}

// PatchMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchMmds(ctx context.Context, params operations.PatchMmdsParams) operations.PatchMmdsResponder {
	panic("unimplemented")
}

// PatchVM implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PatchVM(ctx context.Context, params operations.PatchVMParams) operations.PatchVMResponder {
	panic("unimplemented")
}

// PutBalloon implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutBalloon(ctx context.Context, params operations.PutBalloonParams) operations.PutBalloonResponder {
	panic("unimplemented")
}

// PutCPUConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutCPUConfiguration(ctx context.Context, params operations.PutCPUConfigurationParams) operations.PutCPUConfigurationResponder {
	panic("unimplemented")
}

// PutEntropyDevice implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutEntropyDevice(ctx context.Context, params operations.PutEntropyDeviceParams) operations.PutEntropyDeviceResponder {
	panic("unimplemented")
}

// PutGuestBootSource implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestBootSource(ctx context.Context, params operations.PutGuestBootSourceParams) operations.PutGuestBootSourceResponder {
	panic("unimplemented")
}

// PutGuestDriveByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestDriveByID(ctx context.Context, params operations.PutGuestDriveByIDParams) operations.PutGuestDriveByIDResponder {
	panic("unimplemented")
}

// PutGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestNetworkInterfaceByID(ctx context.Context, params operations.PutGuestNetworkInterfaceByIDParams) operations.PutGuestNetworkInterfaceByIDResponder {
	panic("unimplemented")
}

// PutGuestVsock implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutGuestVsock(ctx context.Context, params operations.PutGuestVsockParams) operations.PutGuestVsockResponder {
	panic("unimplemented")
}

// PutLogger implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutLogger(ctx context.Context, params operations.PutLoggerParams) operations.PutLoggerResponder {
	panic("unimplemented")
}

// PutMachineConfiguration implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMachineConfiguration(ctx context.Context, params operations.PutMachineConfigurationParams) operations.PutMachineConfigurationResponder {
	panic("unimplemented")
}

// PutMetrics implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMetrics(ctx context.Context, params operations.PutMetricsParams) operations.PutMetricsResponder {
	panic("unimplemented")
}

// PutMmds implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMmds(ctx context.Context, params operations.PutMmdsParams) operations.PutMmdsResponder {
	panic("unimplemented")
}

// PutMmdsConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI) PutMmdsConfig(ctx context.Context, params operations.PutMmdsConfigParams) operations.PutMmdsConfigResponder {
	panic("unimplemented")
}
