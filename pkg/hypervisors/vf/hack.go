package vf

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/walteh/ec1/pkg/hack"
)

var (
	// Framework handles
	virtualizationFramework uintptr

	// Classes and selectors
	vzVirtualMachineClass                       objc.Class
	vzVirtioTraditionalMemoryBalloonDeviceClass objc.Class
	nsNumberClass                               objc.Class
	nsArrayClass                                objc.Class

	// Method selectors
	sel_memoryBalloonDevices              objc.SEL
	sel_setTargetVirtualMachineMemorySize objc.SEL
	sel_targetVirtualMachineMemorySize    objc.SEL
	sel_count                             objc.SEL
	sel_objectAtIndex                     objc.SEL
	sel_numberWithUnsignedLongLong        objc.SEL
	sel_numberWithInt                     objc.SEL
	sel_unsignedLongLongValue             objc.SEL
)

func init() {
	// Load the Virtualization framework
	var err error
	virtualizationFramework, err = purego.Dlopen("/System/Library/Frameworks/Virtualization.framework/Virtualization", purego.RTLD_LAZY)
	if err != nil {
		// Handle error
		panic(err)
	}

	// Get classes
	vzVirtualMachineClass = objc.GetClass("VZVirtualMachine")
	vzVirtioTraditionalMemoryBalloonDeviceClass = objc.GetClass("VZVirtioTraditionalMemoryBalloonDevice")
	nsNumberClass = objc.GetClass("NSNumber")
	nsArrayClass = objc.GetClass("NSArray")

	// Get selectors
	sel_memoryBalloonDevices = objc.RegisterName("memoryBalloonDevices")
	sel_setTargetVirtualMachineMemorySize = objc.RegisterName("setTargetVirtualMachineMemorySize:")
	sel_targetVirtualMachineMemorySize = objc.RegisterName("targetVirtualMachineMemorySize")
	sel_count = objc.RegisterName("count")
	sel_objectAtIndex = objc.RegisterName("objectAtIndex:")
	sel_numberWithUnsignedLongLong = objc.RegisterName("numberWithUnsignedLongLong:")
	sel_numberWithInt = objc.RegisterName("numberWithInt:")
	sel_unsignedLongLongValue = objc.RegisterName("unsignedLongLongValue")
}

// VirtioTraditionalMemoryBalloonDevice represents a Virtio traditional memory balloon device
type VirtioTraditionalMemoryBalloonDevice struct {
	id objc.ID
}

func (v *VirtualMachine) objcPtr() unsafe.Pointer {
	return hack.GetUnexportedFieldOf(v.vzvm, "_ptr").(unsafe.Pointer)
}

// MemoryBalloonDevices returns the list of memory balloon devices for a VM
// Returns an error if the operation fails
func (v *VirtualMachine) MemoryBalloonDevices() ([]*VirtioTraditionalMemoryBalloonDevice, error) {
	objPtr := v.objcPtr()
	if objPtr == nil {
		return nil, fmt.Errorf("invalid virtual machine object")
	}

	// The NSAutoreleasePool is managed automatically by purego/objc
	// Don't create one manually

	// Call the Objective-C method
	nsArray := objc.ID(objPtr).Send(sel_memoryBalloonDevices)
	if nsArray == 0 {
		return nil, fmt.Errorf("failed to get memory balloon devices from VM")
	}

	// No need to manually retain/release with purego/objc
	// It handles object lifetime automatically

	// Get the count - this is where the crash occurs
	// Convert count using explicit type conversion
	count := int(uint(nsArray.Send(sel_count)))
	devices := make([]*VirtioTraditionalMemoryBalloonDevice, count)

	for i := 0; i < count; i++ {
		// Pass index as int - purego/objc handles type conversion
		device := nsArray.Send(sel_objectAtIndex, i)
		if device == 0 {
			return nil, fmt.Errorf("failed to get memory balloon device at index %d", i)
		}

		// No need to manually retain the device
		devices[i] = &VirtioTraditionalMemoryBalloonDevice{
			id: device,
		}
	}

	return devices, nil
}

// SetTargetVirtualMachineMemorySize sets the target memory size for the balloon device
// Returns an error if the operation fails
func (v *VirtioTraditionalMemoryBalloonDevice) SetTargetVirtualMachineMemorySize(size uint64) error {
	if v.id == 0 {
		return fmt.Errorf("invalid memory balloon device object")
	}

	// Create an NSNumber with the size value
	sizeObj := objc.ID(nsNumberClass).Send(sel_numberWithUnsignedLongLong, size)
	if sizeObj == 0 {
		return fmt.Errorf("failed to create NSNumber for memory size %d", size)
	}

	// Set the target memory size
	v.id.Send(sel_setTargetVirtualMachineMemorySize, sizeObj)
	return nil
}

// GetTargetVirtualMachineMemorySize retrieves the current target memory size for the balloon device
// Returns an error if the operation fails
func (v *VirtioTraditionalMemoryBalloonDevice) GetTargetVirtualMachineMemorySize() (uint64, error) {
	if v.id == 0 {
		return 0, fmt.Errorf("invalid memory balloon device object")
	}

	// Call targetVirtualMachineMemorySize to get the NSNumber object
	respNumber := v.id.Send(sel_targetVirtualMachineMemorySize)
	if respNumber == 0 {
		return 0, fmt.Errorf("failed to get target memory size from device")
	}

	// Get the uint64 value from the NSNumber
	value := uint64(objc.ID(respNumber).Send(sel_unsignedLongLongValue))
	return value, nil
}
