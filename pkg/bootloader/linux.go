package bootloader

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"gitlab.com/tozd/go/errors"

	nbd "libguestfs.org/libnbd"
)

// NewLinuxBootloader creates a new bootloader to start a VM with the file at
// vmlinuzPath as the kernel, kernelCmdLine as the kernel command line, and the
// file at initrdPath as the initrd. On ARM64, the kernel must be uncompressed
// otherwise the VM will fail to boot.
func NewLinuxBootloader(vmlinuzPath, kernelCmdLine, initrdPath string) *LinuxBootloader {
	return &LinuxBootloader{
		VmlinuzPath:   vmlinuzPath,
		KernelCmdLine: kernelCmdLine,
		InitrdPath:    initrdPath,
	}
}

func PrepareInitrd(ctx context.Context, file string) error {
	// install the kata container agent and add it to the  initrd located at the file path
	nbd, err := nbd.Create()
	if err != nil {
		return errors.Errorf("creating nbd client: %w", err)
	}
	defer nbd.Close()

	// Use the correct API for connecting to NBD
	if err := nbd.ConnectUri("nbd://127.0.0.1:10809"); err != nil {
		return errors.Errorf("connecting to nbd server: %w", err)
	}
	defer nbd.Shutdown(nil)

	// Create a temporary mount point
	mountDir, err := os.MkdirTemp("", "initrd-mount-*")
	if err != nil {
		return errors.Errorf("creating temp mount directory: %w", err)
	}
	defer os.RemoveAll(mountDir)

	// Mount the initrd
	if err := mountInitrd(file, mountDir); err != nil {
		return errors.Errorf("mounting initrd: %w", err)
	}
	defer unmountInitrd(mountDir)

	// Install Kata agent
	if err := installKataAgent(ctx, mountDir); err != nil {
		return errors.Errorf("installing kata agent: %w", err)
	}

	// Update the initrd file
	if err := updateInitrd(mountDir, file); err != nil {
		return errors.Errorf("updating initrd: %w", err)
	}

	return nil
}

// mountInitrd mounts the initrd file to the specified directory
func mountInitrd(initrdPath, mountDir string) error {
	cmd := exec.Command("mount", "-o", "loop", initrdPath, mountDir)
	return cmd.Run()
}

// unmountInitrd unmounts the initrd
func unmountInitrd(mountDir string) error {
	cmd := exec.Command("umount", mountDir)
	return cmd.Run()
}

// installKataAgent installs the Kata Containers agent into the initrd
func installKataAgent(ctx context.Context, mountDir string) error {
	// Create the bin directory if it doesn't exist
	binDir := filepath.Join(mountDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return errors.Errorf("creating bin directory: %w", err)
	}

	// Download or copy the Kata agent binary
	// TODO: Replace with actual agent binary source and installation logic
	kataAgentPath := filepath.Join(binDir, "kata-agent")

	// Example: Download the agent binary
	// cmd := exec.CommandContext(ctx, "curl", "-L", "https://github.com/kata-containers/agent/releases/download/v1.0.0/kata-agent", "-o", kataAgentPath)
	// if err := cmd.Run(); err != nil {
	//     return errors.Errorf("downloading kata agent: %w", err)
	// }

	// For now, create a placeholder file
	if err := os.WriteFile(kataAgentPath, []byte("#!/bin/sh\necho 'Kata agent placeholder'\n"), 0755); err != nil {
		return errors.Errorf("creating kata agent placeholder: %w", err)
	}

	// Set up init script to start the agent
	initScript := filepath.Join(mountDir, "init")
	initContent := `#!/bin/sh
echo "Starting Kata Containers agent..."
/bin/kata-agent &
exec /bin/sh
`
	if err := os.WriteFile(initScript, []byte(initContent), 0755); err != nil {
		return errors.Errorf("creating init script: %w", err)
	}

	return nil
}

// updateInitrd updates the initrd file with the contents from the mount directory
func updateInitrd(mountDir, initrdPath string) error {
	// Create a temporary file for the new initrd
	tempInitrd, err := os.CreateTemp("", "new-initrd-*")
	if err != nil {
		return errors.Errorf("creating temp initrd: %w", err)
	}
	defer os.Remove(tempInitrd.Name())
	tempInitrd.Close()

	// Create a new initrd from the mount directory
	cmd := exec.Command("cd", mountDir, "&&", "find", ".", "-print0", "|", "cpio", "--null", "-H", "newc", "-o", "|", "gzip", "-9", ">", tempInitrd.Name())
	if err := cmd.Run(); err != nil {
		return errors.Errorf("creating new initrd: %w", err)
	}

	// Replace the original initrd with the new one
	if err := os.Rename(tempInitrd.Name(), initrdPath); err != nil {
		return errors.Errorf("replacing initrd: %w", err)
	}

	return nil
}
