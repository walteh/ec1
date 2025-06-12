package containerd

import (
	"context"
	"log/slog"
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/api/runtime/task/v3"
	"github.com/containerd/containerd/v2/core/runtime"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/containerd/v2/pkg/protobuf"
	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/containerd/containerd/v2/pkg/shutdown"
	"github.com/containerd/errdefs"
	"github.com/containerd/errdefs/pkg/errgrpc"
	"github.com/containerd/log"
	"github.com/containerd/ttrpc"
	"github.com/creack/pty"
	"gitlab.com/tozd/go/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	ptypes "github.com/containerd/containerd/v2/pkg/protobuf/types"

	"github.com/walteh/ec1/pkg/logging/valuelog"
	"github.com/walteh/ec1/pkg/vmm"
	"github.com/walteh/ec1/pkg/vmm/vf"
)

type service struct {
	containersMu sync.Mutex
	containers   map[string]*container
	events       chan interface{}
	sd           shutdown.Service
	hypervisor   vmm.Hypervisor[*vf.VirtualMachine]
	pid          int
}

func NewTaskService(ctx context.Context, publisher shim.Publisher, sd shutdown.Service) (taskService, error) {
	s := service{
		containers: make(map[string]*container),
		sd:         sd,
		events:     make(chan interface{}, 128),
		hypervisor: vf.NewHypervisor(),
		pid:        os.Getpid(),
	}

	go s.forward(ctx, publisher)
	return &s, nil
}

func (s *service) forward(ctx context.Context, publisher shim.Publisher) {
	ns, _ := namespaces.Namespace(ctx)
	ctx = namespaces.WithNamespace(context.Background(), ns)
	for e := range s.events {
		err := publisher.Publish(ctx, runtime.GetTopic(e), e)
		if err != nil {
			log.G(ctx).WithError(err).Error("post event")
		}
	}
	_ = publisher.Close()
}

func (s *service) setContainer(ctx context.Context, c *container) error {
	s.containersMu.Lock()
	defer s.containersMu.Unlock()

	if _, ok := s.containers[c.request.ID]; ok {
		return errgrpc.ToGRPCf(errdefs.ErrAlreadyExists, "container already exists: %s", c.request.ID)
	}

	s.containers[c.request.ID] = c
	return nil
}

func (s *service) getContainer(ctx context.Context, id string) (*container, error) {
	s.containersMu.Lock()
	defer s.containersMu.Unlock()

	if id == "" {
		return nil, errgrpc.ToGRPCf(errdefs.ErrNotFound, "container not created")
	}

	c := s.containers[id]
	if c == nil {
		slog.ErrorContext(ctx, "container not found", "id", id)
		return nil, errgrpc.ToGRPCf(errdefs.ErrNotFound, "container not created")
	}
	return c, nil
}

func getContainerProcess(ctx context.Context, s *service, input interface {
	GetID() string
	GetExecID() string
}) (*container, *managedProcess, error) {
	c, err := s.getContainer(ctx, input.GetID())
	if err != nil {
		return nil, nil, err
	}

	p, err := c.getProcess(ctx, input.GetExecID())
	if err != nil {
		return nil, nil, err
	}

	return c, p, nil
}

func (s *service) deleteContainer(ctx context.Context, id string) {
	s.containersMu.Lock()
	defer s.containersMu.Unlock()

	delete(s.containers, id)
}

func (s *service) RegisterTTRPC(server *ttrpc.Server) error {
	task.RegisterTTRPCTaskService(server, s)
	return nil
}

func protobufTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return protobuf.ToTimestamp(t)
}

func (s *service) State(ctx context.Context, request *task.StateRequest) (*task.StateResponse, error) {

	c, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, err
	}

	state := p.getStatus()

	resp := &task.StateResponse{
		ID:         request.ID,
		Bundle:     c.bundlePath,
		Pid:        uint32(p.pid),
		Status:     state.Status,
		Stdin:      p.io.stdinPath,
		Stdout:     p.io.stdoutPath,
		Stderr:     p.io.stderrPath,
		Terminal:   c.spec.Process.Terminal,
		ExitedAt:   protobufTimestamp(state.ExitedAt),
		ExitStatus: uint32(state.ExitCode),
		ExecID:     request.ExecID,
	}

	slog.InfoContext(ctx, "STATE", "response", valuelog.NewPrettyValue(resp), "pid", p.pid, "status", state.Status)

	// For VM-based processes, we use the stored pid

	return resp, nil
}

func (s *service) Create(ctx context.Context, request *task.CreateTaskRequest) (_ *task.CreateTaskResponse, retErr error) {

	slog.InfoContext(ctx, "CREATE", "request", valuelog.NewPrettyValue(request))

	specPath := path.Join(request.Bundle, oci.ConfigFilename)

	spec, err := oci.ReadSpec(specPath)
	if err != nil {
		return nil, errors.Errorf("reading spec: %w", err)
	}

	// rootfs, err := mount.CanonicalizePath(spec.Root.Path)
	// if err != nil {
	// 	return nil, errors.Errorf("canonicalizing rootfs: %w", err)
	// }

	// canonicalizedRootfs, err := mount.CanonicalizePath(request.Rootfs[0].Target)
	// if err != nil {
	// 	return nil, errors.Errorf("canonicalizing rootfs at %s/: %w", request.Rootfs, err)
	// }

	// slog.InfoContext(ctx, "CREATE", "spec.Root.Path", spec.Root.Path, "spec.Root.Path[canonicalized]", rootfs, "request.Rootfs", request.Rootfs, "canonicalizedRootfs")

	// Workaround for 104-char limit of UNIX socket path
	// shortenedRootfsPath, err := shortenPath(rootfs)
	// if err != nil {
	// 	return nil, errors.Errorf("shortening rootfs at %s/: %w", rootfs, err)
	// }

	// dnsSocketPath := path.Join(shortenedRootfsPath, "var", "run", "mDNSResponder")

	// start := time.Now()

	c, primaryProcess, err := NewContainer(ctx, s.hypervisor, spec, request)
	if err != nil {
		return nil, errors.Errorf("creating vm: %w", err)
	}

	if err := s.setContainer(ctx, c); err != nil {
		return nil, errors.Errorf("setting container: %w", err)
	}

	s.events <- &events.TaskCreate{
		ContainerID: request.ID,
		Bundle:      c.bundlePath,
		Rootfs:      request.Rootfs,
		IO: &events.TaskIO{
			Stdin:    request.Stdin,
			Stdout:   request.Stdout,
			Stderr:   request.Stderr,
			Terminal: c.spec.Process.Terminal,
		},
		Checkpoint: request.Checkpoint,
		Pid:        uint32(primaryProcess.pid),
	}

	return &task.CreateTaskResponse{
		Pid: uint32(primaryProcess.pid),
	}, nil
}

func (s *service) Start(ctx context.Context, request *task.StartRequest) (*task.StartResponse, error) {

	c, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	if err := c.vm.Start(ctx); err != nil {
		return nil, errors.Errorf("starting vm: %w", err)
	}

	go func() {
		err := c.vm.Wait(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "vm run complete with error", "error", err)
		} else {
			slog.InfoContext(ctx, "vm run complete")
		}
	}()

	if err := p.StartSignalRunner(ctx); err != nil {
		return nil, errors.Errorf("starting signal runner: %w", err)
	}

	// // Set a fake PID for compatibility (VM processes don't have host PIDs)
	// p.pid = os.Getpid()
	// p.status = taskt.Status_RUNNING // Set as running immediately

	// Start VM creation asynchronously with better error handling
	// go func() {

	// 	// Use background context since the request context might be cancelled
	// 	vmCtx := context.Background()
	// 	attrs := slogctx.ExtractAppended(ctx, time.Now(), slog.LevelDebug, "start_vm_done")
	// 	for _, attr := range attrs {
	// 		vmCtx = slogctx.Append(vmCtx, attr)
	// 	}

	// 	defer func() {
	// 		defer func() {
	// 			close(p.waitblock)
	// 		}()

	// 		if r := recover(); r != nil {
	// 			log.G(ctx).WithField("panic", r).WithField("pid", p.pid).Error("PANIC in VM creation")
	// 			p.status = taskt.Status_STOPPED
	// 			p.exitStatus = 1
	// 			// Send task exit event to notify containerd of failure
	// 			s.events <- &events.TaskExit{
	// 				ContainerID: request.ID,
	// 				ID:          request.ExecID,
	// 				Pid:         uint32(p.pid),
	// 				ExitStatus:  1,
	// 				ExitedAt:    protobuf.ToTimestamp(time.Now()),
	// 			}
	// 			// Ensure waitblock is closed in case of panic
	// 			select {
	// 			case <-p.waitblock:
	// 				// Already closed
	// 			default:
	// 				close(p.waitblock)
	// 			}
	// 		}
	// 		log.G(ctx).Info("START_VM_WAIT_DONE")
	// 	}()

	// 	// log.G(ctx).Info("Starting VM creation")
	// 	// // Create and start the VM for this container
	// 	// if err := c.createVM(vmCtx, c.spec, request.ID, c.rootfs, c.primary.io); err != nil {
	// 	// 	log.G(ctx).WithError(err).Error("failed to create VM")
	// 	// 	p.status = taskt.Status_STOPPED
	// 	// 	p.exitStatus = 1

	// 	// 	// Send task exit event to notify containerd of failure
	// 	// 	s.events <- &events.TaskExit{
	// 	// 		ContainerID: request.ID,
	// 	// 		ID:          request.ExecID,
	// 	// 		Pid:         uint32(p.pid),
	// 	// 		ExitStatus:  1,
	// 	// 		ExitedAt:    protobuf.ToTimestamp(time.Now()),
	// 	// 	}
	// 	// 	// Close waitblock since start won't be called
	// 	// 	close(p.waitblock)
	// 	// 	return
	// 	// }

	// 	// Start the process in the VM now that VM is created
	// 	rs, err := p.runSignal(ctx)
	// 	if err != nil {
	// 		log.G(ctx).WithError(err).Error("failed to start process in VM")
	// 		p.status = taskt.Status_STOPPED
	// 		p.exitStatus = 1
	// 		// Send task exit event to notify containerd of failure
	// 		s.events <- &events.TaskExit{
	// 			ContainerID: request.ID,
	// 			ID:          request.ExecID,
	// 			Pid:         uint32(p.pid),
	// 			ExitStatus:  1,
	// 			ExitedAt:    protobuf.ToTimestamp(time.Now()),
	// 		}
	// 		// Close waitblock since start failed
	// 		close(p.waitblock)
	// 		return
	// 	}

	// 	log.G(ctx).Info("VM and process started successfully")
	// }()

	// Return immediately - VM will boot in background
	s.events <- &events.TaskStart{
		ContainerID: request.ID,
		Pid:         uint32(p.pid),
	}

	return &task.StartResponse{
		Pid: uint32(p.pid),
	}, nil
}

func (s *service) Delete(ctx context.Context, request *task.DeleteRequest) (*task.DeleteResponse, error) {

	c, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	if err := c.destroy(); err != nil {
		log.G(ctx).WithError(err).Warn("failed to cleanup container")
	}

	s.deleteContainer(ctx, request.ID)

	state := p.getStatus()

	var pid uint32 = uint32(p.pid)

	s.events <- &events.TaskDelete{
		ContainerID: request.ID,
		ExitedAt:    protobufTimestamp(state.ExitedAt),
		ExitStatus:  uint32(state.ExitCode),
		ID:          request.ID,
		Pid:         pid,
	}

	return &task.DeleteResponse{
		ExitedAt:   protobufTimestamp(state.ExitedAt),
		ExitStatus: uint32(state.ExitCode),
		Pid:        pid,
	}, nil
}

func (s *service) Pids(ctx context.Context, request *task.PidsRequest) (*task.PidsResponse, error) {

	return nil, errdefs.ErrNotImplemented
}

func (s *service) Pause(ctx context.Context, request *task.PauseRequest) (*ptypes.Empty, error) {

	return nil, errdefs.ErrNotImplemented
}

func (s *service) Resume(ctx context.Context, request *task.ResumeRequest) (*ptypes.Empty, error) {

	return nil, errdefs.ErrNotImplemented
}

func (s *service) Checkpoint(ctx context.Context, request *task.CheckpointTaskRequest) (*ptypes.Empty, error) {

	return nil, errdefs.ErrNotImplemented
}

func (s *service) Kill(ctx context.Context, request *task.KillRequest) (*ptypes.Empty, error) {

	slog.InfoContext(ctx, "KILL", "request", valuelog.NewPrettyValue(request))

	c, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	if p.runningCmd != nil && p.runningCmd.exitCode == 0 {
		if err := p.SendSignalToRunningCmd(syscall.Signal(request.Signal)); err != nil {
			return nil, errors.Errorf("sending signal to running command: %w", err)
		}
	}

	if err := p.destroy(); err != nil {
		return nil, errors.Errorf("destroying process: %w", err)
	}

	if err := c.destroy(); err != nil {
		return nil, errors.Errorf("destroying container: %w", err)
	}

	s.deleteContainer(ctx, request.ID)

	// if err := p.io.Close(); err != nil {
	// 	return nil, errors.Errorf("closing io: %w", err)
	// }

	// if p.id != "primary" {
	// 	if err != nil {
	// 		return nil, errors.Errorf("getting process: %w", err)
	// 	}

	// 	// TODO: Do we care about error here?
	// 	_ = p.kill(syscall.Signal(request.Signal))
	// }

	return &ptypes.Empty{}, nil
}

func (s *service) Exec(ctx context.Context, request *task.ExecProcessRequest) (_ *ptypes.Empty, retErr error) {

	c, _, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	aux, err := c.AddProcess(ctx, request)
	if err != nil {
		return nil, errors.Errorf("adding process: %w", err)
	}

	defer func() {
		if retErr != nil {
			if err := aux.destroy(); err != nil {
				log.G(ctx).WithError(err).Warn("failed to cleanup aux")
			}
		}
	}()

	s.events <- &events.TaskExecAdded{
		ContainerID: request.ID,
		ExecID:      request.ExecID,
	}

	return &ptypes.Empty{}, nil
}

func (s *service) ResizePty(ctx context.Context, request *task.ResizePtyRequest) (*ptypes.Empty, error) {

	c, err := s.getContainer(ctx, request.ID)
	if err != nil {
		return nil, errors.Errorf("getting container: %w", err)
	}

	p, err := c.getProcess(ctx, request.ExecID)
	if err != nil {
		return nil, errors.Errorf("getting process: %w", err)
	}

	if con := p.getConsoleL(); con != nil {
		if err = pty.Setsize(con, &pty.Winsize{Cols: uint16(request.Width), Rows: uint16(request.Height)}); err != nil {
			return nil, errors.Errorf("setting pty size: %w", err)
		}
	}

	return &ptypes.Empty{}, nil
}

func (s *service) CloseIO(ctx context.Context, request *task.CloseIORequest) (*ptypes.Empty, error) {

	_, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	if stdin := p.io.stdin; stdin != nil {
		_ = stdin.Close()
	}

	return &ptypes.Empty{}, nil
}

func (s *service) Update(ctx context.Context, request *task.UpdateTaskRequest) (*ptypes.Empty, error) {
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Wait(ctx context.Context, request *task.WaitRequest) (*task.WaitResponse, error) {

	_, p, err := getContainerProcess(ctx, s, request)
	if err != nil {
		return nil, errors.Errorf("getting container process: %w", err)
	}

	if p.runningCmd == nil {
		return nil, errdefs.ErrUnavailable
	}

	// libdispatch.DispatchMain()

	exitCode, err := p.runningCmd.Serve(ctx)
	if err != nil {
		return nil, errors.Errorf("serving signal: %w", err)
	}

	slog.InfoContext(ctx, "wait has completed", "exitCode", exitCode)

	return &task.WaitResponse{
		ExitedAt:   protobuf.ToTimestamp(p.runningCmd.exitedAt),
		ExitStatus: uint32(exitCode),
	}, nil
}

func (s *service) Stats(ctx context.Context, request *task.StatsRequest) (*task.StatsResponse, error) {
	return nil, errdefs.ErrNotImplemented
}

func (s *service) Connect(ctx context.Context, request *task.ConnectRequest) (*task.ConnectResponse, error) {

	container, err := s.getContainer(ctx, request.ID)
	if err != nil {
		return nil, errors.Errorf("getting container: %w", err)
	}

	// var pid int
	// if _, p, err := getContainerProcess(ctx, s, request); err == nil {
	// 	pid = p.pid
	// }

	return &task.ConnectResponse{
		ShimPid: uint32(os.Getpid()),
		TaskPid: uint32(container.pid),
		Version: "v2",
	}, nil
}

func (s *service) Shutdown(ctx context.Context, request *task.ShutdownRequest) (*ptypes.Empty, error) {

	// s.containersMu.Lock()
	// defer s.containersMu.Unlock()

	if len(s.containers) > 0 {
		// todo: do we need to kill them all first?
		return &ptypes.Empty{}, nil
	}

	s.sd.Shutdown()

	return &ptypes.Empty{}, nil
}
