package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	slogctx "github.com/veqryn/slog-context"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

func main() {

	pid := os.Getpid()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logging.SetupSlogSimple(ctx)

	ctx = slogctx.With(ctx, slog.Int("pid", pid))

	slog.InfoContext(ctx, "ec1init started", "args", os.Args)

	if pid == 1 {
		ctx = slogctx.With(ctx, slog.String("mode", "initramfs"))
		err := initramfsInit(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "problem initializing initramfs", "error", err)
			os.Exit(1)
		}
	} else {
		ctx = slogctx.With(ctx, slog.String("mode", "vsock"))

		defer func() {
			slog.InfoContext(ctx, "shutting down vsock server")
		}()

		err := serveRawVsock(ctx, ec1init.VsockPort)
		if err != nil {
			slog.ErrorContext(ctx, "problem serving vsock", "error", err)
			os.Exit(1)
		}
	}
}

func serveRawVsock(ctx context.Context, port int) error {

	// os.Mkdir("/ec1", 0755)

	// // just keep printing a message
	// go func() {
	// 	count := 0
	// 	for {
	// 		count++
	// 		// check if /dev/vsock exists
	// 		l, err := vsock.ListenContextID(3, uint32(2020+count), nil)
	// 		if err != nil {
	// 			log.Printf("Error listening on vsock: %v", err)
	// 		} else {
	// 			log.Printf("Listening on vsock")

	// 		}

	// 		log.Printf("Waiting for vsock to be ready (listenErr=%v)", err)
	// 		time.Sleep(100 * time.Millisecond)

	// 		if l != nil {
	// 			l.Close()
	// 		}
	// 	}
	// }()

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutor(1024)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}

func triggerVsock(ctx context.Context) error {
	pid, _, err := syscall.StartProcess(os.Args[0], os.Args[1:], &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	})
	if err != nil {
		return errors.Errorf("starting process: %w", err)
	}

	slog.InfoContext(ctx, "started vsock")

	// // make a pid file, stdout and
	pidFile, err := os.Create(filepath.Join(ec1init.Ec1AbsPath, ec1init.VsockPidFile))
	if err != nil {
		return errors.Errorf("creating pid file: %w", err)
	}

	pidFile.WriteString(strconv.Itoa(pid))
	pidFile.Close()

	return nil
}

func initramfsInit(ctx context.Context) error {

	slog.InfoContext(ctx, "initramfs init started, mounting rootfs")

	mounts := [][]string{
		{"mkdir", "-p", "/proc", "/sys", "/dev", ec1init.NewRootAbsPath, "/run", "/run/ovl", ec1init.Ec1AbsPath},
		{"mount", "-t", "devtmpfs", "devtmpfs", "/dev"},
		{"mount", "-t", "proc", "proc", "/proc"},
		{"mount", "-t", "sysfs", "sysfs", "/sys"},
		{"mount", "-t", "tmpfs", "tmpfs", "/run"},
		{"mkdir", "-p", "/dev/pts", "/dev/shm"},
		{"mount", "-t", "devpts", "devpts", "/dev/pts"},
		{"mount", "-t", "tmpfs", "tmpfs", "/dev/shm"},
		{"mount", "-t", "virtiofs", ec1init.RootfsVirtioTag, ec1init.NewRootAbsPath},
		{"mount", "-t", "virtiofs", ec1init.Ec1VirtioTag, ec1init.Ec1AbsPath},
	}

	// mount dirs
	for _, mount := range mounts {
		err := execCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	slog.InfoContext(ctx, "triggering vsock")

	err := triggerVsock(ctx)
	if err != nil {
		return errors.Errorf("triggering vsock: %w", err)
	}

	slog.InfoContext(ctx, "loading manifest")

	manifest, err := loadManifest(ctx)
	if err != nil {
		return errors.Errorf("loading manifest: %w", err)
	}

	slog.InfoContext(ctx, "loading init cmd line args")

	// switch_root to the new rootfs, calling the entrypoint

	cmd, err := loadInitCmdLineArgs(ctx)
	if err != nil {
		return errors.Errorf("loading init cmd line args: %w", err)
	}

	moveMounts := [][]string{
		{"mount", "-o", "move", "/dev", ec1init.NewRootAbsPath + "/dev"},
		{"mount", "-o", "move", "/proc", ec1init.NewRootAbsPath + "/proc"},
		{"mount", "-o", "move", "/sys", ec1init.NewRootAbsPath + "/sys"},
		{"mount", "-o", "move", "/run", ec1init.NewRootAbsPath + "/run"},
		{"mount", "-o", "move", "/ec1", ec1init.NewRootAbsPath + ec1init.Ec1AbsPath},
	}

	for _, mount := range moveMounts {
		err := execCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	err = switchRoot(ctx, manifest, cmd)
	if err != nil {
		return errors.Errorf("switching root: %w", err)
	}

	panic("unreachable, we should have switched root")
}

func loadInitCmdLineArgs(ctx context.Context) ([]string, error) {
	initCmdLineArgs, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.UserProvidedCmdline))
	if err != nil {
		if os.IsNotExist(err) {
			slog.WarnContext(ctx, "user provided cmdline not found, ignoring")
			return []string{}, nil
		}
		return nil, errors.Errorf("loading user provided cmdline: %w", err)
	}

	var args []string
	err = json.Unmarshal(initCmdLineArgs, &args)
	if err != nil {
		return nil, errors.Errorf("unmarshalling user provided cmdline: %w", err)
	}

	return args, nil
}

func switchRoot(ctx context.Context, manifest *v1.Image, cmd []string) error {

	entrypoint := manifest.Config.Entrypoint
	if len(cmd) == 0 {
		cmd = manifest.Config.Cmd
	}

	env := manifest.Config.Env
	env = append(env, "PATH=/usr/sbin:/usr/bin:/sbin:/bin")

	argc := "/bin/busybox"
	argv := append([]string{"switch_root", ec1init.NewRootAbsPath}, entrypoint...)
	argv = append(argv, cmd...)

	slog.InfoContext(ctx, "switching root - godspeed little process", "rootfs", ec1init.NewRootAbsPath, "argv", argv)

	if err := syscall.Exec(argc, argv, env); err != nil {
		return errors.Errorf("Failed to exec %v %v: %v", entrypoint, cmd, err)
	}

	panic("unreachable, we hand off to the entrypoint")

}

func loadManifest(ctx context.Context) (*v1.Image, error) {
	manifest, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerManifestFile))
	if err != nil {
		return nil, errors.Errorf("reading manifest: %w", err)
	}

	var image v1.Image
	err = json.Unmarshal(manifest, &image)
	if err != nil {
		return nil, errors.Errorf("unmarshalling manifest: %w", err)
	}

	log.Printf("Loaded manifest: %+v", image)

	return &image, nil
}

func execCmdForwardingStdio(ctx context.Context, cmds ...string) error {
	if len(cmds) == 0 {
		return errors.Errorf("no command to execute")
	}

	argc := "/bin/busybox"
	argv := cmds

	// argc := cmds[0]
	// var argv []string
	// if len(cmds) > 1 {
	// 	argv = cmds[1:]
	// } else {
	// 	argv = []string{}
	// }
	slog.DebugContext(ctx, "executing command", "argc", argc, "argv", argv)
	cmd := exec.CommandContext(ctx, argc, argv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Cloneflags: syscall.CLONE_NEWNS,
	}
	cmd.Stdin = bytes.NewBuffer(nil) // set to avoid reading /dev/null since it may not be mounted
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Errorf("running busybox command (stdio was copied to the parent process): %v: %w", cmds, err)
	}

	return nil
}
