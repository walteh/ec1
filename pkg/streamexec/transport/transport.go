package transport

import (
	"io"
	"net"
)

// Transport is an interface for establishing connections
type Transport interface {
	// Dial establishes a connection
	Dial() (io.ReadWriteCloser, error)
	// Listen creates a listener for incoming connections
	Listen() (net.Listener, error)
}
