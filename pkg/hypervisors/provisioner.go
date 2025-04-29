package hypervisors

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/networks/gvnet"
	"github.com/walteh/ec1/pkg/networks/gvnet/tapsock"
	"github.com/walteh/ec1/pkg/port"
	"gitlab.com/tozd/go/errors"
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
}

type GvproxyProvisioner struct {
	sock  tapsock.VMSocket
	chans chan struct{}
}

func parseHardwareAddr(mac string) (net.HardwareAddr, error) {
	macHardwareAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, errors.Errorf("parsing hardware address: %w", err)
	}
	return macHardwareAddr, nil
}

func (me *GvproxyProvisioner) device() *virtio.VirtioNet {

	netdev, err := virtio.VirtioNetNew(gvnet.VIRTUAL_GUEST_MAC)
	if err != nil {
		panic(err)
	}

	if strings.Contains(me.sock.URL(), "://") {
		split := strings.Split(me.sock.URL(), "://")
		netdev.SetUnixSocketPath(split[1])
	} else {
		netdev.SetUnixSocketPath(me.sock.URL())
	}

	if me.chans == nil {
		me.chans = make(chan struct{})
	}

	netdev.ReadyChan = me.chans

	return netdev
}

func (me *GvproxyProvisioner) RunDuringRuntime(ctx context.Context, vm VirtualMachine) error {
	// allocate a random port
	port, err := port.ReservePort(ctx)
	if err != nil {
		return errors.Errorf("reserving port: %w", err)
	}

	cfg := &gvnet.GvproxyConfig{
		VMSocket:           me.sock,
		VMHostPort:         fmt.Sprintf("tcp://127.0.0.1:%d", port),
		EnableDebug:        false,
		EnableStdioSocket:  false,
		EnableNoConnectAPI: true,
		ReadyChan:          me.chans,
	}

	slog.DebugContext(ctx, "running gvproxy provisioner", "config", cfg)

	return gvnet.Proxy(ctx, cfg)
}

func (me *GvproxyProvisioner) VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error) {
	return []virtio.VirtioDevice{
		me.device(),
	}, nil
}
