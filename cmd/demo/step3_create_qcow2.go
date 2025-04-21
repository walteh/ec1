package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	// "github.com/containerd/nydus-snapshotter/pkg/backend"
	iso9660 "github.com/kdomanski/iso9660"
	"golang.org/x/mod/semver"

	// "github.com/diskfs/go-diskfs/filesystem/ext4"

	"github.com/jedib0t/go-pretty/v6/progress"
)

var alpineVersion = "v3.21.2"

func buildAlpineURL(version string) string {
	minor := semver.MajorMinor(version)
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/%s/releases/cloud/nocloud_alpine-%s-aarch64-uefi-cloudinit-r0.qcow2", minor, strings.TrimPrefix(version, "v"))
}

var localImageCacheDir = "~/Developer/disk-images"

func expandTilde(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting user home directory: %w", err)
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
}

type Image struct {
	Qcow2Path            string
	CloudInitSeedISOPath string

	cleanup func()
}

func (i *Image) Cleanup() {
	if i.cleanup != nil {
		i.cleanup()
	}
}

// CreateQCOW2Image creates a qcow2 image for a VM
func CreateQCOW2Image(ctx context.Context) (*Image, error) {
	fmt.Println("Creating QCOW2 image...")

	alpineURL := buildAlpineURL(alpineVersion)
	fmt.Printf("Downloading Alpine image from %s\n", alpineURL)

	expandedPath, err := expandTilde(localImageCacheDir)
	if err != nil {
		return nil, fmt.Errorf("expanding path: %w", err)
	}

	localImage := filepath.Join(expandedPath, filepath.Base(alpineURL))

	// check Developer/disk-images
	if _, err := os.Stat(localImage); os.IsNotExist(err) {
		if err := downloadWithProgress(ctx, alpineURL, localImage); err != nil {
			return nil, fmt.Errorf("downloading Alpine image: %w", err)
		}
	}

	// Generate cloud-init ISO
	isoPath, cleanup, err := injectPureGo()
	if err != nil {
		return nil, fmt.Errorf("injecting cloud-init config: %w", err)
	}

	absPath, err := filepath.Abs(localImage)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path to qcow2: %w", err)
	}

	absIsoPath, err := filepath.Abs(isoPath)
	if err != nil {
		return nil, fmt.Errorf("getting absolute path to iso: %w", err)
	}

	return &Image{
		Qcow2Path:            absPath,
		CloudInitSeedISOPath: absIsoPath,
		cleanup:              cleanup,
	}, nil
}

// downloadWithProgress fetches a URL to a file path, showing a progress bar.
func downloadWithProgress(ctx context.Context, url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Set up progress writer
	pw := progress.NewWriter()
	pw.SetAutoStop(true)
	pw.SetOutputWriter(os.Stdout)
	tracker := &progress.Tracker{
		Total:   resp.ContentLength,
		Message: "Downloading",
	}
	pw.AppendTracker(tracker)
	go pw.Render()

	out, err := os.Create(dest)
	if err != nil {
		pw.Stop()
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		pw.Stop()
		return err
	}
	pw.Stop()
	return nil
}

// parseSize converts strings like "5G", "1024M" into bytes
func parseSize(s string) (int64, error) {
	units := map[string]int64{"K": 1 << 10, "M": 1 << 20, "G": 1 << 30, "T": 1 << 40}
	n := len(s)
	if n < 2 {
		return 0, fmt.Errorf("invalid size: %s", s)
	}
	unit := strings.ToUpper(s[n-1:])
	factor, ok := units[unit]
	if !ok {
		return 0, fmt.Errorf("unknown unit %q", unit)
	}
	var val int64
	if _, err := fmt.Sscan(s[:n-1], &val); err != nil {
		return 0, err
	}
	return val * factor, nil
}

var meta = `instance-id: alpine-001
local-hostname: alpine-vm`

var user = `#cloud-config
hostname: alpine-vm
password: alpine
chpasswd:
  expire: False
ssh_pwauth: True
manage_etc_hosts: true
packages:
  - qemu-img
  - qemu-system-x86_64
  - glib
  - libslirp-dev
  - libvirt
  - cloud-init
write_files:
  - path: /etc/ssh/sshd_config.d/allow_root.conf
    content: |
      PermitRootLogin yes
      PasswordAuthentication yes
  - path: /etc/apk/repositories
    content: |
      https://dl-cdn.alpinelinux.org/alpine/v3.17/main
      https://dl-cdn.alpinelinux.org/alpine/v3.17/community
runcmd:
  - apk update && apk upgrade
  - modprobe kvm_intel nested=1 || true
  - echo "cloud-init done" > /var/log/cloud-init-complete.log`

func injectPureGo() (string, func(), error) {
	// create a temp dir

	writer, err := iso9660.NewWriter()
	if err != nil {
		return "", nil, err
	}
	defer writer.Cleanup()

	for f, content := range map[string]string{"meta-data": meta, "user-data": user} {
		if err = writer.AddFile(strings.NewReader(content), f); err != nil {
			return "", nil, err
		}
	}

	fle, err := os.CreateTemp("", "cloud-init.iso")
	if err != nil {
		return "", nil, err
	}
	defer fle.Close()

	if err := writer.WriteTo(fle, "cidata"); err != nil {
		return "", nil, err
	}

	return fle.Name(), func() { os.Remove(fle.Name()) }, nil
}
