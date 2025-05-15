package hypervisors

import (
	"context"
	"net"

	"github.com/containers/common/pkg/strongunits"

	"github.com/walteh/ec1/pkg/machines/virtio"
)

type VirtualMachine interface {
	ID() string
	VSockConnect(ctx context.Context, port uint32) (net.Conn, error)
	VSockListen(ctx context.Context, port uint32) (net.Listener, error)
	CurrentState() VirtualMachineStateType
	StateChangeNotify(ctx context.Context) <-chan VirtualMachineStateChange
	ListenNetworkBlockDevices(ctx context.Context) error
	Devices() []virtio.VirtioDevice
	StartGraphicApplication(width float64, height float64) error

	// State Change
	Start(ctx context.Context) error
	HardStop(ctx context.Context) error
	RequestStop(ctx context.Context) (bool, error)
	Resume(ctx context.Context) error
	Pause(ctx context.Context) error

	// CanChangeState
	CanStart(ctx context.Context) bool
	CanHardStop(ctx context.Context) bool
	CanRequestStop(ctx context.Context) bool
	CanResume(ctx context.Context) bool
	CanPause(ctx context.Context) bool
	SaveFullSnapshot(ctx context.Context, path string) error
	// SaveDiffSnapshot(ctx context.Context) error
	RestoreFromFullSnapshot(ctx context.Context, path string) error

	Opts() *NewVMOptions

	// // Resources
	// MemoryUsage() strongunits.B
	// VCPUsUsage() float64
	// Resources(ctx context.Context) (*VirtualMachineResourceInfo, error)

	// Memory Balloon control
	SetMemoryBalloonTargetSize(ctx context.Context, targetBytes strongunits.B) error
	GetMemoryBalloonTargetSize(ctx context.Context) (strongunits.B, error)

	// // Uptime
	// Uptime() time.Duration
	// Time() time.Time
}

type VirtualMachineResourceInfo struct {
	MemoryUsed  strongunits.B `json:"memory_used"`
	MemoryTotal strongunits.B `json:"memory_total"`
	VCPUsUsed   float64       `json:"vcpus_used"`
	VCPUsTotal  float64       `json:"vcpus_total"`
}

// func prettyMemoryString(b strongunits.B) string {
// 	return fmt.Sprintf("%f MiB", float64(b.ToBytes())/float64(strongunits.MiB(1)))
// }

// func (v *VirtualMachineResourceInfo) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(map[string]interface{}{
// 		"memory_used":      v.MemoryUsed,
// 		"memory_total":     v.MemoryTotal,
// 		"memory_used_mib":  float64(v.MemoryUsed.ToBytes()) / float64(strongunits.MiB(1)),
// 		"memory_total_mib": float64(v.MemoryTotal.ToBytes()) / float64(strongunits.MiB(1)),
// 		"vcpus_used":       v.VCPUsUsed,
// 		"vcpus_total":      v.VCPUsTotal,
// 	})
// }
