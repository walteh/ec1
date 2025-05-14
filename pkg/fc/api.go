package fc

import (
	"context"
	"log/slog"

	"github.com/go-openapi/swag"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/models"
	"github.com/walteh/ec1/gen/firecracker-swagger-go/restapi/operations"
	"github.com/walteh/ec1/pkg/hypervisors"
)

// EC1FirecrackerAPI implements the Firecracker API using the hypervisors package.
// It provides a translation layer between Firecracker API operations and hypervisor operations
// for a single Firecracker microVM instance.
type EC1FirecrackerAPI[V hypervisors.VirtualMachine] struct {
	hpv             hypervisors.Hypervisor[V]
	vm              V    // The single VM instance managed by this API
	isVMInitialized bool // Tracks if the underlying VM in hypervisor has been created
	// TODO: Add vmConfig *vmConfigState for pre-boot configurations
	// TODO: Add a mutex for concurrent access to vm, vmConfig, and isVMInitialized
}

// NewEC1FirecrackerAPI creates a new Firecracker API implementation
// that leverages the hypervisors package for VM operations for a single microVM.
func NewEC1FirecrackerAPI[V hypervisors.VirtualMachine](hpv hypervisors.Hypervisor[V]) *EC1FirecrackerAPI[V] {
	return &EC1FirecrackerAPI[V]{
		hpv:             hpv,
		// vm is initially nil (zero value for interface/type V)
		isVMInitialized: false,
	}
}

var _ operations.FirecrackerAPI = &EC1FirecrackerAPI[hypervisors.VirtualMachine]{}

// mapHypervisorStateToInstanceInfoState maps hypervisor VM states to Firecracker InstanceInfo states.
// InstanceInfo states are: "Not started", "Running", "Paused".
func mapHypervisorStateToInstanceInfoState(vmState hypervisors.VirtualMachineStateType) string {
	switch vmState {
	case hypervisors.VirtualMachineStateTypeRunning:
		return models.InstanceInfoStateRunning
	case hypervisors.VirtualMachineStateTypePaused:
		return models.InstanceInfoStatePaused
	case hypervisors.VirtualMachineStateTypeStarting,
		hypervisors.VirtualMachineStateTypeStopping,
		hypervisors.VirtualMachineStateTypeStopped,
		hypervisors.VirtualMachineStateTypeError,
		hypervisors.VirtualMachineStateTypeUnknown:
		return models.InstanceInfoStateNotStarted
	default:
		slog.Warn("mapHypervisorStateToInstanceInfoState: unknown hypervisor state, defaulting to NotStarted", "hypervisor_state", vmState)
		return models.InstanceInfoStateNotStarted // Default fallback
	}
}

// DescribeInstance implements operations.FirecrackerAPI.
// It returns general information about the Firecracker microVM instance.
func (f *EC1FirecrackerAPI[V]) DescribeInstance(ctx context.Context, params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {
	const instanceID = "fc-instance-01"
	const appName = "ec1-firecracker"
	// Reuse the version from GetFirecrackerVersion for consistency, or make it truly dynamic.
	const vmmVersion = "1.13.0-ec1-custom"

	var currentVMState string

	if f.isVMInitialized { // Rely on isVMInitialized to indicate f.vm is valid.
		// If isVMInitialized is true, f.vm is assumed to be a valid instance.
		hypervisorState := f.vm.CurrentState()
		currentVMState = mapHypervisorStateToInstanceInfoState(hypervisorState)
	} else {
		currentVMState = models.InstanceInfoStateNotStarted
	}

	payload := &models.InstanceInfo{
		ID:           swag.String(instanceID),
		AppName:      swag.String(appName),
		State:        swag.String(currentVMState),
		VmmVersion:   swag.String(vmmVersion),
	}

	slog.InfoContext(ctx, "Describing instance", "id", instanceID, "app_name", appName, "state", currentVMState, "vmm_version", vmmVersion)
	return operations.NewDescribeInstanceOK().WithPayload(payload)
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

// GetExportVMConfig implements operations.FirecrackerAPI.
func (f *EC1FirecrackerAPI[V]) GetExportVMConfig(ctx context.Context, params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
	return operations.GetExportVMConfigNotImplemented()
}

// GetFirecrackerVersion implements operations.FirecrackerAPI.
// It returns the version of this Firecracker API implementation.
func (f *EC1FirecrackerAPI[V]) GetFirecrackerVersion(ctx context.Context, params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {
	const implementationVersion = "1.13.0-ec1-custom"

	payload := &models.FirecrackerVersion{
		FirecrackerVersion: swag.String(implementationVersion),
	}

	return operations.NewGetFirecrackerVersionOK().WithPayload(payload)
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
