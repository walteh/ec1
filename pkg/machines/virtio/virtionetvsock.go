package virtio

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"

	"gitlab.com/tozd/go/errors"
)

//

// VirtioNet configures the virtual machine networking.
type VirtioNetViaUnixSocket struct {
	MacAddress net.HardwareAddr `json:"-"` // custom marshaller in json.go
	// file parameter is holding a connected datagram socket.
	// see https://github.com/Code-Hex/vz/blob/7f648b6fb9205d6f11792263d79876e3042c33ec/network.go#L113-L155
	// Socket *os.File `json:"socket,omitempty"`
	conn net.Addr
	// UnixSocketPath string `json:"unixSocketPath,omitempty"`
	// LocalAddr      *net.UnixAddr `json:"-"`
	ReadyChan chan struct{} `json:"readyChan,omitempty"`
}

func NewVirtioNetViaUnixSocket(macAddress string, conn net.Addr) (*VirtioNetViaUnixSocket, error) {
	// parse the mac address
	macAddresd, err := net.ParseMAC(macAddress)
	if err != nil {
		return nil, errors.Errorf("parsing mac address: %w", err)
	}

	dev := &VirtioNetViaUnixSocket{
		MacAddress: macAddresd,
		conn:       conn,
	}

	return dev, nil
}

// path for unixgram sockets must be less than 104 bytes on macOS
const MaxUnixgramPathLen = 104

// func (dev *VirtioNetViaUnixSocket) TransformToVirtioNet(ctx context.Context) (*VirtioNet, error) {

// 	if dev.ReadyChan != nil {
// 		slog.DebugContext(ctx, "waiting for virtio-net ready")
// 		<-dev.ReadyChan
// 		slog.DebugContext(ctx, "virtio-net ready")
// 	}

// 	remoteAddr := net.UnixAddr{
// 		Name: dev.conn.RemoteAddr().String(),
// 		Net:  "unixgram",
// 	}
// 	localSocketPath, err := LocalUnixSocketPath(filepath.Dir(dev.conn.LocalAddr().String()))
// 	if err != nil {
// 		return nil, errors.Errorf("creating local unix socket path: %w", err)
// 	}
// 	if len(localSocketPath) >= MaxUnixgramPathLen {
// 		return nil, errors.Errorf("unixgram path '%s' is too long: %d >= %d bytes", localSocketPath, len(localSocketPath), MaxUnixgramPathLen)
// 	}

// 	localAddr := net.UnixAddr{
// 		Name: localSocketPath,
// 		Net:  "unixgram",
// 	}

// 	if _, err := os.Stat(localSocketPath); !os.IsNotExist(err) {
// 		return nil, errors.Errorf("local unix socket path '%s' already exists", localSocketPath)
// 	}

// 	if _, err := os.Stat(dev.UnixSocketPath); os.IsNotExist(err) {
// 		return nil, errors.Errorf("remote unix socket path '%s' does not exist", dev.UnixSocketPath)
// 	}

// 	conn, err := net.DialUnix("unixgram", &localAddr, &remoteAddr)
// 	if err != nil {
// 		return nil, errors.Errorf("dialing unix socket: %w", err)
// 	}
// 	defer conn.Close()

// 	rawConn, err := conn.SyscallConn()
// 	if err != nil {
// 		return nil, errors.Errorf("getting syscall conn: %w", err)
// 	}
// 	err = rawConn.Control(func(fd uintptr) {
// 		err := syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 1*1024*1024)
// 		if err != nil {
// 			return
// 		}
// 		err = syscall.SetsockoptInt(unixFd(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
// 		if err != nil {
// 			return
// 		}
// 	})
// 	if err != nil {
// 		return nil, errors.Errorf("setting socket options: %w", err)
// 	}

// 	/* send vfkit magic so that the remote end can identify our connection attempt */
// 	if _, err := conn.Write([]byte("VFKT")); err != nil {
// 		return nil, errors.Errorf("writing magic: %w", err)
// 	}

// 	slog.InfoContext(ctx, "connected to unix socket", "local", conn.LocalAddr(), "remote", conn.RemoteAddr())

// 	fd, err := conn.File()
// 	if err != nil {
// 		return nil, errors.Errorf("getting file: %w", err)
// 	}

// 	virtioNet := &VirtioNet{
// 		Nat:        false,
// 		MacAddress: dev.MacAddress,
// 		Socket:     fd,
// 		LocalAddr:  &localAddr,
// 	}

// 	util.RegisterExitHandler(func() { _ = virtioNet.Shutdown() })
// 	return virtioNet, nil
// }

func LocalUnixSocketPath(dir string) (string, error) {
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
