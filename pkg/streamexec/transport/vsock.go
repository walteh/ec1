package transport

import (
	"io"
	"net"

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

	// when listening we want to use our own contextid
	id, err := vsock.ContextID()
	if err != nil {
		return nil, errors.Errorf("vsock context id: %w", err)
	}
	t.contextID = id

	listener, err := vsock.ListenContextID(id, t.port, nil)
	if err != nil {
		return nil, errors.Errorf("vsock listen: %w", err)
	}
	return listener, nil
}
