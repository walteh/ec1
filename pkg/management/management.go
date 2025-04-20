package management

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	"connectrpc.com/connect"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/gen/proto/golang/ec1/validate/protovalidate"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Agent represents a registered agent
type Agent struct {
	ID             string
	Address        string
	HypervisorType ec1v1.HypervisorType
	Resources      *ec1v1.Resources
	VMs            map[string]*ec1v1.VMInfo
}

// Server implements the EC1 Management service
type Server struct {
	// Configuration
	config ServerConfig

	// Registered agents
	agents     map[string]*Agent
	agentMutex sync.RWMutex
}

// ServerConfig holds configuration for the management server
type ServerConfig struct {
	// Address where the server listens for incoming requests
	HostAddr string
}

// New creates a new management server
func New(config ServerConfig) *Server {
	return &Server{
		config: config,
		agents: make(map[string]*Agent),
	}
}

// RegisterAgent handles the RegisterAgent RPC
func (s *Server) RegisterAgent(ctx context.Context, req *connect.Request[ec1v1.RegisterAgentRequest]) (*connect.Response[ec1v1.RegisterAgentResponse], error) {
	agentID := req.Msg.GetAgentId()

	// Create a new agent entry
	agent := &Agent{
		ID:             agentID,
		Address:        req.Msg.GetHostAddress(),
		HypervisorType: req.Msg.GetHypervisorType(),
		Resources:      req.Msg.TotalResources,
		VMs:            make(map[string]*ec1v1.VMInfo),
	}

	// Register the agent
	s.agentMutex.Lock()
	s.agents[agentID] = agent
	s.agentMutex.Unlock()

	fmt.Printf("Agent registered: %s (%s) with %s resources\n",
		agentID,
		ec1v1.HypervisorType_name[int32(agent.HypervisorType)],
		fmt.Sprintf("CPU: %s, Memory: %s", agent.Resources.GetCpu(), agent.Resources.GetMemory()),
	)

	// Return success
	return connect.NewResponse(&ec1v1.RegisterAgentResponse{
		Success: boolPtr(true),
	}), nil
}

// ReportAgentStatus handles the ReportAgentStatus RPC
func (s *Server) ReportAgentStatus(ctx context.Context, req *connect.Request[ec1v1.ReportAgentStatusRequest]) (*connect.Response[ec1v1.ReportAgentStatusResponse], error) {
	agentID := req.Msg.GetAgentId()

	// Find the agent
	s.agentMutex.Lock()
	agent, exists := s.agents[agentID]
	if !exists {
		s.agentMutex.Unlock()
		return connect.NewResponse(&ec1v1.ReportAgentStatusResponse{
			Success: boolPtr(false),
			Error:   strPtr(fmt.Sprintf("Agent with ID %s not found", agentID)),
		}), nil
	}

	// Update VM information
	for _, vmInfo := range req.Msg.Vms {
		agent.VMs[vmInfo.GetVmId()] = vmInfo
	}
	s.agentMutex.Unlock()

	// Return success
	return connect.NewResponse(&ec1v1.ReportAgentStatusResponse{
		Success: boolPtr(true),
	}), nil
}

func ptr[T any](v T) *T {
	return &v
}

// StartVM starts a VM on an appropriate agent
func (s *Server) StartVM(ctx context.Context, name, diskImagePath string, resourcesMax *ec1v1.Resources, networkConfig *ec1v1.VMNetworkConfig) (*ec1v1.VMInfo, error) {
	// In a real implementation, we would:
	// 1. Choose an appropriate agent based on resources, hypervisor type, etc.
	// 2. Send StartVM request to the chosen agent
	// 3. Wait for the VM to start and return its info

	// For the POC, we'll select the first agent of each type
	var macAgent, kvmAgent *Agent

	s.agentMutex.RLock()
	for _, agent := range s.agents {
		if agent.HypervisorType == ec1v1.HypervisorType_HYPERVISOR_TYPE_MAC_VIRTUALIZATION && macAgent == nil {
			macAgent = agent
		} else if agent.HypervisorType == ec1v1.HypervisorType_HYPERVISOR_TYPE_KVM && kvmAgent == nil {
			kvmAgent = agent
		}

		if macAgent != nil && kvmAgent != nil {
			break
		}
	}
	s.agentMutex.RUnlock()

	// For nested virtualization demo, if we have both types,
	// start on Mac first (if name contains "linux"),
	// or on Linux KVM (if name contains "nested")
	var targetAgent *Agent
	if macAgent != nil && kvmAgent != nil {
		if name == "linux" || name == "linux-vm" {
			targetAgent = macAgent
		} else if name == "nested" || name == "nested-vm" {
			targetAgent = kvmAgent
		} else {
			// Default to Mac for other names
			targetAgent = macAgent
		}
	} else if macAgent != nil {
		targetAgent = macAgent
	} else if kvmAgent != nil {
		targetAgent = kvmAgent
	} else {
		return nil, fmt.Errorf("no agents available")
	}

	// Create a client for the selected agent
	agentClient := v1poc1connect.NewAgentServiceClient(
		http.DefaultClient,
		targetAgent.Address,
	)

	// Prepare the request
	req := connect.NewRequest(&ec1v1.StartVMRequest{
		Name:          strPtr(name),
		ResourcesMax:  resourcesMax,
		ResourcesBoot: resourcesMax, // Just use the same resources for boot
		DiskImage: &ec1v1.DiskImage{
			Path: strPtr(diskImagePath),
			Type: ptr(ec1v1.DiskImageType_DISK_IMAGE_TYPE_QCOW2),
		},
		NetworkConfig: networkConfig,
	})

	if err := protovalidate.Validate(req.Msg); err != nil {
		return nil, fmt.Errorf("validating start VM request: %w", err)
	}

	// Call the agent to start the VM
	resp, err := agentClient.StartVM(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start VM on agent %s: %w", targetAgent.ID, err)
	}

	// Create VM info from response
	vmInfo := &ec1v1.VMInfo{
		VmId:          resp.Msg.VmId,
		Name:          strPtr(name),
		Status:        resp.Msg.Status,
		IpAddress:     resp.Msg.IpAddress,
		ResourcesMax:  resourcesMax,
		ResourcesLive: resourcesMax, // For POC, assume live = max
	}

	// Store the VM info
	s.agentMutex.Lock()
	targetAgent.VMs[resp.Msg.GetVmId()] = vmInfo
	s.agentMutex.Unlock()

	fmt.Printf("Started VM %s (ID: %s) on agent %s with IP %s\n",
		name,
		resp.Msg.GetVmId(),
		targetAgent.ID,
		resp.Msg.GetIpAddress(),
	)

	return vmInfo, nil
}

// Helper function to convert string to string pointer
func strPtr(s string) *string {
	return &s
}

// Helper function to convert bool to bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// Start starts the management server
func (s *Server) Start(ctx context.Context) error {
	// Create Connect-based service
	path, handler := v1poc1connect.NewManagementServiceHandler(s)

	// Set up a HTTP server with h2c (HTTP/2 over cleartext)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := &http.Server{
		Addr:    s.config.HostAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	// Create listener
	listener, err := net.Listen("tcp", s.config.HostAddr)
	if err != nil {
		return fmt.Errorf("starting listener: %w", err)
	}

	// Start serving
	fmt.Printf("EC1 Management Server listening on %s\n", s.config.HostAddr)
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.Serve(listener)
}
