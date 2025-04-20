package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/gen/proto/golang/ec1/validate/protovalidate"
	"github.com/walteh/ec1/pkg/hypervisor"
	"github.com/walteh/ec1/pkg/id"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

func ptr[T any](v T) *T {
	return &v
}

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	// Host address where the agent listens for incoming requests
	HostAddr string

	// Management server address for registration
	MgtAddr string

	IDStore IDStore
}

// Agent implements the EC1 Agent service
type Agent struct {
	// Configuration
	config AgentConfig

	// ID of the agent
	agentID id.ID

	// Hypervisor driver
	driver hypervisor.Driver

	managementClient v1poc1connect.ManagementServiceClient

	// VM status channel
	vmStatusChan chan *ec1v1.VMStatusResponse
}

func (a *Agent) ID() id.ID {
	return a.agentID
}

// Helper function to convert enum to pointer
func enumPtr[T ~int32](e T) *T {
	return &e
}

// New creates a new agent instance
func New(ctx context.Context, config AgentConfig) (*Agent, error) {
	// Create the appropriate hypervisor driver
	driver, err := hypervisor.NewDriver(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating hypervisor driver: %w", err)
	}

	agentID, ok, err := config.IDStore.GetInstanceID(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting agent ID: %w", err)
	}
	if !ok {
		agentID = id.NewID("agent")
		if err := config.IDStore.SetInstanceID(ctx, agentID); err != nil {
			return nil, fmt.Errorf("setting agent ID: %w", err)
		}
	}

	managementClient := v1poc1connect.NewManagementServiceClient(
		http.DefaultClient,
		config.MgtAddr,
	)

	return &Agent{
		config:           config,
		driver:           driver,
		agentID:          agentID,
		managementClient: managementClient,
	}, nil
}

// StartVM handles the StartVM RPC
func (a *Agent) StartVM(ctx context.Context, req *connect.Request[ec1v1.StartVMRequest]) (*connect.Response[ec1v1.StartVMResponse], error) {
	// Delegate to the hypervisor driver
	resp, err := a.driver.StartVM(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}

// StopVM handles the StopVM RPC
func (a *Agent) StopVM(ctx context.Context, req *connect.Request[ec1v1.StopVMRequest]) (*connect.Response[ec1v1.StopVMResponse], error) {
	// Delegate to the hypervisor driver
	resp, err := a.driver.StopVM(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}

var _ v1poc1connect.AgentServiceHandler = &Agent{}

// GetVMStatus handles the GetVMStatus RPC
func (a *Agent) VMStatus(ctx context.Context, req *connect.Request[ec1v1.VMStatusRequest], stream *connect.ServerStream[ec1v1.VMStatusResponse]) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case vmStatus := <-a.vmStatusChan:
			if err := stream.Send(vmStatus); err != nil {
				fmt.Printf("error sending VM status: %v\n", err)
				return err
			}
		}
	}
}

// GetVMStatus handles the GetVMStatus RPC
func (a *Agent) GetVMStatus(ctx context.Context, req *connect.Request[ec1v1.GetVMStatusRequest]) (*connect.Response[ec1v1.GetVMStatusResponse], error) {
	// Delegate to the hypervisor driver
	resp, err := a.driver.GetVMStatus(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(resp), nil
}

// RegisterWithManagement registers this agent with the management server
func (a *Agent) RegisterWithManagement(ctx context.Context) error {
	// Create a client to the management server
	mgtClient := v1poc1connect.NewManagementServiceClient(
		http.DefaultClient,
		a.config.MgtAddr,
	)

	// Get total resources from the hypervisor (in real impl)
	// For the POC, we'll hardcode some resource values
	memory := "4Gi"
	cpu := "4"
	resources := &ec1v1.Resources{
		Memory: &memory,
		Cpu:    &cpu,
	}

	// Register with the management server
	req := connect.NewRequest(&ec1v1.RegisterAgentRequest{
		AgentId:        ptr(a.agentID.String()),
		HostAddress:    &a.config.HostAddr,
		HypervisorType: enumPtr(a.driver.GetHypervisorType()),
		TotalResources: resources,
	})

	if err := protovalidate.Validate(req.Msg); err != nil {
		return fmt.Errorf("validating register agent request: %w", err)
	}

	_, err := mgtClient.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("calling management server to register agent: %w", err)
	}

	return nil
}

// Start starts the agent service
func (a *Agent) Start(ctx context.Context) error {
	// Create Connect-based service
	path, handler := v1poc1connect.NewAgentServiceHandler(a)

	// Apply any interceptors (monitoring, logging, auth)
	// For POC, we'll keep it simple

	// Set up a HTTP server with h2c (HTTP/2 over cleartext)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := &http.Server{
		Addr:    a.config.HostAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	// Create listener
	listener, err := net.Listen("tcp", a.config.HostAddr)
	if err != nil {
		return fmt.Errorf("starting listener: %w", err)
	}

	// Register with management server
	if a.config.MgtAddr != "" {
		if err := a.RegisterWithManagement(ctx); err != nil {
			return err
		}
	}

	// Start serving
	fmt.Printf("EC1 Agent listening on %s\n", a.config.HostAddr)
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.Serve(listener)
}
