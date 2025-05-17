package fc

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/containers/common/pkg/strongunits"
	"github.com/go-openapi/swag"
	"github.com/rs/xid"
	"gitlab.com/tozd/go/errors"

	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/models"
	"github.com/walteh/ec1/gen/firecracker-swagger-go/restapi/operations"
	"github.com/walteh/ec1/pkg/machines/host"
	"github.com/walteh/ec1/pkg/vmm"
)

const (
	appName    = "firecracker-ec1"
	vmmVersion = "0.0.1-ec1-custom"
)

func ptr[T any](v T) *T { return &v }

// FirecrackerMicroVM implements the Firecracker API using the hypervisors package.
// It provides a translation layer between Firecracker API operations and hypervisor operations
// for a single Firecracker microVM instance.
type FirecrackerMicroVM[V vmm.VirtualMachine] struct {
	vm              V    // The single VM instance managed by this API
	isVMInitialized bool // Tracks if the underlying VM in hypervisor has been created
	// TODO: Add vmConfig *vmConfigState for pre-boot configurations
	// TODO: Add a mutex for concurrent access to vm, vmConfig, and isVMInitialized
	instanceID string
	mu         sync.RWMutex
	cfg        apiConfig
}

type apiConfig struct {
	guestDrives   map[string]*models.PartialDrive
	guestNetworks map[string]*models.PartialNetworkInterface
	machineConfig *models.MachineConfiguration
	mmds          models.MmdsContentsObject
	balloon       *models.Balloon
	vm            *models.VM
}

// NewFirecrackerMicroVM creates a new Firecracker API implementation
// that leverages the hypervisors package for VM operations for a single microVM.
func NewFirecrackerMicroVM[V vmm.VirtualMachine](ctx context.Context, hpv vmm.Hypervisor[V], vmi vmm.VMIProvider) (*FirecrackerMicroVM[V], error) {
	id := "mvm-" + xid.New().String()

	lvmi, ok := vmi.(vmm.LinuxVMIProvider)
	if !ok {
		slog.Error("vmi is not a LinuxVMIProvider")
		return nil, errors.New("vmi is not a LinuxVMIProvider")
	}

	cdf, err := host.EmphiricalVMCacheDir(ctx, id)
	if err != nil {
		return nil, errors.Errorf("getting empirical VM cache dir: %w", err)
	}

	bootloader := lvmi.BootLoaderConfig(cdf)

	vm, err := hpv.NewVirtualMachine(ctx, id, vmm.NewVMOptions{}, bootloader)
	if err != nil {
		slog.Error("failed to create new VM", "error", err)
		return nil, err
	}

	return &FirecrackerMicroVM[V]{
		vm:              vm,
		isVMInitialized: true,
		instanceID:      id,
	}, nil
}

var _ operations.FirecrackerAPI = &FirecrackerMicroVM[vmm.VirtualMachine]{}

// mapHypervisorStateToInstanceInfoState maps hypervisor VM states to Firecracker InstanceInfo states.
// InstanceInfo states are: "Not started", "Running", "Paused".
func mapHypervisorStateToInstanceInfoState(vmState vmm.VirtualMachineStateType) string {
	switch vmState {
	case vmm.VirtualMachineStateTypeRunning:
		return models.InstanceInfoStateRunning
	case vmm.VirtualMachineStateTypePaused:
		return models.InstanceInfoStatePaused
	case vmm.VirtualMachineStateTypeStarting,
		vmm.VirtualMachineStateTypeStopping,
		vmm.VirtualMachineStateTypeStopped,
		vmm.VirtualMachineStateTypeError,
		vmm.VirtualMachineStateTypeUnknown:
		return models.InstanceInfoStateNotStarted
	default:
		slog.Warn("mapHypervisorStateToInstanceInfoState: unknown hypervisor state, defaulting to NotStarted", "hypervisor_state", vmState)
		return models.InstanceInfoStateNotStarted // Default fallback
	}
}

// DescribeInstance implements operations.FirecrackerAPI.
// It returns general information about the Firecracker microVM instance.
func (f *FirecrackerMicroVM[V]) DescribeInstance(ctx context.Context, params operations.DescribeInstanceParams) operations.DescribeInstanceResponder {

	var currentVMState string

	if f.isVMInitialized { // Rely on isVMInitialized to indicate f.vm is valid.
		// If isVMInitialized is true, f.vm is assumed to be a valid instance.
		hypervisorState := f.vm.CurrentState()
		currentVMState = mapHypervisorStateToInstanceInfoState(hypervisorState)
	} else {
		currentVMState = models.InstanceInfoStateNotStarted
	}

	ctx = slogctx.With(ctx, slog.String("instance_id", f.instanceID), slog.String("app_name", appName), slog.String("state", currentVMState), slog.String("vmm_version", vmmVersion))

	payload := &models.InstanceInfo{
		ID:         swag.String(f.instanceID),
		AppName:    swag.String(appName),
		State:      swag.String(currentVMState),
		VmmVersion: swag.String(vmmVersion),
	}

	slog.InfoContext(ctx, "describing instance")

	return operations.NewDescribeInstanceOK().WithPayload(payload)
}

// CreateSnapshot implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) CreateSnapshot(ctx context.Context, params operations.CreateSnapshotParams) operations.CreateSnapshotResponder {

	if params.Body.SnapshotType == "Diff" {
		return operations.NewCreateSnapshotBadRequest().WithPayload(&models.Error{
			FaultMessage: "diff snapshots are not supported",
		})
	}

	if *params.Body.MemFilePath != "" {
		slog.Warn("mem_file_path is not supported, ignoring", "mem_file_path", *params.Body.MemFilePath)
	}

	err := f.vm.SaveFullSnapshot(ctx, *params.Body.SnapshotPath)
	if err != nil {
		slog.Error("failed to create snapshot", "error", err)
		return operations.NewCreateSnapshotDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: err.Error(),
		})
	}

	slog.InfoContext(ctx, "created snapshot")

	return operations.NewCreateSnapshotNoContent()
}

// CreateSyncAction implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) CreateSyncAction(ctx context.Context, params operations.CreateSyncActionParams) operations.CreateSyncActionResponder {
	return operations.CreateSyncActionNotImplemented()
}

// DescribeBalloonConfig implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) DescribeBalloonConfig(ctx context.Context, params operations.DescribeBalloonConfigParams) operations.DescribeBalloonConfigResponder {
	return operations.DescribeBalloonConfigNotImplemented()
}

// DescribeBalloonStats implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) DescribeBalloonStats(ctx context.Context, params operations.DescribeBalloonStatsParams) operations.DescribeBalloonStatsResponder {

	trg, err := f.vm.GetMemoryBalloonTargetSize(ctx)
	if err != nil {
		slog.Error("failed to get memory balloon actual size", "error", err)
		return operations.NewDescribeBalloonStatsDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: err.Error(),
		})
	}

	payload := &models.BalloonStats{
		ActualMib:          ptr(int64(strongunits.ToMib(trg))),
		TargetMib:          ptr(int64(strongunits.ToMib(trg))),
		ActualPages:        ptr(int64(0)),
		TargetPages:        ptr(int64(0)),
		AvailableMemory:    int64(0),
		DiskCaches:         int64(0),
		FreeMemory:         int64(0),
		HugetlbAllocations: int64(0),
		HugetlbFailures:    int64(0),
		MajorFaults:        int64(0),
		MinorFaults:        int64(0),
		SwapIn:             int64(0),
		SwapOut:            int64(0),
		TotalMemory:        int64(0),
	}

	return operations.NewDescribeBalloonStatsOK().WithPayload(payload)
}

// GetExportVMConfig implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) GetExportVMConfig(ctx context.Context, params operations.GetExportVMConfigParams) operations.GetExportVMConfigResponder {
	return operations.GetExportVMConfigNotImplemented()
}

// GetFirecrackerVersion implements operations.FirecrackerAPI.
// It returns the version of this Firecracker API implementation.
func (f *FirecrackerMicroVM[V]) GetFirecrackerVersion(ctx context.Context, params operations.GetFirecrackerVersionParams) operations.GetFirecrackerVersionResponder {

	payload := &models.FirecrackerVersion{
		FirecrackerVersion: swag.String(vmmVersion),
	}

	return operations.NewGetFirecrackerVersionOK().WithPayload(payload)
}

// GetMachineConfiguration implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) GetMachineConfiguration(ctx context.Context, params operations.GetMachineConfigurationParams) operations.GetMachineConfigurationResponder {
	mem, err := f.vm.GetMemoryBalloonTargetSize(ctx)
	if err != nil {
		slog.Error("failed to get memory size", "error", err)
		return operations.NewGetMachineConfigurationDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: err.Error(),
		})
	}

	payload := &models.MachineConfiguration{
		CPUTemplate:     models.CPUTemplateNone.Pointer(),
		HugePages:       models.MachineConfigurationHugePagesNone, // not sure if mac can support M2 hugepages
		VcpuCount:       ptr(int64(1)),                            // 1 or even number <= 32
		MemSizeMib:      ptr(int64(strongunits.ToMib(mem))),
		Smt:             ptr(false), // simultaneous multithreading - can only enabled for x86_64
		TrackDirtyPages: ptr(false),
	}

	return operations.NewGetMachineConfigurationOK().WithPayload(payload)
}

// GetMmds implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) GetMmds(ctx context.Context, params operations.GetMmdsParams) operations.GetMmdsResponder {
	// TODO: what
	payload := map[string]interface{}{
		"enabled": false,
	}

	return operations.NewGetMmdsOK().WithPayload(payload)
}

// LoadSnapshot implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) LoadSnapshot(ctx context.Context, params operations.LoadSnapshotParams) operations.LoadSnapshotResponder {

	// TODO: what
	// payload := map[string]interface{}{
	// 	"enabled": false,
	// }

	if params.Body.EnableDiffSnapshots {
		return operations.NewLoadSnapshotBadRequest().WithPayload(&models.Error{
			FaultMessage: "diff snapshots are not supported",
		})
	}

	if params.Body.MemBackend != nil {
		return operations.NewLoadSnapshotBadRequest().WithPayload(&models.Error{
			FaultMessage: "mem_backend is not supported, use mem_file_path instead",
		})
	}

	if params.Body.MemFilePath != "" {
		slog.Warn("mem_file_path is not supported, ignoring", "mem_file_path", params.Body.MemFilePath)
	}

	if len(params.Body.NetworkOverrides) > 0 {
		slog.Warn("network_overrides is not supported, ignoring", "network_overrides", params.Body.NetworkOverrides)
	}

	err := f.vm.RestoreFromFullSnapshot(ctx, *params.Body.SnapshotPath)
	if err != nil {
		slog.Error("failed to load snapshot", "error", err)
		return operations.NewLoadSnapshotDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: "failed to load snapshot: " + err.Error(),
		})
	}

	if err := vmm.WaitForVMState(ctx, f.vm, vmm.VirtualMachineStateTypePaused, time.After(10*time.Second)); err != nil {
		slog.Error("failed to wait for vm to be paused", "error", err)
		return operations.NewLoadSnapshotDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: "waiting for snapshot to be restored: " + err.Error(),
		})
	}

	if params.Body.ResumeVM {
		err = f.vm.Start(ctx)
		if err != nil {
			slog.Error("failed to start vm", "error", err)
			return operations.NewLoadSnapshotDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				FaultMessage: "failed to start vm: " + err.Error(),
			})
		}

		if err := vmm.WaitForVMState(ctx, f.vm, vmm.VirtualMachineStateTypeRunning, time.After(10*time.Second)); err != nil {
			slog.Error("failed to wait for vm to be running", "error", err)
			return operations.NewLoadSnapshotDefault(http.StatusInternalServerError).WithPayload(&models.Error{
				FaultMessage: "failed to wait for vm to be running: " + err.Error(),
			})
		}
	}

	// attrs := []slog.Attr{
	// 	slog.String("snapshot_path", *params.Body.SnapshotPath),
	// 	slog.Bool("snapshot_mode", params.Body.ResumeVM),
	// 	slog.Bool("snapshot_version", params.Body.EnableDiffSnapshots),
	// }

	// for i, override := range params.Body.NetworkOverrides {
	// 	attrs = append(attrs, slog.String(fmt.Sprintf("network_override[%d]_iface_id", i), *override.IfaceID))
	// 	attrs = append(attrs, slog.String(fmt.Sprintf("network_override[%d]_host_dev_name", i), *override.HostDevName))
	// }

	slog.InfoContext(ctx, "snapshot loaded successfully")

	return operations.NewLoadSnapshotNoContent()
}

// PatchBalloon implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchBalloon(ctx context.Context, params operations.PatchBalloonParams) operations.PatchBalloonResponder {

	amountMib := *params.Body.AmountMib
	amountBytes := strongunits.MiB(amountMib).ToBytes()
	err := f.vm.SetMemoryBalloonTargetSize(ctx, amountBytes)
	if err != nil {
		slog.ErrorContext(ctx, "failed to set memory balloon target size", "error", err)
		return operations.NewPatchBalloonDefault(http.StatusInternalServerError).WithPayload(&models.Error{
			FaultMessage: err.Error(),
		})
	}

	slog.InfoContext(ctx, "patched balloon", slog.Int64("amount_mib", amountMib))

	return operations.NewPatchBalloonNoContent()
}

// PatchBalloonStatsInterval implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchBalloonStatsInterval(ctx context.Context, params operations.PatchBalloonStatsIntervalParams) operations.PatchBalloonStatsIntervalResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cfg.balloon == nil {
		f.cfg.balloon = &models.Balloon{}
	}

	f.cfg.balloon.StatsPollingIntervals = *params.Body.StatsPollingIntervals

	return operations.NewPatchBalloonStatsIntervalNoContent()
}

// PatchGuestDriveByID implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchGuestDriveByID(ctx context.Context, params operations.PatchGuestDriveByIDParams) operations.PatchGuestDriveByIDResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cfg.guestDrives == nil {
		f.cfg.guestDrives = make(map[string]*models.PartialDrive)
	}

	f.cfg.guestDrives[params.DriveID] = params.Body

	slog.InfoContext(ctx, "patched guest drive", slog.String("drive_id", params.DriveID))

	return operations.NewPatchGuestDriveByIDNoContent()
}

// PatchGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchGuestNetworkInterfaceByID(ctx context.Context, params operations.PatchGuestNetworkInterfaceByIDParams) operations.PatchGuestNetworkInterfaceByIDResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cfg.guestNetworks == nil {
		f.cfg.guestNetworks = make(map[string]*models.PartialNetworkInterface)
	}

	f.cfg.guestNetworks[params.IfaceID] = params.Body

	slog.InfoContext(ctx, "patched guest network interface", slog.String("iface_id", params.IfaceID))

	return operations.NewPatchGuestNetworkInterfaceByIDNoContent()
}

// PatchMachineConfiguration implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchMachineConfiguration(ctx context.Context, params operations.PatchMachineConfigurationParams) operations.PatchMachineConfigurationResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cfg.machineConfig = params.Body

	slog.InfoContext(ctx, "patched machine configuration", slog.Any("machine_config", f.cfg.machineConfig))

	return operations.NewPatchMachineConfigurationNoContent()
}

// PatchMmds implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchMmds(ctx context.Context, params operations.PatchMmdsParams) operations.PatchMmdsResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cfg.mmds = params.Body

	slog.InfoContext(ctx, "patched mmds", slog.Any("mmds", f.cfg.mmds))

	return operations.NewPatchMmdsNoContent()
}

// PatchVM implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PatchVM(ctx context.Context, params operations.PatchVMParams) operations.PatchVMResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cfg.vm = params.Body

	slog.InfoContext(ctx, "patched vm", slog.Any("vm", f.cfg.vm))

	return operations.NewPatchVMNoContent()
}

// PutBalloon implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutBalloon(ctx context.Context, params operations.PutBalloonParams) operations.PutBalloonResponder {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cfg.balloon = params.Body

	slog.InfoContext(ctx, "put balloon", slog.Any("balloon", f.cfg.balloon))

	return operations.NewPutBalloonNoContent()
}

// PutCPUConfiguration implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutCPUConfiguration(ctx context.Context, params operations.PutCPUConfigurationParams) operations.PutCPUConfigurationResponder {
	return operations.PutCPUConfigurationNotImplemented()
}

// PutEntropyDevice implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutEntropyDevice(ctx context.Context, params operations.PutEntropyDeviceParams) operations.PutEntropyDeviceResponder {
	return operations.PutEntropyDeviceNotImplemented()
}

// PutGuestBootSource implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutGuestBootSource(ctx context.Context, params operations.PutGuestBootSourceParams) operations.PutGuestBootSourceResponder {
	return operations.PutGuestBootSourceNotImplemented()
}

// PutGuestDriveByID implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutGuestDriveByID(ctx context.Context, params operations.PutGuestDriveByIDParams) operations.PutGuestDriveByIDResponder {
	return operations.PutGuestDriveByIDNotImplemented()
}

// PutGuestNetworkInterfaceByID implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutGuestNetworkInterfaceByID(ctx context.Context, params operations.PutGuestNetworkInterfaceByIDParams) operations.PutGuestNetworkInterfaceByIDResponder {
	return operations.PutGuestNetworkInterfaceByIDNotImplemented()
}

// PutGuestVsock implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutGuestVsock(ctx context.Context, params operations.PutGuestVsockParams) operations.PutGuestVsockResponder {
	return operations.PutGuestVsockNotImplemented()
}

// PutLogger implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutLogger(ctx context.Context, params operations.PutLoggerParams) operations.PutLoggerResponder {
	return operations.PutLoggerNotImplemented()
}

// PutMachineConfiguration implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutMachineConfiguration(ctx context.Context, params operations.PutMachineConfigurationParams) operations.PutMachineConfigurationResponder {
	return operations.PutMachineConfigurationNotImplemented()
}

// PutMetrics implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutMetrics(ctx context.Context, params operations.PutMetricsParams) operations.PutMetricsResponder {
	return operations.PutMetricsNotImplemented()
}

// PutMmds implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutMmds(ctx context.Context, params operations.PutMmdsParams) operations.PutMmdsResponder {
	return operations.PutMmdsNotImplemented()
}

// PutMmdsConfig implements operations.FirecrackerAPI.
func (f *FirecrackerMicroVM[V]) PutMmdsConfig(ctx context.Context, params operations.PutMmdsConfigParams) operations.PutMmdsConfigResponder {
	return operations.PutMmdsConfigNotImplemented()
}
