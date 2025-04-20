package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem/ext4"
	"golang.org/x/mod/semver"

	// "github.com/diskfs/go-diskfs/filesystem/ext4"
	qcow2z "github.com/gpu-ninja/qcow2"
	"github.com/jedib0t/go-pretty/v6/progress"
)

var alpineVersion = "v3.21.2"

func buildAlpineURL(version string) string {
	minor := semver.MajorMinor(version)
	return fmt.Sprintf("https://dl-cdn.alpinelinux.org/alpine/%s/releases/cloud/nocloud_alpine-%s-aarch64-uefi-cloudinit-r0.qcow2", minor, strings.TrimPrefix(version, "v"))
}

var localImageCacheDir = "~/Developer/disk-images"

// CreateQCOW2Image creates a qcow2 image for a VM
func CreateQCOW2Image(ctx context.Context) (string, error) {
	fmt.Println("Creating QCOW2 image...")

	alpineURL := buildAlpineURL(alpineVersion)
	fmt.Printf("Downloading Alpine image from %s\n", alpineURL)

	localImage := filepath.Join(localImageCacheDir, filepath.Base(alpineURL))

	// check Developer/disk-images
	if _, err := os.Stat(localImage); os.IsNotExist(err) {
		if err := downloadWithProgress(ctx, alpineURL, localImage); err != nil {
			return "", fmt.Errorf("downloading Alpine image: %w", err)
		}
	}

	// Generate cloud-init ISO
	if err := injectPureGo(localImage); err != nil {
		return "", fmt.Errorf("injecting cloud-init config: %w", err)
	}

	return localImage, nil
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

func injectPureGo(qcPath string) error {
	// 1) Open QCOW2 container
	img, err := qcow2z.Open(qcPath, false)
	if err != nil {
		return fmt.Errorf("opening qcow2: %w", err)
	}
	defer img.Close()

	// 2) Extract the raw backing file (or access cluster map)
	// 3) Use go-diskfs to open the raw data as a "disk"
	disk, err := diskfs.Open("rawdata.bin", diskfs.WithOpenMode(diskfs.ReadWrite))
	partTbl, err := disk.GetPartitionTable()
	if err != nil {
		return fmt.Errorf("getting partition table: %w", err)
	}
	fs, err := disk.GetFilesystem(int(partTbl.GetPartitions()[0].GetStart()))
	if err != nil {
		return fmt.Errorf("getting filesystem: %w", err)
	}
	extFs := fs.(*ext4.FileSystem)

	// 4) Mkdir /var/lib/cloud/seed/nocloud
	if err := extFs.Mkdir("/var/lib/cloud/seed/nocloud"); err != nil {
		return fmt.Errorf("making directory: %w", err)
	}

	// 5) Copy in meta-data and user-data
	for f, data := range map[string]string{"meta-data": meta, "user-data": user} {
		fle, err := extFs.OpenFile("/var/lib/cloud/seed/nocloud/"+f, os.O_CREATE|os.O_WRONLY)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}
		if _, err := fle.Write([]byte(data)); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
	}

	// // 6) Commit changes
	// if err := extFs.Sync(); err != nil {
	// 	return fmt.Errorf("syncing filesystem: %w", err)
	// }
	// if err := disk.Write(); err != nil {
	// 	return fmt.Errorf("writing disk: %w", err)
	// }
	return nil
}
