package rest

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/walteh/ec1/pkg/hypervisors/vf"
	"github.com/walteh/ec1/pkg/hypervisors/vf/rest/define"
)

type VzVirtualMachine struct {
	*vf.VirtualMachine
}

func NewVzVirtualMachine(vm *vf.VirtualMachine) *VzVirtualMachine {
	return &VzVirtualMachine{vm}
}

// Inspect returns information about the virtual machine like hw resources
// and devices
func (vm *VzVirtualMachine) Inspect(c *gin.Context) {
	c.JSON(http.StatusOK, vm.Config())
}

// GetVMState retrieves the current vm state
func (vm *VzVirtualMachine) GetVMState(c *gin.Context) {
	current := vm.State()
	c.JSON(http.StatusOK, gin.H{
		"state":       current.String(),
		"canStart":    vm.CanStart(),
		"canPause":    vm.CanPause(),
		"canResume":   vm.CanResume(),
		"canStop":     vm.CanRequestStop(),
		"canHardStop": vm.CanStop(),
	})
}

// SetVMState requests a state change on a virtual machine.  At this time only
// the following states are valid:
// Pause - pause a running machine
// Resume - resume a paused machine
// Stop - stops a running machine
// HardStop - forceably stops a running machine
func (vm *VzVirtualMachine) SetVMState(c *gin.Context) {
	var (
		s define.VMState
	)

	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := vm.ChangeState(define.StateChange(s.State))
	if response != nil {
		logrus.Errorf("failed action %s: %q", s.State, response)
		c.JSON(http.StatusInternalServerError, gin.H{"error": response.Error()})
		return
	}
	c.Status(http.StatusAccepted)
}
