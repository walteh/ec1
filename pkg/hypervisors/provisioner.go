package hypervisors

import (
	"context"
	"net"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/networks/gvnet/tapsock"
)

type Provisioner interface {
	VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error)
}

type BootProvisioner interface {
	Provisioner
	RunDuringBoot(ctx context.Context, vm VirtualMachine) error
}

type RuntimeProvisioner interface {
	Provisioner
	RunDuringRuntime(ctx context.Context, vm VirtualMachine) error
	ReadyChan() <-chan struct{}
}

type GvproxyProvisioner struct {
	sock  tapsock.VMSocket
	chans chan struct{}
	dev   *virtio.VirtioNet
	portz int
}

func parseHardwareAddr(mac string) (net.HardwareAddr, error) {
	macHardwareAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, errors.Errorf("parsing hardware address: %w", err)
	}
	return macHardwareAddr, nil
}

// var _ Provisioner = &GvproxyProvisioner{}

// func NewGvproxyProvisioner(sock tapsock.VMSocket) *GvproxyProvisioner {
// 	return &GvproxyProvisioner{
// 		sock: sock,
// 	}
// }

// func (me *GvproxyProvisioner) port(ctx context.Context) (int, error) {
// 	if me.portz != 0 {
// 		return me.portz, nil
// 	}
// 	port, err := port.ReservePort(ctx)
// 	if err != nil {
// 		return 0, errors.Errorf("reserving port: %w", err)
// 	}
// 	me.portz = int(port)
// 	return me.portz, nil
// }

// func (me *GvproxyProvisioner) SSHURL(ctx context.Context) (string, error) {
// 	port, err := me.port(ctx)
// 	if err != nil {
// 		return "", errors.Errorf("getting port: %w", err)
// 	}
// 	return fmt.Sprintf("tcp://127.0.0.1:%d", port), nil
// }

// func (me *GvproxyProvisioner) ReadyChan() <-chan struct{} {
// 	return me.chans
// }

// func (me *GvproxyProvisioner) device() *virtio.VirtioNet {
// 	if me.dev != nil {
// 		return me.dev
// 	}

// 	netdev, err := virtio.VirtioNetNew(gvnet.VIRTUAL_GUEST_MAC)
// 	if err != nil {
// 		panic(err)
// 	}

// 	if strings.Contains(me.sock.URL(), "://") {
// 		split := strings.Split(me.sock.URL(), "://")
// 		netdev.SetUnixSocketPath(split[1])
// 	} else {
// 		netdev.SetUnixSocketPath(me.sock.URL())
// 	}

// 	if me.chans == nil {
// 		me.chans = make(chan struct{})
// 	}

// 	netdev.ReadyChan = me.chans
// 	me.dev = netdev
// 	return netdev
// }

// func (me *GvproxyProvisioner) RunDuringRuntime(ctx context.Context, _ VirtualMachine) error {
// 	// allocate a random port
// 	ss, err := me.SSHURL(ctx)
// 	if err != nil {
// 		return errors.Errorf("getting ssh url: %w", err)
// 	}

// 	cfg := &gvnet.GvproxyConfig{
// 		// VMSocket:           me.sock,
// 		VMHostPort:         ss,
// 		EnableDebug:        false,
// 		EnableStdioSocket:  false,
// 		EnableNoConnectAPI: true,
// 		ReadyChan:          me.chans,
// 	}

// 	slog.DebugContext(ctx, "running gvproxy provisioner", "config", cfg)

// 	return gvnet.Proxy(ctx, cfg)
// }

// func (me *GvproxyProvisioner) VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error) {

// 	dev := me.device()

// 	return []virtio.VirtioDevice{
// 		dev,
// 	}, nil
// }
