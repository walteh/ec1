// Package transport defines interfaces and implementations for communication transports
package transport

import (
	"io"
	"net"
	"time"

	"gitlab.com/tozd/go/errors"
)

var _ Transport = &InMemoryTransport{}

// InMemoryTransport implements Transport for in-memory connections, useful for testing
type InMemoryTransport struct {
	client io.ReadWriteCloser
	server io.ReadWriteCloser
}

// NewInMemoryTransport creates a new InMemoryTransport with connected pipes
func NewInMemoryTransport() (*InMemoryTransport, io.ReadWriteCloser) {
	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()

	client := &pipeReadWriteCloser{
		reader: clientReader,
		writer: clientWriter,
	}

	server := &pipeReadWriteCloser{
		reader: serverReader,
		writer: serverWriter,
	}

	return &InMemoryTransport{
		client: client,
		server: server,
	}, server
}

// Dial returns the client side of the in-memory connection
func (t *InMemoryTransport) Dial() (io.ReadWriteCloser, error) {
	if t.client == nil {
		return nil, errors.New("connection already used")
	}

	conn := t.client
	t.client = nil // Ensure connection can only be used once
	return conn, nil
}

// pipeReadWriteCloser combines a PipeReader and PipeWriter into a ReadWriteCloser
type pipeReadWriteCloser struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func (p *pipeReadWriteCloser) Read(data []byte) (int, error) {
	return p.reader.Read(data)
}

func (p *pipeReadWriteCloser) Write(data []byte) (int, error) {
	return p.writer.Write(data)
}

func (p *pipeReadWriteCloser) Close() error {
	err1 := p.reader.Close()
	err2 := p.writer.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

func (t *InMemoryTransport) Listen() (net.Listener, error) {
	// create a fake listener
	return &fakeListener{
		conn: t.server,
	}, nil
}

type fakeListener struct {
	conn io.ReadWriteCloser
}

func (l *fakeListener) Accept() (net.Conn, error) {
	return &fakeConn{
		ReadWriteCloser: l.conn,
	}, nil
}

func (l *fakeListener) Close() error {
	return nil
}

func (l *fakeListener) Addr() net.Addr {
	return nil
}

var _ net.Conn = &fakeConn{}

type fakeConn struct {
	io.ReadWriteCloser
}

func (c *fakeConn) LocalAddr() net.Addr {
	return nil
}

func (c *fakeConn) RemoteAddr() net.Addr {
	return nil
}

func (c *fakeConn) SetDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *fakeConn) SetWriteDeadline(t time.Time) error {
	return nil
}
