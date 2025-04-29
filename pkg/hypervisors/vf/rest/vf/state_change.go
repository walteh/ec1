package rest

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/walteh/ec1/pkg/hypervisors/vf/rest/define"
)

// ChangeState execute a state change (i.e. running to stopped)
func (vm *ControllableVirtualMachine) ChangeState(ctx context.Context, newState define.StateChange) error {
	var (
		response error
	)
	switch newState {
	case define.Pause:
		logrus.Debug("pausing virtual machine")
		response = vm.Pause(ctx)
	case define.Resume:
		logrus.Debug("resuming machine")
		response = vm.Resume(ctx)
	case define.Stop:
		logrus.Debug("stopping machine")
		_, response = vm.RequestStop(ctx)
	case define.HardStop:
		logrus.Debug("force stopping machine")
		response = vm.HardStop(ctx)
	default:
		return fmt.Errorf("invalid new VMState: %s", newState)
	}
	return response
}
