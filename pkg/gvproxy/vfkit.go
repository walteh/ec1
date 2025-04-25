package gvproxy

import (
	"context"
	"strings"

	"github.com/crc-org/vfkit/pkg/config"
	"gitlab.com/tozd/go/errors"
)

// virtio-net,unixSocketPath=/tmp/vfkit.sock,mac=5a:94:ef:e4:0c:ee

func (me *VFKitSocket) Device(ctx context.Context) (config.VirtioDevice, error) {

	dev, err := config.VirtioNetNew("5a:94:ef:e4:0c:ee")
	if err != nil {
		return nil, errors.Errorf("creating virtio-net device: %w", err)
	}

	dev.SetUnixSocketPath(strings.TrimPrefix(me.Path, "unixgram://"))
	return dev, nil
}
