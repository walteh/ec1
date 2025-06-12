package vmm

import (
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/containerd/ttrpc"
	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/gvnet"
)

const (
	ExecVSockPort = 2019
)

type Hypervisor[VM VirtualMachine] interface {
	NewVirtualMachine(ctx context.Context, id string, opts *NewVMOptions, bl Bootloader) (VM, error)
	OnCreate() <-chan VM
	EncodeLinuxInitramfs(ctx context.Context, initramfs io.Reader) (io.ReadCloser, error)
	EncodeLinuxKernel(ctx context.Context, kernel io.Reader) (io.ReadCloser, error)
	EncodeLinuxRootfs(ctx context.Context, rootfs io.Reader) (io.ReadCloser, error)
	InitramfsCompression() archives.Compression
}

type RunningVM[VM VirtualMachine] struct {
	// streamExecReady bool
	// manager                *VSockManager
	guestServiceConnection harpoonv1.TTRPCGuestServiceClient
	bootloader             Bootloader

	// streamexec   *streamexec.Client
	portOnHostIP uint16
	wait         chan error
	vm           VM
	netdev       gvnet.Proxy
	workingDir   string
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	// connStatus      <-chan VSockManagerState
	start time.Time
}

// func (r *RunningVM[VM]) guestService(ctx context.Context) harpoonv1.TTRPCGuestServiceClient {
// 	r.guestServiceConnectionMu.Lock()
// 	defer r.guestServiceConnectionMu.Unlock()

// 	if r.guestServiceConnection == nil {
// 		conn, err := r.vm.VSockConnect(ctx, uint32(ec1init.VsockPort))
// 		if err != nil {
// 			slog.Error("failed to dial vsock", "error", err)
// 			return nil
// 		}
// 		r.guestServiceConnection = harpoonv1.NewTTRPCGuestServiceClient(ttrpc.NewClient(conn))
// 	}

// 	return r.guestServiceConnection
// }

func connectToVsockWithRetry(ctx context.Context, vm VirtualMachine, port uint32) (net.Conn, error) {

	ticker := time.NewTicker(100 * time.Millisecond)
	timeout := time.NewTimer(3 * time.Second)
	defer ticker.Stop()
	defer timeout.Stop()

	lastError := error(errors.Errorf("initial error"))

	for {
		select {
		case <-ticker.C:
			conn, err := vm.VSockConnect(ctx, port)
			if err != nil {
				lastError = err
				continue
			}
			return conn, nil
		case <-timeout.C:
			return nil, errors.Errorf("timeout waiting for guest service connection: %w", lastError)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (r *RunningVM[VM]) GuestService(ctx context.Context) (harpoonv1.TTRPCGuestServiceClient, error) {
	if r.guestServiceConnection != nil {
		return r.guestServiceConnection, nil
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	timeout := time.NewTimer(3 * time.Second)
	defer ticker.Stop()
	defer timeout.Stop()

	lastError := error(errors.Errorf("initial error"))

	for {
		select {
		case <-ticker.C:
			conn, err := r.vm.VSockConnect(ctx, uint32(ec1init.VsockPort))
			if err != nil {
				lastError = err
				continue
			}
			r.guestServiceConnection = harpoonv1.NewTTRPCGuestServiceClient(ttrpc.NewClient(conn, ttrpc.WithClientDebugging(), ttrpc.WithOnCloseError(func(err error) {
				slog.Error("guest service connection closed", "error", err)
			})))
			return r.guestServiceConnection, nil
		case <-timeout.C:
			return nil, errors.Errorf("timeout waiting for guest service connection: %w", lastError)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// func NewRunningContainerdVM[VM VirtualMachine](ctx context.Context, vm VM, portOnHostIP uint16, start time.Time, workingDir string, ec1DataDir string, cfg *ContainerizedVMConfig) *RunningVM[VM] {

// 	return &RunningVM[VM]{
// 		start:                  start,
// 		vm:                     vm,
// 		stdin:                  cfg.StdinReader,
// 		stdout:                 cfg.StdoutWriter,
// 		stderr:                 cfg.StderrWriter,
// 		portOnHostIP:           portOnHostIP,
// 		wait:                   make(chan error, 1),
// 		manager:                nil,
// 		guestServiceConnection: nil,
// 		streamexec:             nil,
// 		workingDir:             workingDir,
// 		netdev:                 nil,
// 	}
// }

func (r *RunningVM[VM]) ForwardStdio(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return ForwardStdio(ctx, r.vm, stdin, stdout, stderr)
}

// func NewRunningVM[VM VirtualMachine](ctx context.Context, vm VM, portOnHostIP uint16, start time.Time) *RunningVM[VM] {

// 	transporz := NewVSockManager(func(ctx context.Context) (io.ReadWriteCloser, error) {
// 		return vm.VSockConnect(ctx, uint32(ExecVSockPort))
// 	})

// 	tfunc := transport.NewFunctionTransport(func() (io.ReadWriteCloser, error) {
// 		slog.Info("dialing vm transport")
// 		conn, err := transporz.Dial(ctx)
// 		if err != nil {
// 			slog.Error("failed to dial vm transport", "error", err)
// 			return nil, errors.Errorf("dialing vm transport: %w", err)
// 		}
// 		slog.Info("dialed vm transport")
// 		return conn, nil
// 	}, nil)

// 	client := streamexec.NewClient(tfunc, func(conn io.ReadWriter) protocol.Protocol {
// 		return protocol.NewFramedProtocol(conn)
// 	})

// 	go func() {
// 		slog.Info("dialing vm")
// 		err := client.Connect(ctx)
// 		if err != nil {
// 			slog.Error("failed to connect to vm", "error", err)
// 		} else {
// 			slog.Info("connected to vm")
// 		}

// 	}()

// 	// connStatus := transporz.AddStateNotifier()

// 	return &RunningVM[VM]{
// 		start:   start,
// 		vm:      vm,
// 		manager: transporz,
// 		// connStatus:      connStatus,
// 		portOnHostIP: portOnHostIP,
// 		wait:         make(chan error, 1),
// 		// streamExecReady: false,
// 		streamexec: client,
// 	}
// }

func (r *RunningVM[VM]) WaitOnVmStopped() error {
	return <-r.wait
}

// func (r *RunningVM[VM]) WaitOnVMReadyToExec() <-chan struct{} {
// 	ch := make(chan struct{})

// 	if r.manager.State() == StateConnected {
// 		close(ch)
// 		return ch
// 	}
// 	check := r.manager.AddStateNotifier()
// 	go func() {
// 		defer close(check)
// 		for {
// 			select {
// 			case <-check:
// 				if r.manager.State() == StateConnected {
// 					close(ch)
// 				}
// 			}
// 		}
// 	}()
// 	return ch
// }

// func (r *RunningVM[VM]) WaitOnVMReady(ctx context.Context) <-chan struct{} {
// 	ch := make(chan struct{})

// 	cacheDir, err := host.EmphiricalVMCacheDir(ctx, r.vm.ID())

// 	// keep checking the ready file
// 	readyFile := filepath.Join(cacheDir, ec1init.ContainerReadyFile)
// 	dat, err := os.ReadFile(readyFile)
// 	if err != nil {
// 		slog.Error("problem reading ready file", "error", err)
// 		return ch
// 	}
// 	return ch
// }

func (r *RunningVM[VM]) VM() VM {
	return r.vm
}

func (r *RunningVM[VM]) PortOnHostIP() uint16 {
	return r.portOnHostIP
}

func (r *RunningVM[VM]) RunCommandSimple(ctx context.Context, command string) ([]byte, []byte, int64, error) {
	guestService, err := r.GuestService(ctx)
	if err != nil {
		return nil, nil, 0, errors.Errorf("getting guest service: %w", err)
	}

	fields := strings.Fields(command)

	argc := fields[0]
	argv := []string{}
	for _, field := range fields[1:] {
		argv = append(argv, field)
	}

	// req, err := harpoonv1.NewValidatedRunRequest(func(b *harpoonv1.RunRequest_builder) {
	// 	// b.Stdin = stdinData
	// })
	req, err := harpoonv1.NewRunCommandRequestE(func(b *harpoonv1.RunCommandRequest_builder) {
		b.Argc = ptr(argc)
		b.Argv = argv
		b.EnvVars = map[string]string{}
		b.Stdin = []byte{}
		b.UseEntrypoint = ptr(false)
	})
	if err != nil {
		return nil, nil, 0, err
	}

	exec, err := guestService.RunCommand(ctx, req)
	if err != nil {
		return nil, nil, 0, err
	}

	return exec.GetStdout(), exec.GetStderr(), int64(exec.GetExitCode()), nil
}

// func (r *RunningVM[VM]) Exec(ctx context.Context, command string) (stdout []byte, stderr []byte, errorcode []byte, err error) {
// 	if r.manager.State() != StateConnected {
// 		return nil, nil, nil, errors.New("stream exec not ready")
// 	}
// 	return r.streamexec.ExecuteCommand(ctx, command)
// }

// func (r *RunningVM[VM]) RunWithStdio(ctx context.Context, term chan bool, stdin io.Reader, stdout io.Writer, stderr io.Writer) (errorcode int32, err error) {

// 	slog.InfoContext(ctx, "RunWithStdio: starting")

// 	defer func() {
// 		if r := recover(); r != nil {
// 			slog.ErrorContext(ctx, "panic in RunWithStdio", "error", r)
// 			fmt.Fprintln(logging.GetDefaultLogWriter(), string(debug.Stack()))
// 			errorcode = -1
// 			err = errors.Errorf("panic in RunWithStdio: %v", r)
// 			return
// 		}
// 		slog.InfoContext(ctx, "RunWithStdio: finished")

// 	}()

// 	guestService, err := r.GuestService(ctx)
// 	if err != nil {
// 		return 0, errors.Errorf("getting guest service: %w", err)
// 	}

// 	slog.InfoContext(ctx, "RunWithStdio: got guest service")

// 	if term == nil {
// 		term = make(chan bool)
// 	}

// 	if stdin == nil {
// 		stdin = bytes.NewReader([]byte{})
// 	}

// 	slog.InfoContext(ctx, "RunWithStdio: creating exec request")
// 	//
// 	e, err := guestService.Exec(ctx)
// 	if err != nil {
// 		return 0, errors.Errorf("creating start request: %w", err)
// 	}

// 	slog.InfoContext(ctx, "RunWithStdio: sending start request")

// 	start := harpoonv1.NewExecRequest_WithStart(func(b *harpoonv1.ExecRequest_Start_builder) {
// 		b.Argc = ptr("")
// 		b.Argv = []string{}
// 		b.Stdin = ptr(true)
// 		b.EnvVars = map[string]string{}
// 	})
// 	if err != nil {
// 		return 0, errors.Errorf("creating start request: %w", err)
// 	}

// 	err = e.Send(start)
// 	if err != nil {
// 		return 0, errors.Errorf("sending start request: %w", err)
// 	}

// 	slog.InfoContext(ctx, "RunWithStdio: start request sent")

// 	terminate := func(force bool) {
// 		req := harpoonv1.NewExecRequest_WithTerminate(func(b *harpoonv1.ExecRequest_Terminate_builder) {
// 			b.Force = ptr(force)
// 		})
// 		if err != nil {
// 			slog.Error("failed to create terminate request", "error", err)
// 			return
// 		}
// 		err = e.Send(req)
// 		if err != nil {
// 			slog.Error("failed to send terminate to guest service", "error", err)
// 		}

// 	}

// 	defer terminate(false)

// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.ErrorContext(ctx, "panic in term goroutine", "error", r)
// 				slog.DebugContext(ctx, string(debug.Stack()))
// 				panic(r)
// 			}
// 		}()
// 		select {
// 		case <-ctx.Done():
// 			slog.InfoContext(ctx, "RunWithStdio: context done")
// 			terminate(false)
// 			return
// 		case force := <-term:
// 			slog.InfoContext(ctx, "RunWithStdio: term signal received", "force", force)
// 			terminate(force)
// 			return
// 		}

// 	}()

// 	slog.InfoContext(ctx, "RunWithStdio: stating goroutines")

// 	// copy stdin to the guest service
// 	go func() {
// 		defer func() {
// 			if r := recover(); r != nil {
// 				slog.ErrorContext(ctx, "panic in stdin goroutine", "error", r)
// 				slog.DebugContext(ctx, string(debug.Stack()))
// 				panic(r)
// 			}
// 		}()
// 		buf := make([]byte, 1024)
// 		for {

// 			n, err := stdin.Read(buf)
// 			if err != nil {
// 				if err == io.EOF {
// 					req := harpoonv1.NewExecRequest_WithStdin(func(b *harpoonv1.Bytestream_builder) {
// 						b.Data = buf[:n]
// 						b.Done = ptr(true)
// 					})
// 					if err != nil {
// 						slog.Error("failed to create exec request", "error", err)
// 						return
// 					}
// 					err = e.Send(req)
// 					if err != nil {
// 						slog.Error("failed to send stdin to guest service", "error", err)
// 					}
// 					return
// 				}
// 				slog.Error("failed to read stdin", "error", err)
// 			}
// 			if n == 0 {
// 				slog.InfoContext(ctx, "RunWithStdio: stdin read 0 bytes")
// 				return
// 			}

// 			req := harpoonv1.NewExecRequest_WithStdin(func(b *harpoonv1.Bytestream_builder) {
// 				b.Data = buf[:n]
// 				b.Done = ptr(false)
// 			})
// 			if err != nil {
// 				slog.Error("failed to create exec request", "error", err)
// 				return
// 			}

// 			err = e.Send(req)
// 			if err != nil {
// 				slog.Error("failed to send stdin to guest service", "error", err)
// 			}
// 		}
// 	}()

// 	slog.InfoContext(ctx, "RunWithStdio: starting recv loop")

// 	for {

// 		slog.InfoContext(ctx, "RunWithStdio: recv loop started")

// 		msg, err := e.Recv()
// 		if err != nil {
// 			slog.Error("failed to receive message from guest service", "error", err)
// 			return 0, errors.Errorf("failed to receive message from guest service: %w", err)
// 		}

// 		if msg.GetError() == nil {
// 			err = errors.Errorf("error: %s", msg.GetError().GetError())
// 		} else if msg.GetStderr() != nil {
// 			stderr.Write(msg.GetStderr().GetData())
// 		} else if msg.GetStdout() != nil {
// 			stdout.Write(msg.GetStdout().GetData())
// 		} else if msg.GetExit() != nil {
// 			errorcode = msg.GetExit().GetExitCode()
// 			return errorcode, err
// 		} else {
// 			err = errors.Errorf("unknown message: %v", msg)
// 		}
// 	}

// }

// type WrappedWriter struct {
// 	cli harpoonv1.TTRPCGuestService_ExecClient
// }

// func NewWrappedWriter(client harpoonv1.TTRPCGuestService_ExecClient) *WrappedWriter {
// 	return &WrappedWriter{
// 		client: client,
// 	}
// }

// func (w *WrappedWriter) Write(p []byte) (n int, err error) {
// 	req, err := harpoonv1.New(func(b *harpoonv1.Bytestream_builder) {
// 	err = w.cli.Send(&harpoonv1.Bytestream{
// 		Data: p,
// 	})
// 	if err != nil {
// 		return 0, err
// 	}
// 	return len(p), nil
// }

// type WrappedReader struct {
// 	cli harpoonv1.TTRPCGuestService_ExecClient
// }

// type WrappedReader struct {
// 	protocol protocol.Protocol
// 	msgType  protocol.MessageType
// }

// func NewWrappedReader(protocol protocol.Protocol, msgType protocol.MessageType) *WrappedReader {
// 	return &WrappedReader{
// 		protocol: protocol,
// 		msgType:  msgType,
// 	}
// }

// func (w *WrappedReader) Read(p []byte) (n int, err error) {
// 	payload, err := w.protocol.ReadMessage(w.msgType)
// 	if err != nil {
// 		return 0, errors.Errorf("reading message: %w", err)
// 	}
// 	return payload, nil
// }
