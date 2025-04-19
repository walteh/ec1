package hypervisor

import (
	"context"
	"runtime"

	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

// Driver defines the interface for hypervisor implementations
type Driver interface {
	// StartVM starts a virtual machine
	StartVM(ctx context.Context, request *ec1v1.StartVMRequest) (*ec1v1.StartVMResponse, error)

	// StopVM stops a running virtual machine
	StopVM(ctx context.Context, request *ec1v1.StopVMRequest) (*ec1v1.StopVMResponse, error)

	// GetVMStatus gets the status of a virtual machine
	GetVMStatus(ctx context.Context, request *ec1v1.GetVMStatusRequest) (*ec1v1.GetVMStatusResponse, error)

	// GetHypervisorType returns the type of hypervisor this driver manages
	GetHypervisorType() ec1v1.HypervisorType
}

// NewDriver creates a hypervisor driver based on the host platform
func NewDriver(ctx context.Context) (Driver, error) {
	// Detect platform and return appropriate driver
	if isMacOS() {
		return NewAppleDriver(ctx)
	}

	// Assume Linux for POC
	return NewKVMDriver(ctx)

}

// isMacOS returns true if the host is running macOS
func isMacOS() bool {
	return runtime.GOOS == "darwin"
}
