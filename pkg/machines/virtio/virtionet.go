package virtio

import (
	"context"
	"fmt"
	"net"
	"os"
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

	// UnixSocketPath string        `json:"unixSocketPath,omitempty"`
	LocalAddr *net.UnixAddr `json:"-"`

	// ReadyChan      chan struct{} `json:"readyChan,omitempty"`
}

type VirtioNetTransformable interface {
	TransformToVirtioNet(ctx context.Context) (*VirtioNet, error)
}

var _ VirtioDevice = &VirtioNet{}

// var _ VirtioNetTransformable = &VirtioNetViaUnixSocket{}

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

// func (dev *VirtioNet) SetUnixSocketPath(path string) {
// 	dev.UnixSocketPath = path
// 	dev.Nat = false
// }

func (dev *VirtioNet) validate() error {
	if dev.Nat && dev.Socket != nil {
		return fmt.Errorf("'nat' and 'fd' cannot be set at the same time")
	}
	// if dev.Nat && dev.UnixSocketPath != "" {
	// 	return fmt.Errorf("'nat' and 'unixSocketPath' cannot be set at the same time")
	// }
	// if dev.Socket != nil && dev.UnixSocketPath != "" {
	// 	return fmt.Errorf("'fd' and 'unixSocketPath' cannot be set at the same time")
	// }
	if !dev.Nat && dev.Socket == nil {
		return fmt.Errorf("one of 'nat' or 'fd' or 'unixSocketPath' must be set")
	}

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

// func (dev *VirtioNet) ToCmdLine() ([]string, error) {
// 	if err := dev.validate(); err != nil {
// 		return nil, err
// 	}

// 	builder := strings.Builder{}
// 	builder.WriteString("virtio-net")
// 	switch {
// 	case dev.Nat:
// 		builder.WriteString(",nat")
// 	case dev.UnixSocketPath != "":
// 		fmt.Fprintf(&builder, ",unixSocketPath=%s", dev.UnixSocketPath)
// 	default:
// 		fmt.Fprintf(&builder, ",fd=%d", dev.Socket.Fd())
// 	}

// 	if len(dev.MacAddress) != 0 {
// 		builder.WriteString(fmt.Sprintf(",mac=%s", dev.MacAddress))
// 	}

// 	return []string{"--device", builder.String()}, nil
// }

// func (dev *VirtioNet) FromOptions(options []option) error {
// 	for _, option := range options {
// 		switch option.key {
// 		case "nat":
// 			if option.value != "" {
// 				return fmt.Errorf("unexpected value for virtio-net 'nat' option: %s", option.value)
// 			}
// 			dev.Nat = true
// 		case "mac":
// 			macAddress, err := net.ParseMAC(option.value)
// 			if err != nil {
// 				return err
// 			}
// 			dev.MacAddress = macAddress
// 		case "fd":
// 			fd, err := strconv.Atoi(option.value)
// 			if err != nil {
// 				return err
// 			}
// 			dev.Socket = os.NewFile(uintptr(fd), "vfkit virtio-net socket")
// 		case "unixSocketPath":
// 			dev.UnixSocketPath = option.value
// 		default:
// 			return fmt.Errorf("unknown option for virtio-net devices: %s", option.key)
// 		}
// 	}

// 	return dev.validate()
// }
