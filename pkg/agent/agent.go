package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/pkg/hypervisor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
)

// AgentConfig holds configuration for the agent
type AgentConfig struct {
	// Host address where the agent listens for incoming requests
	HostAddr string

	// ID of the agent
	AgentID string

	// Management server address for registration
	MgtAddr string
}

// Agent implements the EC1 Agent service
type Agent struct {
	// Configuration
	config AgentConfig

	// Hypervisor driver
	driver hypervisor.Driver
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

	return &Agent{
		config: config,
		driver: driver,
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
		AgentId:        &a.config.AgentID,
		HostAddress:    &a.config.HostAddr,
		HypervisorType: enumPtr(a.driver.GetHypervisorType()),
		TotalResources: resources,
	})

	_, err := mgtClient.RegisterAgent(ctx, req)
	if err != nil {
		return fmt.Errorf("registering with management server: %w", err)
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
