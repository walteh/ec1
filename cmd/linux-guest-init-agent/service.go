package main

import (
	"context"

	"connectrpc.com/connect"

	"github.com/walteh/ec1/gen/proto/golang/ec1/guest/v1/guestv1connect"

	guestv1 "github.com/walteh/ec1/gen/proto/golang/ec1/guest/v1"
)

var _ guestv1connect.AgentServiceHandler = (*GuestAgentService)(nil)

type GuestAgentService struct {
}

func (s *GuestAgentService) ExecuteCommand(ctx context.Context, stream *connect.BidiStream[guestv1.ExecuteCommandRequest, guestv1.ExecuteCommandResponse]) error {

	return nil
}
