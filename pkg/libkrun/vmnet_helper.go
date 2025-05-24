//go:build darwin && vmnet_helper && cgo && libkrun

package libkrun

/*
#cgo pkg-config: libkrun
#include <stdlib.h>
#include "libkrun.h"
*/
import "C"
import (
	"context"
	"log/slog"
	"unsafe"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/vmnet"
)

// SetVMNetNetwork configures networking using vmnet (requires entitlement)
// This provides better performance than the default TSI backend when vmnet entitlement is available
func (c *Context) SetVMNetNetwork(ctx context.Context, config VMNetConfig) error {
	log := slog.With(slog.String("component", "libkrun-vmnet"), slog.Uint64("ctx_id", uint64(c.id)))
	log.InfoContext(ctx, "configuring vmnet networking (requires entitlement)")

	// Check if vmnet-helper is available
	if !vmnet.HelperAvailable() {
		return errors.Errorf("vmnet-helper not available at %s", "/opt/vmnet-helper/bin/vmnet-helper")
	}

	// Create socket pair for vmnet communication
	sock1, sock2, err := vmnet.Socketpair()
	if err != nil {
		return errors.Errorf("creating vmnet socket pair: %w", err)
	}
	defer sock1.Close()
	// sock2 will be passed to libkrun

	// Configure vmnet helper options
	helperOptions := vmnet.HelperOptions{
		Fd:              sock1,
		Verbose:         config.Verbose,
		OperationMode:   config.OperationMode,
		EnableIsolation: config.EnableIsolation != nil && *config.EnableIsolation,
	}

	// Set interface ID if specified, otherwise generate from VM context
	if config.InterfaceID != nil {
		helperOptions.InterfaceID = *config.InterfaceID
	} else {
		// Generate a UUID from the context ID for consistent MAC addresses
		helperOptions.InterfaceID = vmnet.UUIDFromName(string(rune(c.id)))
	}

	// Set network addressing
	if config.StartAddress != nil {
		helperOptions.StartAddress = *config.StartAddress
	}
	if config.EndAddress != nil {
		helperOptions.EndAddress = *config.EndAddress
	}
	if config.SubnetMask != nil {
		helperOptions.SubnetMask = *config.SubnetMask
	}

	// Set shared interface for bridged mode
	if config.OperationMode == vmnet.OperationModeBridged {
		if config.SharedInterface == nil {
			return errors.Errorf("shared interface required for bridged mode")
		}
		helperOptions.SharedInterface = *config.SharedInterface
	}

	// Start vmnet helper
	helper := vmnet.NewHelper(helperOptions)
	if err := helper.Start(); err != nil {
		sock2.Close()
		return errors.Errorf("starting vmnet helper: %w", err)
	}

	// Get interface info
	interfaceInfo := helper.InterfaceInfo()
	if interfaceInfo == nil {
		sock2.Close()
		helper.Stop()
		return errors.Errorf("vmnet interface info not available")
	}

	log.InfoContext(ctx, "vmnet interface created",
		slog.String("interface_id", interfaceInfo.InterfaceID),
		slog.String("mac_address", interfaceInfo.MACAddress),
		slog.String("start_address", interfaceInfo.StartAddress),
		slog.String("end_address", interfaceInfo.EndAddress),
		slog.Uint64("mtu", uint64(interfaceInfo.MTU)))

	// Pass the socket to libkrun for networking
	sock2FD := int(sock2.Fd())
	result := C.krun_set_passt_fd(C.uint32_t(c.id), C.int(sock2FD))
	if result < 0 {
		sock2.Close()
		helper.Stop()
		return errors.Errorf("setting vmnet socket for libkrun: %d", result)
	}

	// Configure port mapping if specified
	if len(config.PortMap) > 0 {
		log.DebugContext(ctx, "setting port mapping", slog.Any("port_map", config.PortMap))

		// Use the same fixed port mapping logic as SetNetwork
		cPortMap := make([]*C.char, len(config.PortMap)+1)
		for i, port := range config.PortMap {
			cPortMap[i] = C.CString(port)
			defer C.free(unsafe.Pointer(cPortMap[i]))
		}
		cPortMap[len(config.PortMap)] = nil

		var portMapPtr **C.char
		if len(cPortMap) > 0 {
			portMapPtr = &cPortMap[0]
		} else {
			nullPtr := (*C.char)(nil)
			portMapPtr = &nullPtr
		}

		result := C.krun_set_port_map(C.uint32_t(c.id), portMapPtr)
		if result < 0 {
			sock2.Close()
			helper.Stop()
			return errors.Errorf("setting port map: %d", result)
		}
	}

	// Store helper for cleanup (would need to add to Context struct)
	// For now, we'll let the helper run and clean up when the process exits
	// In production, you'd want to properly manage the helper lifecycle

	log.InfoContext(ctx, "vmnet networking configured successfully")
	return nil
}
