package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
		err := mountInitramfs(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "problem mounting initramfs", "error", err)
			os.Exit(1)
		}

		// if len(os.Args) > 1 && os.Args[1] == "vsock" {
		ctx = slogctx.With(ctx, slog.String("mode", "primary_vsock"))

		err = serveRawVsockChroot(ctx, ec1init.VsockPort)
		if err != nil {
			slog.ErrorContext(ctx, "problem serving vsock", "error", err)
			os.Exit(1)
		}
		// } else {
		// 	ctx = slogctx.With(ctx, slog.String("mode", "switch_root"))
		// 	err := handOffToContainer(ctx)
		// 	if err != nil {
		// 		slog.ErrorContext(ctx, "problem initializing initramfs", "error", err)
		// 		os.Exit(1)
		// 	}
		// }

	} else if len(os.Args) > 1 && os.Args[1] == "vsock" {
		ctx = slogctx.With(ctx, slog.String("mode", "secondary_vsock"))

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

func serveRawVsockChroot(ctx context.Context, port int) error {

	manifest, err := loadManifest(ctx)
	if err != nil {
		return errors.Errorf("loading manifest: %w", err)
	}

	go func() {
		err := bindMountsToChroot(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "binding mounts to chroot", "error", err)
			os.Exit(1)
		}
	}()

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutorWithCommandCreationFunc(1024, func(ctx context.Context, command string) *exec.Cmd {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return nil
		}

		entrypointString := strings.Join(manifest.Config.Entrypoint, " ")
		fullCommand := strings.Join(parts, " ")

		if !strings.HasPrefix(fullCommand, entrypointString) {
			fullCommand = entrypointString + " " + fullCommand
		}

		// Use busybox sh -c to execute the command in the chrooted environment
		// This ensures PATH resolution happens within the chroot
		cmd := exec.CommandContext(ctx, "/bin/busybox", "sh", "-c", fullCommand)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Chroot: ec1init.NewRootAbsPath,
		}
		cmd.Dir = "/" // without this, we end up with a stderr sh: 0: getcwd() failed: No such file or directory

		// Set the environment from the container manifest
		env := manifest.Config.Env
		// Ensure PATH is set correctly for the container
		env = append(env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
		cmd.Env = env

		slog.DebugContext(ctx, "executing chrooted command", "command", fullCommand, "env", env)

		return cmd
	})
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}

func serveRawVsock(ctx context.Context, port int) error {

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutor(1024)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}

func triggerSecondaryVsock(ctx context.Context) error {
	pid, _, err := syscall.StartProcess(os.Args[0], append([]string{"vsock"}, os.Args[1:]...), &syscall.ProcAttr{
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

func mountInitramfs(ctx context.Context) error {
	slog.InfoContext(ctx, "initramfs init started, mounting rootfs")

	mounts := [][]string{
		{"mkdir", "-p", "/proc", "/sys", "/dev", ec1init.NewRootAbsPath, "/run", ec1init.Ec1AbsPath, "/mnt/lower", "/mnt/upper", "/mnt/overlay", "/mnt/wrk"},
		{"mount", "-t", "devtmpfs", "devtmpfs", "/dev"},
		{"mount", "-t", "proc", "proc", "/proc"},
		{"mount", "-t", "sysfs", "sysfs", "/sys"},
		{"mkdir", "-p", "/dev/pts"},
		{"mount", "-t", "devpts", "devpts", "/dev/pts"},
		{"mount", "-t", "virtiofs", ec1init.Ec1VirtioTag, ec1init.Ec1AbsPath},

		{"mount", "-t", "virtiofs", "-o", "ro", ec1init.RootfsVirtioTag, "/mnt/lower"},
		// read only lower layer
		// {"mount", "-t", "ext4", "-o", "ro", "/dev/vdb", "/mnt/lower"},
		// {"ls", "-la", "/bin/mke2fs"},
		// {"/bin/file", "/bin/mke2fs"},
		// mkfs.ext4 -L upper -O has_journal /dev/vda
		// {"truncate", "-s", "128M", ec1init.Ec1AbsPath + "upper.img"},
		// {"/bin/mke2fs", "-t", "ext4", "-L", "upper", "-O", "has_journal", ec1init.Ec1AbsPath + "upper.img"},
		// writable upper layer

		{"mount", "-t", "tmpfs", "tmpfs", "/mnt/upper"},
		{"mkdir", "-p", "/mnt/upper/upper", "/mnt/upper/work"},
		{"mount", "-t", "overlay", "overlay", "-o", "lowerdir=/mnt/lower,upperdir=/mnt/upper/upper,workdir=/mnt/upper/work", ec1init.NewRootAbsPath},
	}

	for _, mount := range mounts {
		err := execCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	return nil
}

func bindMountsToChroot(ctx context.Context) error {

	mounts := [][]string{
		{"mkdir", "-p", ec1init.NewRootAbsPath + "/dev", ec1init.NewRootAbsPath + "/proc", ec1init.NewRootAbsPath + "/sys", ec1init.NewRootAbsPath + "/run", ec1init.NewRootAbsPath + ec1init.Ec1AbsPath},
		{"mount", "--bind", "/dev", ec1init.NewRootAbsPath + "/dev"},
		{"mount", "--bind", "/proc", ec1init.NewRootAbsPath + "/proc"},
		{"mount", "--bind", "/sys", ec1init.NewRootAbsPath + "/sys"},
		{"mount", "--bind", "/run", ec1init.NewRootAbsPath + "/run"},
		{"mount", "--bind", "/ec1", ec1init.NewRootAbsPath + ec1init.Ec1AbsPath},
	}

	for _, mount := range mounts {
		err := execCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	return nil
}

func handOffToContainer(ctx context.Context) error {

	slog.InfoContext(ctx, "triggering vsock")

	err := triggerSecondaryVsock(ctx)
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
	initCmdLineArgs, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerCmdlineFile))
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

	return &image, nil
}

func execCmdForwardingStdio(ctx context.Context, cmds ...string) error {
	if len(cmds) == 0 {
		return errors.Errorf("no command to execute")
	}

	argc := "/bin/busybox"
	if strings.HasPrefix(cmds[0], "/") {
		argc = cmds[0]
		cmds = cmds[1:]
	}
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
