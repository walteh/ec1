// Code generated by protoc-gen-go-ttrpc. DO NOT EDIT.
// source: harpoon/v1/harpoon.proto
package harpoonv1

import (
	context "context"
	ttrpc "github.com/containerd/ttrpc"
)

type TTRPCGuestServiceService interface {
	Exec(context.Context, TTRPCGuestService_ExecServer) error
}

type TTRPCGuestService_ExecServer interface {
	Send(*ExecResponse) error
	Recv() (*ExecRequest, error)
	ttrpc.StreamServer
}

type ttrpcguestserviceExecServer struct {
	ttrpc.StreamServer
}

func (x *ttrpcguestserviceExecServer) Send(m *ExecResponse) error {
	return x.StreamServer.SendMsg(m)
}

func (x *ttrpcguestserviceExecServer) Recv() (*ExecRequest, error) {
	m := new(ExecRequest)
	if err := x.StreamServer.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func RegisterTTRPCGuestServiceService(srv *ttrpc.Server, svc TTRPCGuestServiceService) {
	srv.RegisterService("harpoon.v1.GuestService", &ttrpc.ServiceDesc{
		Streams: map[string]ttrpc.Stream{
			"Exec": {
				Handler: func(ctx context.Context, stream ttrpc.StreamServer) (interface{}, error) {
					return nil, svc.Exec(ctx, &ttrpcguestserviceExecServer{stream})
				},
				StreamingClient: true,
				StreamingServer: true,
			},
		},
	})
}

type TTRPCGuestServiceClient interface {
	Exec(context.Context) (TTRPCGuestService_ExecClient, error)
}

type ttrpcguestserviceClient struct {
	client *ttrpc.Client
}

func NewTTRPCGuestServiceClient(client *ttrpc.Client) TTRPCGuestServiceClient {
	return &ttrpcguestserviceClient{
		client: client,
	}
}

func (c *ttrpcguestserviceClient) Exec(ctx context.Context) (TTRPCGuestService_ExecClient, error) {
	stream, err := c.client.NewStream(ctx, &ttrpc.StreamDesc{
		StreamingClient: true,
		StreamingServer: true,
	}, "harpoon.v1.GuestService", "Exec", nil)
	if err != nil {
		return nil, err
	}
	x := &ttrpcguestserviceExecClient{stream}
	return x, nil
}

type TTRPCGuestService_ExecClient interface {
	Send(*ExecRequest) error
	Recv() (*ExecResponse, error)
	ttrpc.ClientStream
}

type ttrpcguestserviceExecClient struct {
	ttrpc.ClientStream
}

func (x *ttrpcguestserviceExecClient) Send(m *ExecRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *ttrpcguestserviceExecClient) Recv() (*ExecResponse, error) {
	m := new(ExecResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}
