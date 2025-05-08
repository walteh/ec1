package virtio

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/crc-org/vfkit/pkg/util"
	"gitlab.com/tozd/go/errors"
)

// TODO: Add BridgedNetwork support
// https://github.com/Code-Hex/vz/blob/d70a0533bf8ed0fa9ab22fa4d4ca554b7c3f3ce5/network.go#L81-L82

// VirtioNet configures the virtual machine networking.
type VirtioNet struct {
	Nat        bool             `json:"nat"`
	MacAddress net.HardwareAddr `json:"-"` // custom marshaller in json.go
	// file parameter is holding a connected datagram socket.
	// see https://github.com/Code-Hex/vz/blob/7f648b6fb9205d6f11792263d79876e3042c33ec/network.go#L113-L155
	Socket *os.File `json:"socket,omitempty"`

	UnixSocketPath string        `json:"unixSocketPath,omitempty"`
	LocalAddr      *net.UnixAddr `json:"-"`
	ReadyChan      chan struct{} `json:"readyChan,omitempty"`
}

var _ VirtioDevice = &VirtioNet{}

func (v *VirtioNet) isVirtioDevice() {}

// VirtioNetNew creates a new network device for the virtual machine. It will
// use macAddress as its MAC address.
func VirtioNetNew(macAddress string) (*VirtioNet, error) {
	var hwAddr net.HardwareAddr

	if macAddress != "" {
		var err error
		if hwAddr, err = net.ParseMAC(macAddress); err != nil {
			return nil, err
		}
	}
	return &VirtioNet{
		Nat:        true,
		MacAddress: hwAddr,
	}, nil
}

func unixFd(fd uintptr) int {
	// On unix the underlying fd is int, overflow is not possible.
	return int(fd) //#nosec G115 -- potential integer overflow
}

// SetSocket Set the socket to use for the network communication
//
// This maps the virtual machine network interface to a connected datagram
// socket. This means all network traffic on this interface will go through
// file.
// file must be a connected datagram (SOCK_DGRAM) socket.
func (dev *VirtioNet) SetSocket(file *os.File) {
	dev.Socket = file
	dev.Nat = false
}

func (dev *VirtioNet) SetUnixSocketPath(path string) {
	dev.UnixSocketPath = path
	dev.Nat = false
}

func (dev *VirtioNet) validate() error {
	if dev.Nat && dev.Socket != nil {
		return fmt.Errorf("'nat' and 'fd' cannot be set at the same time")
	}
	if dev.Nat && dev.UnixSocketPath != "" {
		return fmt.Errorf("'nat' and 'unixSocketPath' cannot be set at the same time")
	}
	if dev.Socket != nil && dev.UnixSocketPath != "" {
		return fmt.Errorf("'fd' and 'unixSocketPath' cannot be set at the same time")
	}
	if !dev.Nat && dev.Socket == nil && dev.UnixSocketPath == "" {
		return fmt.Errorf("one of 'nat' or 'fd' or 'unixSocketPath' must be set")
	}

	return nil
}

func (dev *VirtioNet) ToCmdLine() ([]string, error) {
	if err := dev.validate(); err != nil {
		return nil, err
	}

	builder := strings.Builder{}
	builder.WriteString("virtio-net")
	switch {
	case dev.Nat:
		builder.WriteString(",nat")
	case dev.UnixSocketPath != "":
		fmt.Fprintf(&builder, ",unixSocketPath=%s", dev.UnixSocketPath)
	default:
		fmt.Fprintf(&builder, ",fd=%d", dev.Socket.Fd())
	}

	if len(dev.MacAddress) != 0 {
		builder.WriteString(fmt.Sprintf(",mac=%s", dev.MacAddress))
	}

	return []string{"--device", builder.String()}, nil
}

func (dev *VirtioNet) FromOptions(options []option) error {
	for _, option := range options {
		switch option.key {
		case "nat":
			if option.value != "" {
				return fmt.Errorf("unexpected value for virtio-net 'nat' option: %s", option.value)
			}
			dev.Nat = true
		case "mac":
			macAddress, err := net.ParseMAC(option.value)
			if err != nil {
				return err
			}
			dev.MacAddress = macAddress
		case "fd":
			fd, err := strconv.Atoi(option.value)
			if err != nil {
				return err
			}
			dev.Socket = os.NewFile(uintptr(fd), "vfkit virtio-net socket")
		case "unixSocketPath":
			dev.UnixSocketPath = option.value
		default:
			return fmt.Errorf("unknown option for virtio-net devices: %s", option.key)
		}
	}

	return dev.validate()
}
func localUnixSocketPath(dir string) (string, error) {
	// unix socket endpoints are filesystem paths, but their max length is
	// quite small (a bit over 100 bytes).
	// In this function we try to build a filename which is relatively
	// unique, not easily guessable (to prevent hostile collisions), and
	// short (`os.CreateTemp` filenames are a bit too long)
	//
	// os.Getpid() is unique but guessable. We append a short 16 bit random
	// number to it. We only use hex values to make the representation more
	// compact
	filename := filepath.Join(dir, fmt.Sprintf("vfkit-%x-%x.sock", os.Getpid(), rand.Int31n(0xffff))) //#nosec G404 -- no need for crypto/rand here

	tmpFile, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	// slightly racy, but hopefully this is in a directory only user-writable
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	return tmpFile.Name(), nil
}

// path for unixgram sockets must be less than 104 bytes on macOS
const maxUnixgramPathLen = 104

func (dev *VirtioNet) ConnectUnixPath(ctx context.Context) error {

	remoteAddr := net.UnixAddr{
		Name: dev.UnixSocketPath,
		Net:  "unixgram",
	}
	localSocketPath, err := localUnixSocketPath(filepath.Dir(dev.UnixSocketPath))
	if err != nil {
		return err
	}
	if len(localSocketPath) >= maxUnixgramPathLen {
		return fmt.Errorf("unixgram path '%s' is too long: %d >= %d bytes", localSocketPath, len(localSocketPath), maxUnixgramPathLen)
	}
	localAddr := net.UnixAddr{
		Name: localSocketPath,
		Net:  "unixgram",
	}
	if dev.ReadyChan != nil {
		slog.DebugContext(ctx, "waiting for virtio-net ready")
		<-dev.ReadyChan
		slog.DebugContext(ctx, "virtio-net ready")
	}

	// // create both ends of the socket
	// os.Remove(localSocketPath)
	// os.Create(localSocketPath)

	conn, err := net.DialUnix("unixgram", &localAddr, &remoteAddr)
	if err != nil {
		return errors.Errorf("dialing unix socket: %w", err)
	}
	defer conn.Close()

	rawConn, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	err = rawConn.Control(func(fd uintptr) {
		err := syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024)
		if err != nil {
			return
		}
		err = syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
		if err != nil {
			return
		}
	})
	if err != nil {
		return err
	}

	/* send vfkit magic so that the remote end can identify our connection attempt */
	if _, err := conn.Write([]byte("VFKT")); err != nil {
		return err
	}
	slog.InfoContext(ctx, "connected to unix socket", "local", conn.LocalAddr(), "remote", conn.RemoteAddr())

	fd, err := conn.File()
	if err != nil {
		return err
	}

	dev.Socket = fd
	dev.LocalAddr = &localAddr
	dev.UnixSocketPath = ""
	util.RegisterExitHandler(func() { _ = dev.Shutdown() })
	return nil
}

func (dev *VirtioNet) Shutdown() error {
	if dev.LocalAddr != nil {
		if err := os.Remove(dev.LocalAddr.Name); err != nil {
			return err
		}
	}
	if dev.Socket != nil {
		if err := dev.Socket.Close(); err != nil {
			return err
		}
	}

	return nil
}
