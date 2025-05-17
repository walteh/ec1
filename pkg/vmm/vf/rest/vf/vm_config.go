package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/walteh/ec1/pkg/hypervisors"
	"github.com/walteh/ec1/pkg/hypervisors/vf/rest/define"
)

type ControllableVirtualMachine struct {
	hypervisors.VirtualMachine
}

func NewControllableVirtualMachine(vm hypervisors.VirtualMachine) *ControllableVirtualMachine {
	return &ControllableVirtualMachine{vm}
}

// Inspect returns information about the virtual machine like hw resources
// and devices
func (vm *ControllableVirtualMachine) Inspect(c *gin.Context) {
	// resources, err := vm.Resources(c.Request.Context())
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	c.JSON(http.StatusOK, gin.H{
		"state": vm.CurrentState(),
	})
}

// GetVMState retrieves the current vm state
func (vm *ControllableVirtualMachine) GetVMState(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"id": vm.ID(),

		"time": time.Now().Format(time.RFC3339),
		// "uptime":      vm.Uptime().String(),
		"state":       vm.CurrentState(),
		"canStart":    vm.CanStart(c.Request.Context()),
		"canPause":    vm.CanPause(c.Request.Context()),
		"canResume":   vm.CanResume(c.Request.Context()),
		"canStop":     vm.CanRequestStop(c.Request.Context()),
		"canHardStop": vm.CanHardStop(c.Request.Context()),
	})
}

// SetVMState requests a state change on a virtual machine.  At this time only
// the following states are valid:
// Pause - pause a running machine
// Resume - resume a paused machine
// Stop - stops a running machine
// HardStop - forceably stops a running machine
func (vm *ControllableVirtualMachine) SetVMState(ctx context.Context, c *gin.Context) {
	var (
		s define.VMState
	)

	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := vm.ChangeState(ctx, define.StateChange(s.State))
	if response != nil {
		logrus.Errorf("failed action %s: %q", s.State, response)
		c.JSON(http.StatusInternalServerError, gin.H{"error": response.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}
