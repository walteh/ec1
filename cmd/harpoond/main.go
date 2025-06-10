package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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
)

type mode string

const (
	modeRootfs   mode = "rootfs"
	modeOCI      mode = "oci"
	modeManifest mode = "manifest"
)

var binariesToCopy = []string{
	"/hbin/lshw",
	// "/hbin/mount",
	// "/hbin/umount",
	// "/hbin/lsblk",
	// "/hbin/fdisk",
	// "/hbin/findmnt",
	// "/hbin/losetup",
	// "/hbin/mkswap",
	// "/hbin/swapon",
	// "/hbin/swapoff",
	// "/hbin/dmesg",
	// "/hbin/fsck",
	// "/hbin/blkid",
	// "/hbin/blockdev",
	// "/hbin/fstrim",
	// "/hbin/lscpu",
	// "/hbin/lsmem",
	// "/hbin/hwclock",
	// "/hbin/sfdisk",
}

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
	err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "-t", "virtiofs", ec1init.Ec1VirtioTag, ec1init.Ec1AbsPath)
	if err != nil {
		slog.ErrorContext(ctx, "problem mounting ec1 virtiofs", "error", err)
		os.Exit(1)
	}

	spec, manifest, bindMounts, err := loadSpecOrManifest(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "problem loading spec or manifest", "error", err)
		os.Exit(1)
	}

	if spec != nil {
		ctx = slogctx.Append(ctx, slog.String("mode", string(modeOCI)))

		if bindMounts == nil {
			slog.ErrorContext(ctx, "no bind mounts found")
			os.Exit(1)
		}

		if err := mountRootfsSecondary(ctx, ec1init.NewRootAbsPath, bindMounts); err != nil {
			slog.ErrorContext(ctx, "problem mounting rootfs secondary", "error", err)
			os.Exit(1)
		}

		err = mountRootfsPrimary(ctx)
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

func logFile(ctx context.Context, path string) {
	fmt.Println()
	fmt.Println("---------------" + path + "-----------------")
	_ = harpoon.ExecCmdForwardingStdio(ctx, "ls", "-lah", path)
	_ = harpoon.ExecCmdForwardingStdio(ctx, "cat", path)

}

func logCommand(ctx context.Context, cmd string) {
	fmt.Println()
	fmt.Println("---------------" + cmd + "-----------------")
	_ = harpoon.ExecCmdForwardingStdio(ctx, "sh", "-c", cmd)
}

func logDirContents(ctx context.Context, path string) {
	fmt.Println()
	fmt.Println("---------------" + path + "-----------------")
	_ = harpoon.ExecCmdForwardingStdio(ctx, "ls", "-lah", path)
}

func runContainerd(ctx context.Context) error {
	ctx = slogctx.Append(ctx, slog.String("mode", string(modeRootfs)))
	slog.InfoContext(ctx, "running in rootfs, gonna just wait to be killed")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// spec, _, _, err := loadSpecOrManifest(ctx)
	// if err != nil {
	// 	slog.ErrorContext(ctx, "problem loading spec or manifest", "error", err)
	// 	return errors.Errorf("loading spec or manifest: %w", err)
	// }

	// // if spec == nil {
	// // 	return errors.Errorf("no spec found")
	// // }

	// logFile(ctx, "/proc/self/mountinfo")

	// logDirContents(ctx, "/dev")
	// logDirContents(ctx, "/dev/pts")
	// logDirContents(ctx, "/proc/self/fd")
	// logDirContents(ctx, "/sys/class/virtio-ports")
	// // logDirContents(ctx, "/dev/vport")
	// logDirContents(ctx, "/dev/hvc")
	// logDirContents(ctx, "/dev/tty")

	// logDirContents(ctx, "/dev/vport2p0")
	// logDirContents(ctx, "/dev/vport3p0")
	// logCommand(ctx, "dmesg")
	// logCommand(ctx, "ls -lah /hbin")
	// logCommand(ctx, "lsmod | grep virtio")
	// // logCommand(ctx, "dmesg | grep virtio")
	// logCommand(ctx, "/hbin/lshw")

	// // logDirContents(ctx, "/proc/self/fd")
	// logDirContents(ctx, "/sys/class/virtio-ports/vport2p0")
	// logFile(ctx, "/sys/class/virtio-ports/vport2p0/dev")
	// logFile(ctx, "/sys/class/virtio-ports/vport2p0/uevent")
	// logDirContents(ctx, "/sys/class/virtio-ports/vport2p0/subsystem")
	// logDirContents(ctx, "/sys/class/virtio-ports/vport2p0/device")
	// logDirContents(ctx, "/sys/class/virtio-ports/vport2p0")

	// logCommand(ctx, "ls -la /sys/bus/virtio/devices/virtio*/driver")

	// fmt.Println("--------------------------------")

	// debug1()
	// debug2()
	// debug3()
	// fmt.Println("--------------------------------")
	// debug4(ctx)

	// fmt.Println("--------------------------------")

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
	ttrpcServe, err := ttrpc.NewServer(ttrpc.WithServerDebugging())
	if err != nil {
		return errors.Errorf("creating ttrpc server: %w", err)
	}

	harpoonv1.RegisterTTRPCGuestServiceService(ttrpcServe, harpoon.NewAgentService().WrapWithErrorLogging())

	listener, err := vsock.ListenContextID(3, uint32(ec1init.VsockPort), nil)
	if err != nil {
		return errors.Errorf("dialing vsock: %w", err)
	}

	return ttrpcServe.Serve(ctx, listener)
}

// func getCopyMountCommands(ctx context.Context) ([][]string, error) {
// 	cmds := [][]string{}

// 	mountsBytes, err := os.ReadFile(filepath.Join(ec1init.Ec1AbsPath, ec1init.ContainerMountsFile))
// 	if err != nil {
// 		return nil, errors.Errorf("loading mounts: %w", err)
// 	}

// 	var mounts []specs.Mount
// 	err = json.Unmarshal(mountsBytes, &mounts)
// 	if err != nil {
// 		return nil, errors.Errorf("unmarshalling mounts: %w", err)
// 	}

// 	for _, mount := range mounts {
// 		if mount.Type != "copy" {
// 			continue
// 		}

// 		cmds = append(cmds, []string{"mkdir", "-p", filepath.Join(ec1init.NewRootAbsPath, filepath.Dir(mount.Destination))})
// 		cmds = append(cmds, []string{"touch", filepath.Join(ec1init.NewRootAbsPath, mount.Destination)})
// 		cmds = append(cmds, []string{"mount", "--bind", mount.Destination, filepath.Join(ec1init.NewRootAbsPath, mount.Destination)})
// 	}

//		return cmds, nil
//	}
func mountRootfsPrimary(ctx context.Context) error {

	// mkdir and mount the rootfs
	// if err := os.MkdirAll(ec1init.NewRootAbsPath, 0755); err != nil {
	// 	return errors.Errorf("making directories: %w", err)
	// }

	// if err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "-t", "virtiofs", ec1init.RootfsVirtioTag, ec1init.NewRootAbsPath); err != nil {
	// 	return errors.Errorf("mounting rootfs: %w", err)
	// }

	_ = harpoon.ExecCmdForwardingStdio(ctx, "ls", "-lah", "/newroot")

	if err := os.MkdirAll(filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath), 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	if err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "--move", ec1init.Ec1AbsPath, filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath)); err != nil {
		return errors.Errorf("mounting ec1: %w", err)
	}

	cmds := [][]string{}

	// copyMounts, err := getCopyMountCommands(ctx)
	// if err != nil {
	// 	return errors.Errorf("getting copy mounts: %w", err)
	// }

	// cmds = append(cmds, copyMounts...)

	for _, binary := range binariesToCopy {
		cmds = append(cmds, []string{"mkdir", "-p", filepath.Join(ec1init.NewRootAbsPath, filepath.Dir(binary))})
		cmds = append(cmds, []string{"touch", filepath.Join(ec1init.NewRootAbsPath, binary)})
		cmds = append(cmds, []string{"mount", "--bind", binary, filepath.Join(ec1init.NewRootAbsPath, binary)})
	}

	for _, cmd := range cmds {
		err := harpoon.ExecCmdForwardingStdio(ctx, cmd...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", cmd, err)
		}
	}

	return nil
}

func mountRootfsSecondary(ctx context.Context, prefix string, customMounts []specs.Mount) error {
	// dirs := []string{}
	cmds := [][]string{}

	// cmds = append(cmds, []string{"rm", "-rf", prefix + "/etc/hosts"})
	// cmds = append(cmds, []string{"rm", "-rf", prefix + "/etc/resolv.conf"})

	// if err := os.MkdirAll(filepath.Join(prefix, "etc"), 0755); err != nil {
	// 	return errors.Errorf("making directories: %w", err)
	// }

	cmds = append(cmds, []string{"mkdir", "-p", prefix + "/dev/pts"})
	cmds = append(cmds, []string{"mount", "-t", "devpts", "devpts", prefix + "/dev/pts", "-o", "gid=5,mode=620,ptmxmode=666"})

	// dirs = append(dirs, filepath.Join(prefix, ec1init.Ec1AbsPath))

	// trying to figure out how to proerly do this to not skip things
	for _, mount := range customMounts {

		dest := filepath.Join(prefix, mount.Destination)
		// if mount.Destination == "/etc/resolv.conf" || mount.Destination == "/etc/hosts" {
		// 	continue
		// }
		// if mount.Type != "ec1-virtiofs" {
		// 	if mount.Type == "bind" || slices.Contains(mount.Options, "rbind") {
		// 		continue
		// 	}
		// }
		cmds = append(cmds, []string{"mkdir", "-p", dest})
		// if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		// 	return errors.Errorf("making directories: %w", err)
		// }

		if dest == prefix+"/ec1" {
			continue
		}

		opd := strings.Join(mount.Options, ",")
		opd = strings.TrimSuffix(opd, ",")

		opts := []string{"-o", opd}
		if len(mount.Options) == 1 {
			opts = []string{}
		}

		// if mount.Destination == "/dev" {
		// 	mount.Type = "devtmpfs"
		// 	mount.Source = "devtmpfs"
		// }

		switch mount.Type {

		case "bind", "copy":
			continue
		default:
			allOpts := []string{"mount", "-t", mount.Type, mount.Source}
			allOpts = append(allOpts, opts...)
			allOpts = append(allOpts, dest)
			cmds = append(cmds, allOpts)
		}
	}

	for _, cmd := range cmds {
		err := harpoon.ExecCmdForwardingStdio(ctx, cmd...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", cmd, err)
		}
	}

	harpoon.ExecCmdForwardingStdio(ctx, "ls", "-lah", "/app/scripts")

	return nil
}

func mountRootfs(ctx context.Context, spec *oci.Spec, customMounts []specs.Mount) error {
	// dirs := []string{}
	cmds := [][]string{}

	// mkdir and mount the rootfs
	if err := os.MkdirAll(ec1init.NewRootAbsPath, 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	if err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "-t", "virtiofs", ec1init.RootfsVirtioTag, ec1init.NewRootAbsPath); err != nil {
		return errors.Errorf("mounting rootfs: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath), 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	if err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "--move", ec1init.Ec1AbsPath, filepath.Join(ec1init.NewRootAbsPath, ec1init.Ec1AbsPath)); err != nil {
		return errors.Errorf("mounting ec1: %w", err)
	}

	cmds = append(cmds, []string{"rm", "-rf", "/newroot/etc/hosts"})
	cmds = append(cmds, []string{"rm", "-rf", "/newroot/etc/resolv.conf"})

	if err := os.MkdirAll(filepath.Join(ec1init.NewRootAbsPath, "etc"), 0755); err != nil {
		return errors.Errorf("making directories: %w", err)
	}

	cmds = append(cmds, []string{"mkdir", "-p", "/newroot/dev/pts"})
	cmds = append(cmds, []string{"mount", "-t", "devpts", "devpts", "/newroot/dev/pts", "-o", "gid=5,mode=620,ptmxmode=666"})

	for _, binary := range binariesToCopy {
		cmds = append(cmds, []string{"mkdir", "-p", filepath.Join(ec1init.NewRootAbsPath, filepath.Dir(binary))})
		cmds = append(cmds, []string{"touch", filepath.Join(ec1init.NewRootAbsPath, binary)})
		cmds = append(cmds, []string{"mount", "--bind", binary, filepath.Join(ec1init.NewRootAbsPath, binary)})
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

		if mount.Destination == "/dev" {
			mount.Type = "devtmpfs"
			mount.Source = "devtmpfs"
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

	// cmds = append(cmds, []string{"mkdir", "-p", "/newroot/dev/fd"})
	// cmds = append(cmds, []string{"mount", "-t", "none", "-o", "bind", "/proc/self/fd", "/newroot/dev/fd"})

	for _, cmd := range cmds {
		err := harpoon.ExecCmdForwardingStdio(ctx, cmd...)
		if err != nil {
			return errors.Errorf("running command: %v: %w", cmd, err)
		}
	}

	return nil
}

func switchRoot(ctx context.Context) error {

	if err := harpoon.ExecCmdForwardingStdio(ctx, "touch", "/newroot/harpoond"); err != nil {
		return errors.Errorf("touching harpoond: %w", err)
	}

	// bind hbin
	if err := harpoon.ExecCmdForwardingStdio(ctx, "ls", "-lah", "/newroot/hbin"); err != nil {
		return errors.Errorf("binding hbin: %w", err)
	}

	// rename ourself to new root
	if err := harpoon.ExecCmdForwardingStdio(ctx, "mount", "--bind", os.Args[0], "/newroot/harpoond"); err != nil {
		return errors.Errorf("renaming self: %w", err)
	}

	entrypoint := []string{"/harpoond"}

	env := []string{}
	env = append(env, "PATH=/usr/sbin:/usr/bin:/sbin:/bin:/hbin")

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
