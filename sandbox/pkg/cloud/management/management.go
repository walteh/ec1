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

	"gitlab.com/tozd/go/errors"

	"connectrpc.com/connect"
	"github.com/pkg/sftp"
	slogctx "github.com/veqryn/slog-context"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	ec1v1 "github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1"
	"github.com/walteh/ec1/gen/proto/golang/ec1/v1poc1/v1poc1connect"
	"github.com/walteh/ec1/gen/proto/golang/ec1/validate/protovalidate"
	"github.com/walteh/ec1/sandbox/pkg/cloud/agent"
	"github.com/walteh/ec1/sandbox/pkg/cloud/id"
)

var _ v1poc1connect.ManagementServiceHandler = &Server{}

const OWN_AGENT_ID = "agent-00000000000000000000"

// Agent represents a registered agent
type RegisteredAgent struct {
	ID     string
	NodeIP string
	// AgentConnectUri    string
	LastProbeTime      time.Time
	LastProbeLiveness  bool
	LastProbeReadiness bool
	IsInMemory         bool

	// Address        string
	// HypervisorType ec1v1.HypervisorType
	// Resources      *ec1v1.Resources
	// VMs            map[string]*ec1v1.VMInfo
}

// Server implements the EC1 Management service
type Server struct {
	// Configuration
	config ServerConfig

	// ownAgent v1poc1connect.AgentServiceClient

	clients       map[string]v1poc1connect.AgentServiceClient
	clientClosers map[string]func()

	clientsMutex           sync.RWMutex
	clientsIndividualMutex sync.RWMutex

	db *Database
}

// ServerConfig holds configuration for the management server
type ServerConfig struct {
	// Address where the server listens for incoming requests
	// HostAddr string
}

// New creates a new management server
func New(ctx context.Context, config ServerConfig) (*Server, error) {

	srv := &Server{
		config: config,
	}

	db, err := NewDatabase("./wrk/mgt.db")
	if err != nil {
		return nil, errors.Errorf("failed to create database: %w", err)
	}
	srv.db = db

	srv.clients = make(map[string]v1poc1connect.AgentServiceClient)
	srv.clientClosers = make(map[string]func())
	srv.clientsMutex = sync.RWMutex{}
	srv.clientsIndividualMutex = sync.RWMutex{}

	return srv, nil
}

func (s *Server) startProbeLoop(ctx context.Context, registeredAgent RegisteredAgent) error {

	s.clientsMutex.Lock()
	id := s.clients[registeredAgent.ID]
	s.clientsMutex.Unlock()

	slogctx.Info(ctx, "Starting probe loop", "agent_id", registeredAgent.ID, "node_ip", registeredAgent.NodeIP)

	if id != nil {
		slogctx.Error(ctx, "Agent already registered", "agent_id", registeredAgent.ID, "node_ip", registeredAgent.NodeIP)
		return errors.New("agent already registered: " + registeredAgent.ID)
	}

	var client v1poc1connect.AgentServiceClient
	var agentClientCleanup func()
	if registeredAgent.IsInMemory {
		c, err := agent.New(ctx, agent.AgentConfig{
			HostAddr:                 ":memory:",
			MgtAddr:                  ":memory:",
			IDStore:                  &agent.FSIDStore{},
			InMemoryManagementClient: s,
		})
		if err != nil {
			return errors.Errorf("failed to create agent: %w", err)
		}
		i, cleanup := agent.NewInMemoryAgentClient(ctx, c)
		// defer cleanup()
		client = i
		agentClientCleanup = cleanup
	} else {
		client = v1poc1connect.NewAgentServiceClient(http.DefaultClient, fmt.Sprintf("%s:9091/ec1-agent", registeredAgent.NodeIP))
	}

	slogctx.Info(ctx, "Probing agent", "agent_id", registeredAgent.ID, "node_ip", registeredAgent.NodeIP)

	// make the probe request
	probeStream := client.AgentProbe(ctx)
	err := probeStream.Send(&ec1v1.AgentProbeRequest{})
	if err != nil {
		return errors.Errorf("failed to make probe request: %w", err)
	}

	defer probeStream.CloseRequest()

	// wait for the probe response
	resp, err := probeStream.Receive()
	if err != nil {
		return errors.Errorf("failed to receive probe response: %w", err)
	}

	registeredAgent, err = s.OnAgentProbe(ctx, registeredAgent, resp)
	if err != nil {
		return errors.Errorf("failed to handle probe response: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)

	s.clientsMutex.Lock()
	s.clients[registeredAgent.ID] = client
	s.clientClosers[registeredAgent.ID] = cancel
	s.clientsMutex.Unlock()

	fmt.Printf("Agent registered: %s @ %s\n", registeredAgent.ID, registeredAgent.NodeIP)

	probeChannel := make(chan *ec1v1.AgentProbeResponse)
	probeErrChannel := make(chan error)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				resp, err := probeStream.Receive()
				if err != nil {
					slogctx.Error(ctx, "failed to receive probe response", "error", err)
					probeErrChannel <- err
					return
				}
				probeChannel <- resp
			}
		}
	}()
	go func() {
		defer func() {
			s.clientsMutex.Lock()
			s.clients[registeredAgent.ID] = nil
			s.clientClosers[registeredAgent.ID] = nil
			s.clientsMutex.Unlock()
			if agentClientCleanup != nil {
				agentClientCleanup()
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case err := <-probeErrChannel:
				if err != nil {
					slogctx.Error(ctx, "failed to receive probe response", "error", err)
				}
				return
			case resp := <-probeChannel:
				registeredAgent, err = s.OnAgentProbe(ctx, registeredAgent, resp)
				if err != nil {
					slogctx.Error(ctx, "failed to handle probe response", "error", err)
				}
			}
		}
	}()

	slogctx.Info(ctx, "Probe loop started", "agent_id", registeredAgent.ID, "node_ip", registeredAgent.NodeIP)

	return nil
}

func ptr[T any](v T) *T {
	return &v
}

func (s *Server) OnAgentProbe(ctx context.Context, agent RegisteredAgent, resp *ec1v1.AgentProbeResponse) (RegisteredAgent, error) {
	// save in the database
	agent.LastProbeTime = time.Now()
	agent.LastProbeLiveness = resp.GetLive()
	agent.LastProbeReadiness = resp.GetReady()
	err := s.db.SaveAgent(agent)
	if err != nil {
		return agent, errors.Errorf("failed to save agent: %w", err)
	}

	slogctx.Info(ctx, "OnAgentProbe saved", "agent_id", agent.ID, "node_ip", agent.NodeIP, "live", resp.GetLive(), "ready", resp.GetReady())
	return agent, nil
}

// StartVM starts a VM on an appropriate agent
func (s *Server) StartVM(ctx context.Context, agent RegisteredAgent, cloudInitPath, qcow2Path string, resourcesMax *ec1v1.Resources, networkConfig *ec1v1.VMNetworkConfig) (*ec1v1.StartVMResponse, error) {

	// upload the disk image to the agent
	err := s.uploadViaAgent(ctx, agent, qcow2Path)
	if err != nil {
		return nil, errors.Errorf("failed to upload disk image: %w", err)
	}

	// upload the cloud init file to the agent
	err = s.uploadViaAgent(ctx, agent, cloudInitPath)
	if err != nil {
		return nil, errors.Errorf("failed to upload cloud init file: %w", err)
	}

	agentClient, cleanup, err := s.useAgentClient(ctx, agent)
	if err != nil {
		return nil, errors.Errorf("failed to use agent client: %w", err)
	}
	defer cleanup()

	slogctx.Info(ctx, "Starting VM", "agent_id", agent.ID, "node_ip", agent.NodeIP, "qcow2_path", qcow2Path, "cloud_init_path", cloudInitPath)

	// Prepare the request
	req := connect.NewRequest(&ec1v1.StartVMRequest{
		ResourcesMax:  resourcesMax,
		ResourcesBoot: resourcesMax, // Just use the same resources for boot
		DiskImage: &ec1v1.DiskImage{
			Path: ptr("file://" + qcow2Path),
			Type: ptr(ec1v1.DiskImageType_DISK_IMAGE_TYPE_QCOW2),
		},
		NetworkConfig: networkConfig,
	})

	if err := protovalidate.Validate(req.Msg); err != nil {
		return nil, errors.Errorf("validating start VM request: %w", err)
	}

	// Call the agent to start the VM
	resp, err := agentClient.StartVM(ctx, req)
	if err != nil {
		return nil, errors.Errorf("failed to start VM on agent %s: %w", agent.ID, err)
	}

	// // Create VM info from response
	// vmInfo := &ec1v1.VMInfo{
	// 	VmId:          resp.Msg.VmId,
	// 	Status:        resp.Msg.Status,
	// 	IpAddress:     resp.Msg.IpAddress,
	// 	ResourcesMax:  resourcesMax,
	// 	ResourcesLive: resourcesMax, // For POC, assume live = max
	// }

	// // Store the VM info
	// s.agentMutex.Lock()
	// s.agents[name].VMs[resp.Msg.GetVmId()] = vmInfo
	// s.agentMutex.Unlock()

	fmt.Printf("Started VM(ID: %s) on agent %s with IP %s\n",
		resp.Msg.GetVmId(),
		agent.ID,
		resp.Msg.GetIpAddress(),
	)

	return resp.Msg, nil
}

// Start starts the management server
func (s *Server) Start(ctx context.Context, hostAddr string) (func() error, error) {
	slogctx.Info(ctx, "Starting management server", "host_addr", hostAddr)
	// Create Connect-based service
	path, handler := v1poc1connect.NewManagementServiceHandler(s)

	// Set up a HTTP server with h2c (HTTP/2 over cleartext)
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := &http.Server{
		Addr:    hostAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	// Create listener
	listener, err := net.Listen("tcp", hostAddr)
	if err != nil {
		return nil, errors.Errorf("starting listener: %w", err)
	}

	slogctx.Info(ctx, "Management server listening", "host_addr", hostAddr)

	// Start serving
	fmt.Printf("EC1 Management Server listening on %s\n", hostAddr)
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
		slogctx.Info(ctx, "Management server shutdown", "host_addr", hostAddr)
	}()

	err = s.startProbeLoop(ctx, RegisteredAgent{
		ID:         OWN_AGENT_ID,
		NodeIP:     ":memory:",
		IsInMemory: true,
	})
	slogctx.Error(ctx, "failed to start probe loop", "error", err)
	if err != nil {
		slogctx.Error(ctx, "failed to start probe loop", "error", err)
		return nil, errors.Errorf("failed to start probe loop: %w", err)
	}

	// init the clients
	registeredAgents, err := s.db.GetAllAgents()
	if err != nil {
		return nil, errors.Errorf("failed to get all agents: %w", err)
	}

	fmt.Printf("Found %d registered agents\n", len(registeredAgents))

	for _, agent := range registeredAgents {
		if agent.IsInMemory {
			continue
		}
		err := s.startProbeLoop(ctx, agent)
		if err != nil {
			return nil, errors.Errorf("starting probe loop for agent %s: %w", agent.ID, err)
		}
	}

	return func() error {
		return server.Serve(listener)
	}, nil
}

func (s *Server) useAgentClient(ctx context.Context, agent RegisteredAgent) (v1poc1connect.AgentServiceClient, func(), error) {

	s.clientsMutex.Lock()
	client := s.clients[agent.ID]
	s.clientsMutex.Unlock()

	if client == nil {
		return nil, nil, errors.Errorf("agent not found: %s", agent.ID)
	}

	s.clientsIndividualMutex.Lock()

	return client, func() {
		s.clientsIndividualMutex.Unlock()
	}, nil

}

func (s *Server) uploadViaAgent(ctx context.Context, agent RegisteredAgent, path string) error {

	if agent.IsInMemory {
		return nil
	}

	// get the size of the disk image
	diskImageStats, err := os.Stat(path)
	if err != nil {
		return errors.Errorf("failed to get disk image size: %w", err)
	}
	diskImageSize := diskImageStats.Size()

	// upload the disk image to the agent
	diskImage, err := os.Open(path)
	if err != nil {
		return errors.Errorf("failed to open disk image: %w", err)
	}
	defer diskImage.Close()

	agentClient, cleanup, err := s.useAgentClient(ctx, agent)
	if err != nil {
		return errors.Errorf("failed to use agent client: %w", err)
	}
	defer cleanup()

	agentClientFileUploadReq := agentClient.FileTransfer(ctx)

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
			return errors.Errorf("failed to read disk image: %w", err)
		}
		bytesToSend := buffer[:bytesRead]

		err = agentClientFileUploadReq.Send(&ec1v1.FileTransferRequest{
			Name:            ptr(filepath.Base(path)),
			ChunkBytes:      bytesToSend,
			ChunkByteSize:   ptr(uint32(len(bytesToSend))),
			ChunkOfTotal:    ptr(uint32(totalBytesRead / chunkSize)),
			ChunkTotalCount: ptr(uint32(diskImageSize / int64(chunkSize))),
		})
		if err != nil {
			return errors.Errorf("failed to send file transfer request: %w", err)
		}

		totalBytesRead += len(bytesToSend)
	}

	resp, err := agentClientFileUploadReq.CloseAndReceive()
	if err != nil {
		return errors.Errorf("failed to close and receive file transfer request: %w", err)
	}

	if resp.Msg.GetError() != "" {
		return errors.Errorf("failed to upload disk image: %s", resp.Msg.GetError())
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

	_, found, err := s.db.GetAgent(agentID)
	if err != nil {
		return nil, errors.Errorf("failed to get agent: %w", err)
	}

	if found {
		slogctx.Info(ctx, "agent already registered", "agent_id", agentID, "node_ip", req.Msg.GetNodeIp())
		return connect.NewResponse(&ec1v1.IdentifyRemoteAgentResponse{}), nil
	}

	// Create a new agent entry
	agent := RegisteredAgent{
		ID:     agentID,
		NodeIP: req.Msg.GetNodeIp(),
		// AgentConnectUri: req.Msg.GetAgentConnectUri(),
	}

	// client := v1poc1connect.NewAgentServiceClient(http.DefaultClient, req.Msg.GetAgentConnectUri())

	// Register the agent
	err = s.db.SaveAgent(agent)
	if err != nil {
		return nil, errors.Errorf("failed to save agent: %w", err)
	}

	// Return success
	return connect.NewResponse(&ec1v1.IdentifyRemoteAgentResponse{}), nil
}

// InitializeLocalAgentInsideLocalVM implements v1poc1connect.ManagementServiceHandler.
func (s *Server) InitializeLocalAgentInsideLocalVM(ctx context.Context, req *connect.Request[ec1v1.InitializeLocalAgentInsideLocalVMRequest]) (*connect.Response[ec1v1.InitializeLocalAgentInsideLocalVMResponse], error) {

	agentBinary, err := GetAgentBinary(ctx, req.Msg.GetArch(), req.Msg.GetOs())
	if err != nil {
		return nil, errors.Errorf("getting agent binary: %w", err)
	}

	extraFiles := map[string]string{
		"agent": agentBinary,
	}

	cloudInitISO, cleanup, err := CreateCloudInitISO(ctx, req.Msg.GetCloudinitMetadata(), req.Msg.GetCloudinitUserdata(), extraFiles, addAgentBinaryToSystemd, addNetworkInitRunCmd)
	if err != nil {
		return nil, errors.Errorf("failed to create cloud init ISO: %w", err)
	}
	defer cleanup()

	absCloudInitISO, err := filepath.Abs(cloudInitISO)
	if err != nil {
		return nil, errors.Errorf("failed to get absolute path: %w", err)
	}

	ownRegisteredAgent, found, err := s.db.GetAgent(OWN_AGENT_ID)
	if err != nil || !found {
		return nil, errors.Errorf("failed to get own agent: %w", err)
	}

	absQcow2Path, err := filepath.Abs(req.Msg.GetQcow2ImagePath())
	if err != nil {
		return nil, errors.Errorf("failed to get absolute path: %w", err)
	}

	// use the local agent to start a new VM
	vm, err := s.StartVM(ctx, ownRegisteredAgent, absCloudInitISO, absQcow2Path, ptr(ec1v1.Resources{Cpu: ptr("1"), Memory: ptr("1024")}), nil)
	if err != nil {
		return nil, errors.Errorf("failed to start VM: %w", err)
	}

	_, err = s.InitilizeRemoteAgent(ctx, connect.NewRequest(&ec1v1.InitializeRemoteAgentRequest{
		NodeIp:      vm.IpAddress,
		Arch:        ptr(req.Msg.GetArch()),
		Os:          ptr(req.Msg.GetOs()),
		SshUsername: vm.SshUsername,
		SshPassword: vm.SshPassword,
	}))
	if err != nil {
		return nil, errors.Errorf("failed to initialize remote agent: %w", err)
	}

	// TODO: register the VM with the management server

	return connect.NewResponse(&ec1v1.InitializeLocalAgentInsideLocalVMResponse{}), nil
}

// StartLocalAgent starts the agent on the local machine in a separate process
func GetAgentBinary(ctx context.Context, archstr, osstr string) (string, error) {
	path := fmt.Sprintf("build/agent-%s-%s", osstr, archstr)

	_, err := os.Stat(path)
	if err != nil {
		return "", errors.Errorf("checking for agent binary: %w", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.Errorf("getting absolute path: %w", err)
	}

	return absPath, nil
}

func (s *Server) runCommandViaSSH(ctx context.Context, sshClient *ssh.Client, command string) (string, error) {
	session, err := sshClient.NewSession()
	if err != nil {
		return "", errors.Errorf("failed to create new SSH session: %w", err)
	}
	defer session.Close()

	pr, pw := io.Pipe()
	session.Stdout = pw
	session.Stderr = pw

	err = session.Run(command)

	if err != nil {
		output, _ := io.ReadAll(pr)
		return "", errors.Errorf("failed to run command: %w, output: %s", err, string(output))
	}

	output, err := io.ReadAll(pr)
	if err != nil {
		return "", errors.Errorf("failed to read command output: %w", err)
	}

	return string(output), nil
}

// InitilizeRemoteAgent implements v1poc1connect.ManagementServiceHandler.
func (s *Server) InitilizeRemoteAgent(ctx context.Context, req *connect.Request[ec1v1.InitializeRemoteAgentRequest]) (*connect.Response[ec1v1.InitializeRemoteAgentResponse], error) {

	sshClient, err := ssh.Dial("tcp", req.Msg.GetNodeIp(), &ssh.ClientConfig{
		User: req.Msg.GetSshUsername(),
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Msg.GetSshPassword()),
		},
	})
	if err != nil {
		return nil, errors.Errorf("failed to dial SSH: %w", err)
	}
	defer sshClient.Close()

	agentBinary, err := GetAgentBinary(ctx, req.Msg.GetArch(), req.Msg.GetOs())
	if err != nil {
		return nil, errors.Errorf("getting agent binary: %w", err)
	}

	// upload the agent binary to the agent
	err = s.uploadViaSSH(ctx, sshClient, agentBinary, "/bin/ec1-agent")
	if err != nil {
		return nil, errors.Errorf("failed to upload agent binary: %w", err)
	}

	// call the agent binary "setup" command on the remote machine
	output, err := s.runCommandViaSSH(ctx, sshClient, "/bin/ec1-agent")
	if err != nil {
		return nil, errors.Errorf("failed to run setup command: %w, output: %s", err, output)
	}

	fmt.Printf("Setup command output: %s\n", output)

	id := id.NewID("agent")

	output, err = s.runCommandViaSSH(ctx, sshClient, "echo "+id.String()+" > /etc/ec1/agent-id")
	if err != nil {
		return nil, errors.Errorf("failed to write agent ID: %w, output: %s", err, output)
	}

	// now that the agent is setup, we trigger systemd to start the agent
	output, err = s.runCommandViaSSH(ctx, sshClient, "systemctl start ec1-agent")
	if err != nil {
		return nil, errors.Errorf("failed to start agent: %w, output: %s", err, output)
	}

	// the agent should have started and contacted us - make a temporary agent client to confirm
	// agentClient := v1poc1connect.NewAgentServiceClient(http.DefaultClient, req.Msg.GetHostAddress())

	agentData := RegisteredAgent{
		ID:     id.String(),
		NodeIP: req.Msg.GetNodeIp(),
		// AgentConnectUri: req.Msg.GetHostAddress() + ":9091/ec1-agent",
		IsInMemory: false,
	}

	err = s.db.SaveAgent(agentData)
	if err != nil {
		return nil, errors.Errorf("failed to save agent: %w", err)
	}

	// wait for the agent to register
	time.Sleep(1 * time.Second)

	err = s.startProbeLoop(ctx, agentData)
	if err != nil {
		return nil, errors.Errorf("failed to start probe loop: %w", err)
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
// 			Error:   ptr(fmt.Sprintf("Agent with ID %s not found", agentID)),
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
