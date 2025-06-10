package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/harpoon"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

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
		{"mount", "-t", "tmpfs", "tmpfs", "/mnt/upper"},
		{"mkdir", "-p", "/mnt/upper/upper", "/mnt/upper/work"},
		{"mount", "-t", "overlay", "overlay", "-o", "lowerdir=/mnt/lower,upperdir=/mnt/upper/upper,workdir=/mnt/upper/work", ec1init.NewRootAbsPath},
	}

	for _, mount := range mounts {
		err := harpoon.ExecCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	return nil
}

func runManifest(ctx context.Context, port int, entrypoint []string, env []string) error {

	err := mountInitramfs(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "problem mounting initramfs", "error", err)
		os.Exit(1)
	}

	go func() {
		err := bindMountsToChroot(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "binding mounts to chroot", "error", err)
			os.Exit(1)
		}
	}()

	vsockFunc := func(ctx context.Context, command string) *exec.Cmd {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return nil
		}

		entrypointString := strings.Join(entrypoint, " ")
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

		// Ensure PATH is set correctly for the container
		env = append(env, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
		cmd.Env = env

		slog.DebugContext(ctx, "executing chrooted command", "command", fullCommand, "env", env)

		return cmd
	}

	err = runVsock(ctx, port, vsockFunc)
	if err != nil {
		slog.ErrorContext(ctx, "problem serving vsock", "error", err)
		return errors.Errorf("serving vsock: %w", err)
	}

	return nil
}

func runVsock(ctx context.Context, port int, f func(ctx context.Context, command string) *exec.Cmd) error {
	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutorWithCommandCreationFunc(1024, f)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	slog.InfoContext(ctx, "serving vsock", "port", port)

	err := server.Serve()
	if err != nil {
		return errors.Errorf("serving vsock: %w", err)
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
		err := harpoon.ExecCmdForwardingStdio(ctx, mount...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", mount, err)
		}
	}

	return nil
}
