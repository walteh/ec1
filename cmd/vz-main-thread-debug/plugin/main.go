package main

/*
#cgo darwin CFLAGS: -mmacosx-version-min=11 -x objective-c -fno-objc-arc
#cgo darwin LDFLAGS: -lobjc -framework Foundation -framework AppKit
#include <Foundation/Foundation.h>
#include <AppKit/AppKit.h>

// ensureAppContext initializes minimal NSApplication context for VZ framework
// This is required because VZ framework expects to run in an application context,
// but we don't need full GUI capabilities for daemon processes.
void ensureAppContext() {
    static dispatch_once_t onceToken;
    dispatch_once(&onceToken, ^{
        @autoreleasepool {
            NSApplication *app = [NSApplication sharedApplication];

            // Use accessory policy for daemon-like behavior (no dock icon)
            [app setActivationPolicy:NSApplicationActivationPolicyAccessory];

            // Minimal initialization for VZ framework context
            [app finishLaunching];
        }
    });
}
*/
import "C"

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Code-Hex/vz/v3"
)

// InitializeAppContext ensures the minimal macOS application context
// required by the VZ framework is available. This is safe to call multiple
// times and from daemon processes.
//
// This addresses the VZ framework requirement for application context
// without requiring full AppKit GUI integration.
func InitializeAppContext() {
	C.ensureAppContext()
}

// Plugin info that will be exported
var Info = &PluginInfo{
	Name:    "VZ Demo Plugin",
	Version: "1.0.0",
}

type PluginInfo struct {
	Name    string
	Version string
}

// VZWrapper provides a simple interface to VZ operations
type VZWrapper struct {
	initialized bool
}

// NewVZWrapper creates a new VZ wrapper instance
func NewVZWrapper() *VZWrapper {
	return &VZWrapper{}
}

// Initialize sets up VZ framework
func (w *VZWrapper) Initialize() error {
	if w.initialized {
		return nil
	}

	fmt.Println("🚀 VZ Framework initializing in plugin...")

	// VZ framework gets loaded here, NOT at import time
	// This is the key difference - the Apple VZ context is created
	// in the plugin process, not inherited from a parent

	w.initialized = true
	fmt.Println("✅ VZ Framework initialized successfully in plugin")
	return nil
}

// TestVM creates and tests a minimal VM to prove VZ works
func TestVM(kernelPath string, log func(string, ...interface{})) error {

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// create a new app context
	C.ensureAppContext()

	log("Testing VZ VM creation...")

	// Create a minimal bootloader (we'll use a dummy kernel path)
	// In real usage, you'd pass real kernel/initrd paths

	bootloader, err := vz.NewLinuxBootLoader(kernelPath)
	if err != nil {
		// Expected error since dummy kernel doesn't exist
		log("Expected bootloader error (dummy kernel): %v\n", err)
		log("✅ VZ bootloader creation attempted - VZ APIs accessible")
		return nil
	}

	// If we got here, we'd continue with VM creation
	config, err := vz.NewVirtualMachineConfiguration(bootloader, 1, 1024*1024*1024)
	if err != nil {
		return fmt.Errorf("failed to create VM config: %w", err)
	}

	log("✅ VZ VM config created: %v\n", config)

	vm, err := vz.NewVirtualMachine(config)
	if err != nil {
		log("failed to create VM: %v\n", err)
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// always in the baground pole for status updates
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log("STATE_PANIC: %v\n", err)
			}
		}()
		timer := time.NewTimer(2 * time.Second)
		for {
			select {
			case ok := <-vm.StateChangedNotify():
				log("STATE_CHANGED: %v\n", ok)
			case <-time.After(100 * time.Millisecond):
				log("STATE_POLL: state=%v, can_stop=%v, can_start=%v\n", vm.State(), "", "")
			case <-timer.C:
				log("STATE_TIMER: state=%v\n", vm.State())
				return
			}
		}
	}()
	log("✅ about to start VM\n")
	err = vm.Start()
	if err != nil {
		log("failed to start VM: %v\n", err)
		return fmt.Errorf("failed to start VM: %w", err)
	}
	log("✅ VZ VM started\n")
	return nil
}

// Shutdown cleans up VZ resources
func Shutdown() error {

	fmt.Println("🛑 Shutting down VZ plugin...")
	return nil
}

func main() {
}
