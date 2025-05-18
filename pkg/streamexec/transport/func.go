package transport

import (
	"io"
	"net"
)

type FunctionTransport struct {
	dialer   func() (io.ReadWriteCloser, error)
	listener func() (net.Listener, error)
}

func NewFunctionTransport(dialer func() (io.ReadWriteCloser, error), listener func() (net.Listener, error)) *FunctionTransport {
	return &FunctionTransport{
		dialer:   dialer,
		listener: listener,
	}
}

func (t *FunctionTransport) Dial() (io.ReadWriteCloser, error) {
	return t.dialer()
}

func (t *FunctionTransport) Listen() (net.Listener, error) {
	return t.listener()
}
