package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/mdlayher/vsock"

	"github.com/walteh/ec1/gen/proto/golang/ec1/guest/v1/guestv1connect"
)

const (
	vsockPort     = 2019
	vsockExecPath = "/usr/local/bin/vsock_exec"
	realInitPath  = "/sbin/init.real"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := serveVsock(ctx, vsockPort, &GuestAgentService{})
		if err != nil {
			log.Fatalf("Failed to serve vsock: %v", err)
		}
	}()

	// Execute the original init process
	if err := syscall.Exec(realInitPath, os.Args, os.Environ()); err != nil {
		log.Fatalf("Failed to exec original init: %v", err)
	}

	<-ctx.Done()
}

func serveVsock(ctx context.Context, port int, service *GuestAgentService) error {
	route, handler := guestv1connect.NewAgentServiceHandler(service)

	server, err := buildGRPCServer(ctx, route, handler)
	if err != nil {
		return err
	}

	listener, err := vsock.Listen(uint32(port), nil)
	if err != nil {
		return err
	}

	err = server.Serve(listener)
	if err != nil {
		return err
	}

	return nil
}

func buildGRPCServer(ctx context.Context, route string, handler http.Handler) (*http.Server, error) {

	mux := http.NewServeMux()
	mux.Handle(route, handler)

	httpServer := &http2.Server{}

	// Mount some handlers here.
	server := &http.Server{
		Addr:    ":http",
		Handler: h2c.NewHandler(mux, httpServer),
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return ctx
		},
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		// Don't forget timeouts!
	}

	return server, nil

}
