package gvnet

// virtio-net,unixSocketPath=/tmp/vfkit.sock,mac=5a:94:ef:e4:0c:ee

// func (me *GvproxyConfig) VirtioNetDevice(ctx context.Context) (*virtio.VirtioNet, error) {
// 	dev, err := virtio.VirtioNetNew(VIRTUAL_GUEST_MAC)
// 	if err != nil {
// 		return nil, errors.Errorf("creating virtio-net device: %w", err)
// 	}

// 	if strings.Contains(me.VMSocket.URL(), "://") {
// 		split := strings.Split(me.VMSocket.URL(), "://")

// 		slog.InfoContext(ctx, "setting unix socket path", "path", split[1])
// 		dev.SetUnixSocketPath(split[1])
// 	} else {
// 		dev.SetUnixSocketPath(me.VMSocket.URL())
// 	}

// 	return dev, nil
// }
