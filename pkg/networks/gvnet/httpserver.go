package gvnet

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/soheilhy/cmux"
	"github.com/superblocksteam/run"
	"gitlab.com/tozd/go/errors"
)

type httpServer struct {
	runningServer *http.Server
	ln            net.Listener
	mux           http.Handler
	name          string
	running       bool
}

func NewHTTPServer(name string, mux http.Handler, ln net.Listener) *httpServer {
	return &httpServer{
		mux:  mux,
		ln:   ln,
		name: name,
	}
}

var _ run.Runnable = (*httpServer)(nil)

func (me *httpServer) Run(ctx context.Context) error {
	s := &http.Server{
		Handler:      me.mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	me.runningServer = s
	me.running = true
	defer func() {
		me.running = false
	}()
	return s.Serve(me.ln)
}

func (me *httpServer) Close(ctx context.Context) error {
	<-ctx.Done()
	return me.runningServer.Shutdown(ctx)
}

func (me *httpServer) Alive() bool {
	return me.running
}

func (me *httpServer) Fields() []slog.Attr {
	return []slog.Attr{slog.String("http_server_name", me.name)}
}

func (me *httpServer) Name() string {
	return me.name
}

type cmuxServer struct {
	mux     cmux.CMux
	name    string
	running bool
}

func NewCmuxServer(name string, mux cmux.CMux) *cmuxServer {

	return &cmuxServer{
		mux:  mux,
		name: name,
	}
}

var _ run.Runnable = (*cmuxServer)(nil)

func (me *cmuxServer) Run(ctx context.Context) error {
	me.running = true
	defer func() {
		me.running = false
	}()
	err := me.mux.Serve()
	if err != nil {
		return errors.Errorf("serving cmux: %w", err)
	}
	return nil
}

func (me *cmuxServer) Close(ctx context.Context) error {
	<-ctx.Done()
	me.mux.Close()
	return nil
}

func (me *cmuxServer) Alive() bool {
	return me.running
}

func (me *cmuxServer) Fields() []slog.Attr {
	return []slog.Attr{slog.String("cmux_server_name", me.name)}
}

func (me *cmuxServer) Name() string {
	return me.name
}
