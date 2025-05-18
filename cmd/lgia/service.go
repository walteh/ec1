package main

// import (
// 	"context"

// 	"connectrpc.com/connect"

// 	"github.com/walteh/ec1/gen/proto/golang/ec1/guest/v1/guestv1connect"

// 	guestv1 "github.com/walteh/ec1/gen/proto/golang/ec1/guest/v1"
// )

// var _ guestv1connect.AgentServiceHandler = (*GuestAgentService)(nil)

// type GuestAgentService struct {
// }

// func (s *GuestAgentService) ExecuteCommand(ctx context.Context, stream *connect.BidiStream[guestv1.ExecuteCommandRequest, guestv1.ExecuteCommandResponse]) error {

// 	return nil
// }

// func serveGrpcVsock(ctx context.Context, port int, service *GuestAgentService) error {
// 	route, handler := guestv1connect.NewAgentServiceHandler(service)

// 	server, err := buildGRPCServer(ctx, route, handler)
// 	if err != nil {
// 		return err
// 	}

// 	listener, err := vsock.Listen(uint32(port), nil)
// 	if err != nil {
// 		return err
// 	}

// 	err = server.Serve(listener)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func buildGRPCServer(ctx context.Context, route string, handler http.Handler) (*http.Server, error) {

// 	mux := http.NewServeMux()
// 	mux.Handle(route, handler)

// 	httpServer := &http2.Server{}

// 	// Mount some handlers here.
// 	server := &http.Server{
// 		Addr:    ":http",
// 		Handler: h2c.NewHandler(mux, httpServer),
// 		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
// 			return ctx
// 		},
// 		BaseContext: func(l net.Listener) context.Context {
// 			return ctx
// 		},
// 		// Don't forget timeouts!
// 	}

// 	return server, nil

// }
