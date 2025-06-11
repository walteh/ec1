package rest

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/walteh/ec1/pkg/vmm/vf/rest/define"
)

// ChangeState execute a state change (i.e. running to stopped)
func (vm *ControllableVirtualMachine) ChangeState(ctx context.Context, newState define.StateChange) error {
	var (
		response error
	)
	switch newState {
	case define.Pause:
		slog.DebugContext(ctx, "pausing virtual machine")
		response = vm.Pause(ctx)
	case define.Resume:
		slog.DebugContext(ctx, "resuming machine")
		response = vm.Resume(ctx)
	case define.Stop:
		slog.DebugContext(ctx, "stopping machine")
		_, response = vm.RequestStop(ctx)
	case define.HardStop:
		slog.DebugContext(ctx, "force stopping machine")
		response = vm.HardStop(ctx)
	default:
		return fmt.Errorf("invalid new VMState: %s", newState)
	}
	return response
}
