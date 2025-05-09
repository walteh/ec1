// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: ec1/v1poc1/management.proto

package v1poc1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ManagementService_IdentifyRemoteAgent_FullMethodName               = "/ec1.v1poc1.ManagementService/IdentifyRemoteAgent"
	ManagementService_InitializeLocalAgentInsideLocalVM_FullMethodName = "/ec1.v1poc1.ManagementService/InitializeLocalAgentInsideLocalVM"
	ManagementService_InitilizeRemoteAgent_FullMethodName              = "/ec1.v1poc1.ManagementService/InitilizeRemoteAgent"
)

// ManagementServiceClient is the client API for ManagementService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// ManagementService provides functions for agents to register and report status
type ManagementServiceClient interface {
	IdentifyRemoteAgent(ctx context.Context, in *IdentifyRemoteAgentRequest, opts ...grpc.CallOption) (*IdentifyRemoteAgentResponse, error)
	InitializeLocalAgentInsideLocalVM(ctx context.Context, in *InitializeLocalAgentInsideLocalVMRequest, opts ...grpc.CallOption) (*InitializeLocalAgentInsideLocalVMResponse, error)
	InitilizeRemoteAgent(ctx context.Context, in *InitializeRemoteAgentRequest, opts ...grpc.CallOption) (*InitializeRemoteAgentResponse, error)
}

type managementServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewManagementServiceClient(cc grpc.ClientConnInterface) ManagementServiceClient {
	return &managementServiceClient{cc}
}

func (c *managementServiceClient) IdentifyRemoteAgent(ctx context.Context, in *IdentifyRemoteAgentRequest, opts ...grpc.CallOption) (*IdentifyRemoteAgentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(IdentifyRemoteAgentResponse)
	err := c.cc.Invoke(ctx, ManagementService_IdentifyRemoteAgent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementServiceClient) InitializeLocalAgentInsideLocalVM(ctx context.Context, in *InitializeLocalAgentInsideLocalVMRequest, opts ...grpc.CallOption) (*InitializeLocalAgentInsideLocalVMResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitializeLocalAgentInsideLocalVMResponse)
	err := c.cc.Invoke(ctx, ManagementService_InitializeLocalAgentInsideLocalVM_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *managementServiceClient) InitilizeRemoteAgent(ctx context.Context, in *InitializeRemoteAgentRequest, opts ...grpc.CallOption) (*InitializeRemoteAgentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(InitializeRemoteAgentResponse)
	err := c.cc.Invoke(ctx, ManagementService_InitilizeRemoteAgent_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ManagementServiceServer is the server API for ManagementService service.
// All implementations must embed UnimplementedManagementServiceServer
// for forward compatibility.
//
// ManagementService provides functions for agents to register and report status
type ManagementServiceServer interface {
	IdentifyRemoteAgent(context.Context, *IdentifyRemoteAgentRequest) (*IdentifyRemoteAgentResponse, error)
	InitializeLocalAgentInsideLocalVM(context.Context, *InitializeLocalAgentInsideLocalVMRequest) (*InitializeLocalAgentInsideLocalVMResponse, error)
	InitilizeRemoteAgent(context.Context, *InitializeRemoteAgentRequest) (*InitializeRemoteAgentResponse, error)
	mustEmbedUnimplementedManagementServiceServer()
}

// UnimplementedManagementServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedManagementServiceServer struct{}

func (UnimplementedManagementServiceServer) IdentifyRemoteAgent(context.Context, *IdentifyRemoteAgentRequest) (*IdentifyRemoteAgentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method IdentifyRemoteAgent not implemented")
}
func (UnimplementedManagementServiceServer) InitializeLocalAgentInsideLocalVM(context.Context, *InitializeLocalAgentInsideLocalVMRequest) (*InitializeLocalAgentInsideLocalVMResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitializeLocalAgentInsideLocalVM not implemented")
}
func (UnimplementedManagementServiceServer) InitilizeRemoteAgent(context.Context, *InitializeRemoteAgentRequest) (*InitializeRemoteAgentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method InitilizeRemoteAgent not implemented")
}
func (UnimplementedManagementServiceServer) mustEmbedUnimplementedManagementServiceServer() {}
func (UnimplementedManagementServiceServer) testEmbeddedByValue()                           {}

// UnsafeManagementServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ManagementServiceServer will
// result in compilation errors.
type UnsafeManagementServiceServer interface {
	mustEmbedUnimplementedManagementServiceServer()
}

func RegisterManagementServiceServer(s grpc.ServiceRegistrar, srv ManagementServiceServer) {
	// If the following call pancis, it indicates UnimplementedManagementServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ManagementService_ServiceDesc, srv)
}

func _ManagementService_IdentifyRemoteAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IdentifyRemoteAgentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServiceServer).IdentifyRemoteAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ManagementService_IdentifyRemoteAgent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServiceServer).IdentifyRemoteAgent(ctx, req.(*IdentifyRemoteAgentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ManagementService_InitializeLocalAgentInsideLocalVM_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeLocalAgentInsideLocalVMRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServiceServer).InitializeLocalAgentInsideLocalVM(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ManagementService_InitializeLocalAgentInsideLocalVM_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServiceServer).InitializeLocalAgentInsideLocalVM(ctx, req.(*InitializeLocalAgentInsideLocalVMRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ManagementService_InitilizeRemoteAgent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InitializeRemoteAgentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ManagementServiceServer).InitilizeRemoteAgent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ManagementService_InitilizeRemoteAgent_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ManagementServiceServer).InitilizeRemoteAgent(ctx, req.(*InitializeRemoteAgentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// ManagementService_ServiceDesc is the grpc.ServiceDesc for ManagementService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ManagementService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ec1.v1poc1.ManagementService",
	HandlerType: (*ManagementServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "IdentifyRemoteAgent",
			Handler:    _ManagementService_IdentifyRemoteAgent_Handler,
		},
		{
			MethodName: "InitializeLocalAgentInsideLocalVM",
			Handler:    _ManagementService_InitializeLocalAgentInsideLocalVM_Handler,
		},
		{
			MethodName: "InitilizeRemoteAgent",
			Handler:    _ManagementService_InitilizeRemoteAgent_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ec1/v1poc1/management.proto",
}
