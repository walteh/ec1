package management

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/sftp"
	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/gen/proto/golang/ec1/validate/protovalidate"
	"github.com/walteh/ec1/pkg/agent"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var _ v1poc1connect.ManagementServiceHandler = &Server{}

// Agent represents a registered agent
type RegisteredAgent struct {
	ID      string
	Address string
	Client  v1poc1connect.AgentServiceClient

	// Address        string
	// HypervisorType ec1v1.HypervisorType
	// Resources      *ec1v1.Resources
	// VMs            map[string]*ec1v1.VMInfo
}

// Server implements the EC1 Management service
type Server struct {
	// Configuration
	config ServerConfig

	// Registered agents
	agents     map[string]*RegisteredAgent
	agentMutex sync.RWMutex
	ownAgent   v1poc1connect.AgentServiceClient
}

// ServerConfig holds configuration for the management server
type ServerConfig struct {
	// Address where the server listens for incoming requests
	HostAddr string
}

// New creates a new management server
func New(ctx context.Context, config ServerConfig) (*Server, error) {

	srv := &Server{
		config: config,
		agents: make(map[string]*RegisteredAgent),
	}

	c, err := agent.New(ctx, agent.AgentConfig{
		HostAddr:                 "localhost:9091",
		IDStore:                  &agent.FSIDStore{},
		InMemoryManagementClient: srv,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	ownAgent, cleanup := agent.NewInMemoryAgentClient(ctx, c)
	defer cleanup()

	srv.ownAgent = ownAgent

	_, err = srv.IdentifyRemoteAgent(ctx, connect.NewRequest(&ec1v1.IdentifyRemoteAgentRequest{
		AgentId:     ptr("own-agent"),
		HostAddress: ptr("localhost:9091"),
		// TotalResources: ptr(ec1v1.Resources{Cpu: ptr("1"), Memory: ptr("1024")}),
		// HypervisorType: ptr(ec1v1.HypervisorType_HYPERVISOR_TYPE_MAC_VIRTUALIZATION),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to register own agent: %w", err)
	}

	return srv, nil
}

func ptr[T any](v T) *T {
	return &v
}

func (s *Server) OnAgentProbe(ctx context.Context, agent *RegisteredAgent, resp *ec1v1.AgentProbeResponse) error {
	fmt.Printf("Probe response: %v\n", resp)
	return nil
}

// StartVM starts a VM on an appropriate agent
func (s *Server) StartVM(ctx context.Context, agent *RegisteredAgent, cloudInitPath, qcow2Path string, resourcesMax *ec1v1.Resources, networkConfig *ec1v1.VMNetworkConfig) (*ec1v1.VMInfo, error) {

	// upload the disk image to the agent
	err := s.uploadViaAgent(ctx, agent, qcow2Path)
	if err != nil {
		return nil, fmt.Errorf("failed to upload disk image: %w", err)
	}

	// upload the cloud init file to the agent
	err = s.uploadViaAgent(ctx, agent, cloudInitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to upload cloud init file: %w", err)
	}

	// Prepare the request
	req := connect.NewRequest(&ec1v1.StartVMRequest{
		ResourcesMax:  resourcesMax,
		ResourcesBoot: resourcesMax, // Just use the same resources for boot
		DiskImage: &ec1v1.DiskImage{
			Path: strPtr(qcow2Path),
			Type: ptr(ec1v1.DiskImageType_DISK_IMAGE_TYPE_QCOW2),
		},
		NetworkConfig: networkConfig,
	})

	if err := protovalidate.Validate(req.Msg); err != nil {
		return nil, fmt.Errorf("validating start VM request: %w", err)
	}

	// Call the agent to start the VM
	resp, err := agent.Client.StartVM(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start VM on agent %s: %w", agent.ID, err)
	}

	// Create VM info from response
	vmInfo := &ec1v1.VMInfo{
		VmId:          resp.Msg.VmId,
		Status:        resp.Msg.Status,
		IpAddress:     resp.Msg.IpAddress,
		ResourcesMax:  resourcesMax,
		ResourcesLive: resourcesMax, // For POC, assume live = max
	}

	// // Store the VM info
	// s.agentMutex.Lock()
	// s.agents[name].VMs[resp.Msg.GetVmId()] = vmInfo
	// s.agentMutex.Unlock()

	fmt.Printf("Started VM(ID: %s) on agent %s with IP %s\n",
		resp.Msg.GetVmId(),
		agent.ID,
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

func (s *Server) uploadViaAgent(ctx context.Context, agent *RegisteredAgent, path string) error {

	// get the size of the disk image
	diskImageStats, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get disk image size: %w", err)
	}
	diskImageSize := diskImageStats.Size()

	// upload the disk image to the agent
	diskImage, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open disk image: %w", err)
	}
	defer diskImage.Close()

	agentClientFileUploadReq := agent.Client.FileTransfer(ctx)

	chunkSize := 1024

	buffer := make([]byte, chunkSize)
	totalBytesRead := 0

	for {
		bytesRead, err := diskImage.Read(buffer)
		if err != nil {
			if err.Error() == "EOF" {
				break // End of file
			}
			fmt.Println("Error reading file:", err)
			return fmt.Errorf("failed to read disk image: %w", err)
		}
		bytesToSend := buffer[:bytesRead]

		err = agentClientFileUploadReq.Send(&ec1v1.FileTransferRequest{
			Name:            strPtr(filepath.Base(path)),
			ChunkBytes:      bytesToSend,
			ChunkByteSize:   ptr(uint32(len(bytesToSend))),
			ChunkOfTotal:    ptr(uint32(totalBytesRead / chunkSize)),
			ChunkTotalCount: ptr(uint32(diskImageSize / int64(chunkSize))),
		})
		if err != nil {
			return fmt.Errorf("failed to send file transfer request: %w", err)
		}

		totalBytesRead += len(bytesToSend)
	}

	resp, err := agentClientFileUploadReq.CloseAndReceive()
	if err != nil {
		return fmt.Errorf("failed to close and receive file transfer request: %w", err)
	}

	if resp.Msg.GetError() != "" {
		return fmt.Errorf("failed to upload disk image: %s", resp.Msg.GetError())
	}

	return nil
}

func (s *Server) uploadViaSSH(ctx context.Context, sshClient *ssh.Client, localPath, remotePath string) error {
	return copyViaSFTP(sshClient, localPath, remotePath)
}

func copyViaSFTP(sshClient *ssh.Client, srcPath, dstPath string) error {
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := sftpClient.Create(dstPath)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := dstFile.ReadFrom(srcFile); err != nil {
		return err
	}
	return nil
}

// RegisterAgent handles the RegisterAgent RPC
func (s *Server) IdentifyRemoteAgent(ctx context.Context, req *connect.Request[ec1v1.IdentifyRemoteAgentRequest]) (*connect.Response[ec1v1.IdentifyRemoteAgentResponse], error) {
	agentID := req.Msg.GetAgentId()

	// Create a new agent entry
	agent := &RegisteredAgent{
		ID:      agentID,
		Address: req.Msg.GetHostAddress(),
		Client:  v1poc1connect.NewAgentServiceClient(http.DefaultClient, req.Msg.GetHostAddress()),
	}

	// Register the agent
	s.agentMutex.Lock()
	s.agents[agentID] = agent
	s.agentMutex.Unlock()

	// make the probe request
	probeStream := agent.Client.AgentProbe(ctx)
	err := probeStream.Send(&ec1v1.AgentProbeRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to make probe request: %w", err)
	}

	fmt.Printf("Agent registered: %s\n", agentID)

	// wait for the probe response
	_, err = probeStream.Receive()
	if err != nil {
		return nil, fmt.Errorf("failed to receive probe response: %w", err)
	}

	go func() {
		for {
			resp, err := probeStream.Receive()
			if err != nil {
				return
			}
			err = s.OnAgentProbe(ctx, agent, resp)
			if err != nil {
				fmt.Printf("failed to handle probe response: %s\n", err)
			}
		}
	}()

	// Return success
	return connect.NewResponse(&ec1v1.IdentifyRemoteAgentResponse{}), nil
}

// InitializeLocalAgentInsideLocalVM implements v1poc1connect.ManagementServiceHandler.
func (s *Server) InitializeLocalAgentInsideLocalVM(ctx context.Context, req *connect.Request[ec1v1.InitializeLocalAgentInsideLocalVMRequest]) (*connect.Response[ec1v1.InitializeLocalAgentInsideLocalVMResponse], error) {

	ownRegisteredAgent := &RegisteredAgent{
		ID:      "own-agent",
		Address: "localhost:9091",
		Client:  s.ownAgent,
	}

	agentBinary, err := GetAgentBinary(ctx, req.Msg.GetArch(), req.Msg.GetOs())
	if err != nil {
		return nil, fmt.Errorf("failed 	to get agent binary: %w", err)
	}

	extraFiles := map[string]string{
		"agent": agentBinary,
	}

	cloudInitISO, cleanup, err := CreateCloudInitISO(ctx, req.Msg.GetCloudinitMetadata(), req.Msg.GetCloudinitUserdata(), extraFiles, addAgentBinaryToSystemd)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud init ISO: %w", err)
	}
	defer cleanup()

	// use the local agent to start a new VM
	vm, err := s.StartVM(ctx, ownRegisteredAgent, req.Msg.GetQcow2ImagePath(), cloudInitISO, ptr(ec1v1.Resources{Cpu: ptr("1"), Memory: ptr("1024")}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start VM: %w", err)
	}

	_, err = s.InitilizeRemoteAgent(ctx, connect.NewRequest(&ec1v1.InitializeRemoteAgentRequest{
		HostAddress: vm.IpAddress,
		Arch:        ptr(req.Msg.GetArch()),
		Os:          ptr(req.Msg.GetOs()),
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote agent: %w", err)
	}

	// TODO: register the VM with the management server

	return connect.NewResponse(&ec1v1.InitializeLocalAgentInsideLocalVMResponse{}), nil
}

// StartLocalAgent starts the agent on the local machine in a separate process
func GetAgentBinary(ctx context.Context, archstr, osstr string) (string, error) {
	path := fmt.Sprintf("bin/agent-%s-%s", archstr, osstr)

	_, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to get agent binary: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	return absPath, nil
}

func (s *Server) runCommandViaSSH(ctx context.Context, sshClient *ssh.Client, command string) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create new SSH session: %w", err)
	}
	defer session.Close()

	pr, pw := io.Pipe()
	session.Stdout = pw
	session.Stderr = pw

	err = session.Run(command)

	if err != nil {
		output, _ := io.ReadAll(pr)
		return "", fmt.Errorf("failed to run command: %w, output: %s", err, string(output))
	}

	output, err := io.ReadAll(pr)
	if err != nil {
		return "", fmt.Errorf("failed to read command output: %w", err)
	}

	return string(output), nil
}

// InitilizeRemoteAgent implements v1poc1connect.ManagementServiceHandler.
func (s *Server) InitilizeRemoteAgent(ctx context.Context, req *connect.Request[ec1v1.InitializeRemoteAgentRequest]) (*connect.Response[ec1v1.InitializeRemoteAgentResponse], error) {

	sshClient, err := ssh.Dial("tcp", req.Msg.GetHostAddress(), &ssh.ClientConfig{
		User: req.Msg.GetSshUsername(),
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Msg.GetSshPassword()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer sshClient.Close()

	agentBinary, err := GetAgentBinary(ctx, req.Msg.GetArch(), req.Msg.GetOs())
	if err != nil {
		return nil, fmt.Errorf("failed 	to get agent binary: %w", err)
	}

	// upload the agent binary to the agent
	err = s.uploadViaSSH(ctx, sshClient, agentBinary, "/bin/ec1-agent")
	if err != nil {
		return nil, fmt.Errorf("failed to upload agent binary: %w", err)
	}

	// call the agent binary "setup" command on the remote machine
	output, err := s.runCommandViaSSH(ctx, sshClient, "/bin/ec1-agent")
	if err != nil {
		return nil, fmt.Errorf("failed to run setup command: %w, output: %s", err, output)
	}

	fmt.Printf("Setup command output: %s\n", output)

	// now that the agent is setup, we trigger systemd to start the agent
	output, err = s.runCommandViaSSH(ctx, sshClient, "systemctl start ec1-agent")
	if err != nil {
		return nil, fmt.Errorf("failed to start agent: %w, output: %s", err, output)
	}

	// the agent should have started and contacted us - make a temporary agent client to confirm
	agentClient := v1poc1connect.NewAgentServiceClient(http.DefaultClient, req.Msg.GetHostAddress())

	// wait for the agent to register
	time.Sleep(1 * time.Second)

	// make a probe request
	statusResp, err := agentClient.Status(ctx, connect.NewRequest(&ec1v1.StatusRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to make probe request: %w", err)
	}

	fmt.Printf("Agent status: %v\n", statusResp.Msg)

	if s.agents[statusResp.Msg.GetAgentId()] == nil {
		return nil, fmt.Errorf("agent not found")
	}

	return connect.NewResponse(&ec1v1.InitializeRemoteAgentResponse{}), nil
}

// // ReportAgentStatus handles the ReportAgentStatus RPC
// func (s *Server) ReportAgentStatus(ctx context.Context, req *connect.Request[ec1v1.ReportAgentStatusRequest]) (*connect.Response[ec1v1.ReportAgentStatusResponse], error) {
// 	agentID := req.Msg.GetAgentId()

// 	// Find the agent
// 	s.agentMutex.Lock()
// 	agent, exists := s.agents[agentID]
// 	if !exists {
// 		s.agentMutex.Unlock()
// 		return connect.NewResponse(&ec1v1.ReportAgentStatusResponse{
// 			Success: boolPtr(false),
// 			Error:   strPtr(fmt.Sprintf("Agent with ID %s not found", agentID)),
// 		}), nil
// 	}

// 	// Update VM information
// 	for _, vmInfo := range req.Msg.Vms {
// 		agent.VMs[vmInfo.GetVmId()] = vmInfo
// 	}
// 	s.agentMutex.Unlock()

// 	// Return success
// 	return connect.NewResponse(&ec1v1.ReportAgentStatusResponse{
// 		Success: boolPtr(true),
// 	}), nil
// }
