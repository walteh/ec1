package vf

import (
	"fmt"

	"github.com/walteh/ec1/pkg/virtio"
)

func (vmConfig *vzVirtioDeviceApplier) applyRosettaShare(dev *virtio.RosettaShare) error {
	return fmt.Errorf("rosetta is unsupported on non-arm64 platforms")
}
