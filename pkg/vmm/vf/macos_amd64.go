package vf

import (
	"fmt"

	"github.com/Code-Hex/vz/v3"

	"github.com/walteh/ec1/pkg/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

func NewMacPlatformConfiguration(_, _, _ string) (vz.PlatformConfiguration, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}

func toVzMacOSBootloader(_ *vmm.MacOSBootloader) (vz.BootLoader, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}

func newMacGraphicsDeviceConfiguration(_ *virtio.VirtioGPU) (vz.GraphicsDeviceConfiguration, error) {
	return nil, fmt.Errorf("running macOS guests is only supported on ARM devices")
}
