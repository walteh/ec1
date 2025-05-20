package transport

import (
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"gitlab.com/tozd/go/errors"
)

// VSockTransport implements Transport for VSOCK connections
type VSockTransport struct {
	contextID uint32
	port      uint32
}

// NewVSockTransport creates a new VSockTransport
func NewVSockTransport(contextID, port uint32) *VSockTransport {
	return &VSockTransport{
		contextID: contextID,
		port:      port,
	}
}

// Dial establishes a VSOCK connection
func (t *VSockTransport) Dial() (io.ReadWriteCloser, error) {
	conn, err := vsock.Dial(t.contextID, t.port, nil)
	if err != nil {
		return nil, errors.Errorf("vsock dial: %w", err)
	}
	return conn, nil
}

// ListenVsock creates a VSOCK listener on the given CID and port
func (t *VSockTransport) Listen() (net.Listener, error) {

	log.Printf("Listening on vsock context id %d port %d", 3, t.port)

	listener, err := vsock.ListenContextID(3, t.port, nil)
	if err != nil {
		// retry for 10 seconds if socket address family not supported by protocol
		for strings.Contains(err.Error(), "address family not supported by protocol") {
			listener, err = vsock.ListenContextID(3, t.port, nil)
			if err != nil {
				time.Sleep(1 * time.Second)
			} else {
				log.Printf("retried vsock listen")
				return listener, nil
			}
		}

		return nil, errors.Errorf("vsock listen: %w", err)
	}
	return listener, nil
}
