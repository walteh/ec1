package qemuguestagent

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/apex/log"

	"github.com/walteh/ec1/pkg/machines/virtio"
	"github.com/walteh/ec1/pkg/vmm"
)

const (
	QEMU_GUEST_AGENT_SOCKET_PORT = 1024
)

var _ vmm.RuntimeProvisioner = &QemuGuestAgentTimesyncProvisioner{}

type QemuGuestAgentTimesyncProvisioner struct {
}

func (me *QemuGuestAgentTimesyncProvisioner) device() *virtio.VirtioVsock {
	return &virtio.VirtioVsock{
		Port:      QEMU_GUEST_AGENT_SOCKET_PORT,
		SocketURL: "", // no proxied socket path
		Direction: virtio.VirtioVsockDirectionGuestConnectsAsClient,
	}
}

func (me *QemuGuestAgentTimesyncProvisioner) RunDuringRuntime(ctx context.Context, vm vmm.VirtualMachine) error {

	var vsockConn net.Conn

	go func() {
		<-ctx.Done()
		if vsockConn != nil {
			if err := vsockConn.Close(); err != nil {
				slog.WarnContext(ctx, "failed to shutdown ignition server", "error", err)
			}
		}
	}()

	sleepNotifierCh := StartSleepNotifier()
	for activity := range sleepNotifierCh {
		log.Debugf("Sleep notification: %s", activity)
		if activity == SleepNotifierActivityTypeAwake {
			log.Infof("machine awake")
			if vsockConn == nil {
				var err error
				vsockConn, err = vmm.ConnectVsock(ctx, vm, me.device())
				if err != nil {
					log.Debugf("error connecting to vsock port %d: %v", QEMU_GUEST_AGENT_SOCKET_PORT, err)
					break
				}
			}
			if err := SyncGuestTime(ctx, vsockConn); err != nil {
				log.Debugf("error syncing guest time: %v", err)
			}
		}
	}

	slog.DebugContext(ctx, "qemu guest agent timesync done")

	return nil
}

func (me *QemuGuestAgentTimesyncProvisioner) ReadyChan() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (me *QemuGuestAgentTimesyncProvisioner) VirtioDevices(ctx context.Context) ([]virtio.VirtioDevice, error) {
	return []virtio.VirtioDevice{me.device()}, nil
}

func SyncGuestTime(ctx context.Context, conn net.Conn) error {
	qemugaCmdTemplate := `{"execute": "guest-set-time", "arguments":{"time": %d}}` + "\n"
	qemugaCmd := fmt.Sprintf(qemugaCmdTemplate, time.Now().UnixNano())

	slog.DebugContext(ctx, "syncing guest time", "command", qemugaCmd)
	_, err := conn.Write([]byte(qemugaCmd))
	if err != nil {
		return err
	}
	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}

	if response != `{"return": {}}`+"\n" {
		return fmt.Errorf("Unexpected response from qemu-guest-agent: %s", response)
	}

	return nil
}
