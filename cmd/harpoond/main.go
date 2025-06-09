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
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd/v2/pkg/oci"
	"github.com/containerd/ttrpc"
	"github.com/mdlayher/vsock"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gitlab.com/tozd/go/errors"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	slogctx "github.com/veqryn/slog-context"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
	"github.com/walteh/ec1/pkg/ec1init"
	"github.com/walteh/ec1/pkg/harpoon"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

type mode string

const (
	modeRootfs   mode = "rootfs"
	modeOCI      mode = "oci"
	modeManifest mode = "manifest"
)

func main() {

	pid := os.Getpid()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logging.SetupSlogSimpleToWriterWithProcessName(ctx, os.Stdout, true, "harpoond")

	ctx = slogctx.Append(ctx, slog.Int("pid", pid))

	if _, err := os.Stat(ec1init.Ec1AbsPath); err == nil {
		err := runContainerd(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "problem running containerd", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if _, err := os.Stat(ec1init.Ec1AbsPath); os.IsNotExist(err) {
		os.MkdirAll(ec1init.Ec1AbsPath, 0755)
	}

	// mount the ec1 virtiofs
	err := execCmdForwardingStdio(ctx, "mount", "-t", "virtiofs", ec1init.Ec1VirtioTag, ec1init.Ec1AbsPath)
	if err != nil {
		slog.ErrorContext(ctx, "problem mounting ec1 virtiofs", "error", err)
		os.Exit(1)
	}

	spec, manifest, bindMounts, err := loadSpecOrManifest(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "problem loading spec or manifest", "error", err)
		os.Exit(1)
	}

	// timesyncFile := filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerTimesyncFile)
	// dat, err := os.ReadFile(timesyncFile)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "problem reading timesync file", "error", err)
	// 	os.Exit(1)
	// }

	// parts := strings.Split(string(dat), ":")

	// timesync, err := strconv.Atoi(parts[0])
	// if err != nil {
	// 	slog.ErrorContext(ctx, "problem parsing timesync file", "error", err)
	// 	os.Exit(1)
	// }

	// zoneoffset, err := strconv.Atoi(parts[1])
	// if err != nil {
	// 	slog.ErrorContext(ctx, "problem parsing timesync file", "error", err)
	// 	os.Exit(1)
	// }

	// tv := unix.NsecToTimeval(int64(timesync)) // helper to build Timeval

	// if err := unix.Settimeofday(&tv); err != nil {
	// 	slog.ErrorContext(ctx, "Settimeofday failed", "error", err)
	// }

	// slog.InfoContext(ctx, "setting time", "time", tv)

	if spec != nil {
		ctx = slogctx.Append(ctx, slog.String("mode", string(modeOCI)))

		if bindMounts == nil {
			slog.ErrorContext(ctx, "no bind mounts found")
			os.Exit(1)
		}

		err = mountRootfs(ctx, spec, bindMounts)
		if err != nil {
			slog.ErrorContext(ctx, "problem mounting rootfs", "error", err)
			os.Exit(1)
		}

		err = switchRoot(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "problem switching root", "error", err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	if manifest != nil {
		ctx = slogctx.Append(ctx, slog.String("mode", string(modeManifest)))
		err = runManifest(ctx, ec1init.VsockPort, manifest.Config.Entrypoint, manifest.Config.Env)
		if err != nil {
			slog.ErrorContext(ctx, "problem serving vsock", "error", err)
			os.Exit(1)
		}
	}

	slog.ErrorContext(ctx, "no spec or manifest found")
	os.Exit(1)

}

func runContainerd(ctx context.Context) error {
	ctx = slogctx.Append(ctx, slog.String("mode", string(modeRootfs)))
	slog.InfoContext(ctx, "running in rootfs, gonna just wait to be killed")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	go func() {
		err := runTtrpc(ctx)
		if err != nil {
			slog.ErrorContext(ctx, "problem serving ttrpc", "error", err)
		}
	}()

	go func() {
		for tick := range ticker.C {
			slog.InfoContext(ctx, "still running in rootfs, waiting to be killed", "tick", tick)
		}
	}()

	select {}
}

func runTtrpc(ctx context.Context) error {
	ttrpcServe, err := ttrpc.NewServer()
	if err != nil {
		return errors.Errorf("creating ttrpc server: %w", err)
	}

	harpoonv1.RegisterTTRPCGuestServiceService(ttrpcServe, harpoon.NewAgentService())

	listener, err := vsock.ListenContextID(3, uint32(ec1init.VsockPort), nil)
	if err != nil {
		return errors.Errorf("dialing vsock: %w", err)
	}

	return ttrpcServe.Serve(ctx, listener)
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

// func serveRawVsock(ctx context.Context, port int) error {

// 	tranport := transport.NewVSockTransport(0, uint32(port))
// 	executor := executor.NewStreamingExecutor(1024)
// 	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
// 		return protocol.NewFramedProtocol(conn)
// 	})

// 	return server.Serve()
// }

// func triggerSecondaryVsock(ctx context.Context) error {
// 	pid, _, err := syscall.StartProcess(os.Args[0], append([]string{"vsock"}, os.Args[1:]...), &syscall.ProcAttr{
// 		Env:   os.Environ(),
// 		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
// 	})
// 	if err != nil {
// 		return errors.Errorf("starting process: %w", err)
// 	}

// 	slog.InfoContext(ctx, "started vsock")

// 	// // make a pid file, stdout and
// 	pidFile, err := os.Create(filepath.Join(ec1init.Ec1AbsPath, ec1init.VsockPidFile))
// 	if err != nil {
// 		return errors.Errorf("creating pid file: %w", err)
// 	}

// 	pidFile.WriteString(strconv.Itoa(pid))
// 	pidFile.Close()

// 	return nil
// }

// 2025-06-06 11:37 49.8894 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"proc","Source":"proc","Target":"/proc","Options":["nosuid","noexec","nodev"]}
// 2025-06-06 11:37 49.8894 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"tmpfs","Source":"tmpfs","Target":"/dev","Options":["nosuid","strictatime","mode=755","size=65536k"]}
// 2025-06-06 11:37 49.8894 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"sysfs","Source":"sysfs","Target":"/sys","Options":["nosuid","noexec","nodev","ro"]}
// 2025-06-06 11:37 49.8894 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"tmpfs","Source":"tmpfs","Target":"/run","Options":["nosuid","strictatime","mode=755","size=65536k"]}
// 2025-06-06 11:37 49.8895 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"devpts","Source":"devpts","Target":"/dev/pts","Options":["nosuid","noexec","newinstance","ptmxmode=0666","mode=0620","gid=5"]}
// 2025-06-06 11:37 49.8895 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"tmpfs","Source":"shm","Target":"/dev/shm","Options":["nosuid","noexec","nodev","mode=1777","size=65536k"]}
// 2025-06-06 11:37 49.8895 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"mqueue","Source":"mqueue","Target":"/dev/mqueue","Options":["nosuid","noexec","nodev"]}
// 2025-06-06 11:37 49.8895 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"bind","Source":"/var/lib/nerdctl/e0ce5476/containers/harpoon/8762c7401028e46f6692bf50ead961cabaaebb8c571c35a23ed8c8179eaf950d/resolv.conf","Target":"/etc/resolv.conf","Options":["bind",""]}
// 2025-06-06 11:37 49.8895 WRN [shim] containerd/service.go:280 [logrus] skipping mount: {"Type":"bind","Source":"/var/lib/nerdctl/e0ce5476/etchosts/harpoon/8762c7401028e46f6692bf50ead961cabaaebb8c571c35a23ed8c8179eaf950d/hosts","Target":"/etc/hosts","Options":["bind",""]}
func mountRootfs(ctx context.Context, spec *oci.Spec, customMounts []specs.Mount) error {
	// dirs := []string{}
	cmds := [][]string{}

	// mkdir and mount the rootfs
	if err := os.MkdirAll(ec1init.NewRootAbsPath, 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	if err := execCmdForwardingStdio(ctx, "mount", "-t", "virtiofs", ec1init.RootfsVirtioTag, ec1init.NewRootAbsPath); err != nil {
		return errors.Errorf("mounting rootfs: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath), 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	if err := execCmdForwardingStdio(ctx, "mount", "--move", ec1init.Ec1AbsPath, filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath)); err != nil {
		return errors.Errorf("mounting ec1: %w", err)
	}

	cmds = append(cmds, []string{"rm", "-rf", "/newroot/etc/hosts"})
	cmds = append(cmds, []string{"rm", "-rf", "/newroot/etc/resolv.conf"})

	if err := os.MkdirAll(filepath.Join(ec1init.NewRootAbsPath, "etc"), 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	// dirs = append(dirs, filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath))

	// trying to figure out how to proerly do this to not skip things
	for _, mount := range append(spec.Mounts, customMounts...) {
		dest := filepath.Join(ec1init.NewRootAbsPath, mount.Destination)
		if mount.Destination == "/etc/resolv.conf" || mount.Destination == "/etc/hosts" {
			continue
		}
		cmds = append(cmds, []string{"mkdir", "-p", dest})
		// if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		// 	return errors.Errorf("making directories: %w", err)
		// }

		if dest == "/newroot/ec1" {
			continue
		}

		opts := []string{"-o", strings.Join(mount.Options, ",")}
		if len(mount.Options) == 1 {
			opts = []string{}
		}

		switch mount.Type {
		case "ec1-virtiofs":
			allOpts := []string{"mount", "-t", "virtiofs", mount.Source}
			allOpts = append(allOpts, opts...)
			allOpts = append(allOpts, dest)
			cmds = append(cmds, allOpts)
		case "bind":
			continue
		default:
			allOpts := []string{"mount", "-t", mount.Type, mount.Source}
			allOpts = append(allOpts, opts...)
			allOpts = append(allOpts, dest)
			cmds = append(cmds, allOpts)
		}
	}

	// // mkdir command
	// err := execCmdForwardingStdio(ctx, "mkdir", "-p", strings.Join(dirs, " "))
	// if err != nil {
	// 	return errors.Errorf("making directories: %w", err)
	// }

	for _, cmd := range cmds {
		err := execCmdForwardingStdio(ctx, cmd...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", cmd, err)
		}
	}

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

// func handOffToContainer(ctx context.Context) error {

// 	slog.InfoContext(ctx, "triggering vsock")

// 	err := triggerSecondaryVsock(ctx)
// 	if err != nil {
// 		return errors.Errorf("triggering vsock: %w", err)
// 	}

// 	slog.InfoContext(ctx, "loading manifest")

// 	manifest, err := loadManifest(ctx)
// 	if err != nil {
// 		return errors.Errorf("loading manifest: %w", err)
// 	}

// 	slog.InfoContext(ctx, "loading init cmd line args")

// 	// switch_root to the new rootfs, calling the entrypoint

// 	cmd, err := loadInitCmdLineArgs(ctx)
// 	if err != nil {
// 		return errors.Errorf("loading init cmd line args: %w", err)
// 	}

// 	moveMounts := [][]string{
// 		{"mount", "-o", "move", "/dev", ec1init.NewRootAbsPath + "/dev"},
// 		{"mount", "-o", "move", "/proc", ec1init.NewRootAbsPath + "/proc"},
// 		{"mount", "-o", "move", "/sys", ec1init.NewRootAbsPath + "/sys"},
// 		{"mount", "-o", "move", "/run", ec1init.NewRootAbsPath + "/run"},
// 		{"mount", "-o", "move", "/ec1", ec1init.NewRootAbsPath + ec1init.Ec1AbsPath},
// 	}

// 	for _, mount := range moveMounts {
// 		err := execCmdForwardingStdio(ctx, mount...)
// 		if err != nil {
// 			return errors.Errorf("running command: %v: %w", mount, err)
// 		}
// 	}

// 	err = switchRoot(ctx, manifest, cmd)
// 	if err != nil {
// 		return errors.Errorf("switching root: %w", err)
// 	}

// 	panic("unreachable, we should have switched root")
// }

// func loadInitCmdLineArgs(ctx context.Context) ([]string, error) {
// 	initCmdLineArgs, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerCmdlineFile))
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			slog.WarnContext(ctx, "user provided cmdline not found, ignoring")
// 			return []string{}, nil
// 		}
// 		return nil, errors.Errorf("loading user provided cmdline: %w", err)
// 	}

// 	var args []string
// 	err = json.Unmarshal(initCmdLineArgs, &args)
// 	if err != nil {
// 		return nil, errors.Errorf("unmarshalling user provided cmdline: %w", err)
// 	}

// 	return args, nil
// }

func switchRoot(ctx context.Context) error {

	if err := execCmdForwardingStdio(ctx, "touch", "/newroot/harpoond"); err != nil {
		return errors.Errorf("touching harpoond: %w", err)
	}

	// rename ourself to new root
	if err := execCmdForwardingStdio(ctx, "mount", "--bind", os.Args[0], "/newroot/harpoond"); err != nil {
		return errors.Errorf("renaming self: %w", err)
	}

	entrypoint := []string{"/harpoond"}

	env := []string{}
	env = append(env, "PATH=/usr/sbin:/usr/bin:/sbin:/bin")

	argc := "/bin/busybox"
	argv := append([]string{"switch_root", ec1init.NewRootAbsPath}, entrypoint...)

	slog.InfoContext(ctx, "switching root - godspeed little process", "rootfs", ec1init.NewRootAbsPath, "argv", argv)

	if err := syscall.Exec(argc, argv, env); err != nil {
		return errors.Errorf("Failed to exec %v %v: %v", entrypoint, argv, err)
	}

	panic("unreachable, we hand off to the entrypoint")

}

func loadSpecOrManifest(ctx context.Context) (spec *oci.Spec, manifest *v1.Image, bindMounts []specs.Mount, err error) {
	specd, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerSpecFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, nil, errors.Errorf("reading spec: %w", err)
		}
	} else {
		err = json.Unmarshal(specd, &spec)
		if err != nil {
			return nil, nil, nil, errors.Errorf("unmarshalling spec: %w", err)
		}
	}

	manifestd, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerManifestFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, nil, errors.Errorf("reading manifest: %w", err)
		}
	} else {
		var manifest v1.Image
		err = json.Unmarshal(manifestd, &manifest)
		if err != nil {
			return nil, nil, nil, errors.Errorf("unmarshalling manifest: %w", err)
		}
	}

	bindMountsBytes, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerMountsFile))
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, nil, nil, errors.Errorf("reading bind mounts: %w", err)
		}
	} else {
		err = json.Unmarshal(bindMountsBytes, &bindMounts)
		if err != nil {
			return nil, nil, nil, errors.Errorf("unmarshalling bind mounts: %w", err)
		}
	}

	return spec, manifest, bindMounts, nil
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
