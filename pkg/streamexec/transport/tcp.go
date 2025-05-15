// Package transport defines interfaces and implementations for communication transports
package transport

import (
	"io"
	"net"

	"gitlab.com/tozd/go/errors"
)

// TCPTransport implements Transport for TCP connections
type TCPTransport struct {
	address string
}

// NewTCPTransport creates a new TCPTransport
func NewTCPTransport(address string) *TCPTransport {
	return &TCPTransport{
		address: address,
	}
}

// Dial establishes a TCP connection
func (t *TCPTransport) Dial() (io.ReadWriteCloser, error) {
	conn, err := net.Dial("tcp", t.address)
	if err != nil {
		return nil, errors.Errorf("tcp dial: %w", err)
	}
	return conn, nil
}

// Listen creates a listener for incoming connections
func (t *TCPTransport) Listen() (net.Listener, error) {
	listener, err := net.Listen("tcp", t.address)
	if err != nil {
		return nil, errors.Errorf("tcp listen: %w", err)
	}
	return listener, nil
}
