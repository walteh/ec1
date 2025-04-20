package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	alpineVersion     = "3.17.3"
	alpineDownloadURL = "https://dl-cdn.alpinelinux.org/alpine/v3.17/releases/x86_64/alpine-virt-3.17.3-x86_64.iso"
	alpineFilename    = "alpine-virt-3.17.3-x86_64.iso"
	qcow2Size         = "5G"
)

// CreateQCOW2Image creates a qcow2 image for a VM
func CreateQCOW2Image(ctx context.Context) (string, error) {
	fmt.Println("Creating QCOW2 image...")

	// Create directories if they don't exist
	if err := os.MkdirAll("images", 0755); err != nil {
		return "", fmt.Errorf("creating images directory: %w", err)
	}

	// Full paths for images
	isoPath := filepath.Join("images", alpineFilename)
	qcow2Path := filepath.Join("images", "alpine.qcow2")

	// Check if the ISO already exists, if not download it
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		fmt.Printf("Downloading Alpine Linux ISO (%s)...\n", alpineVersion)

		// Use curl to download the ISO
		downloadCmd := exec.CommandContext(ctx, "curl", "-L", "-o", isoPath, alpineDownloadURL)
		downloadCmd.Stdout = os.Stdout
		downloadCmd.Stderr = os.Stderr

		if err := downloadCmd.Run(); err != nil {
			return "", fmt.Errorf("downloading Alpine ISO: %w", err)
		}

		fmt.Println("Download complete")
	} else {
		fmt.Println("Using existing Alpine ISO")
	}

	// Check if the qcow2 already exists, if not create it
	if _, err := os.Stat(qcow2Path); os.IsNotExist(err) {
		fmt.Println("Creating QCOW2 image...")

		// Create the qcow2 image
		createCmd := exec.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", qcow2Path, qcow2Size)
		createCmd.Stdout = os.Stdout
		createCmd.Stderr = os.Stderr

		if err := createCmd.Run(); err != nil {
			return "", fmt.Errorf("creating QCOW2 image: %w", err)
		}

		fmt.Println("QCOW2 image created")
	} else {
		fmt.Println("Using existing QCOW2 image")
	}

	// Create cloud-init configuration for Alpine
	if err := createCloudInit(ctx, "images/cloud-init"); err != nil {
		return "", fmt.Errorf("creating cloud-init configuration: %w", err)
	}

	return qcow2Path, nil
}

// createCloudInit creates cloud-init configuration for Alpine Linux
func createCloudInit(ctx context.Context, dir string) error {
	fmt.Println("Creating cloud-init configuration...")

	// Create cloud-init directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cloud-init directory: %w", err)
	}

	// Create meta-data file
	metaData := `instance-id: alpine-001
local-hostname: alpine-vm`

	if err := os.WriteFile(filepath.Join(dir, "meta-data"), []byte(metaData), 0644); err != nil {
		return fmt.Errorf("writing meta-data: %w", err)
	}

	// Create user-data file
	userData := `#cloud-config
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
  - path: /etc/ec1/agent-setup.sh
    permissions: '0755'
    content: |
      #!/bin/sh
      echo "Setting up EC1 agent..."
      mkdir -p /etc/ec1
      wget -O /etc/ec1/agent http://192.168.64.1:8888/agent
      chmod +x /etc/ec1/agent
      echo "[Unit]
      Description=EC1 Agent
      After=network.target
      
      [Service]
      ExecStart=/etc/ec1/agent --host 0.0.0.0:9091 --mgt http://192.168.64.1:9090
      Restart=always
      
      [Install]
      WantedBy=multi-user.target" > /etc/systemd/system/ec1-agent.service
      systemctl enable ec1-agent
      systemctl start ec1-agent
      echo "EC1 agent setup complete"
runcmd:
  - apk update && apk upgrade
  - apk add qemu-img qemu-system-x86_64 glib libslirp-dev libvirt cloud-init
  - modprobe kvm_intel nested=1 || true
  - sh /etc/ec1/agent-setup.sh
  - echo "cloud-init setup complete" > /var/log/cloud-init-complete.log`

	if err := os.WriteFile(filepath.Join(dir, "user-data"), []byte(userData), 0644); err != nil {
		return fmt.Errorf("writing user-data: %w", err)
	}

	// Create cloud-init ISO image
	isoPath := filepath.Join("images", "cloud-init.iso")
	genisoArgs := []string{
		"mkisofs",
		"-output", isoPath,
		"-volid", "cidata",
		"-joliet",
		"-rock",
		filepath.Join(dir, "user-data"),
		filepath.Join(dir, "meta-data"),
	}

	// Check if genisoimage or mkisofs is available
	isoCmdName := "mkisofs"
	_, err := exec.LookPath(isoCmdName)
	if err != nil {
		isoCmdName = "genisoimage"
		_, err = exec.LookPath(isoCmdName)
		if err != nil {
			return fmt.Errorf("neither mkisofs nor genisoimage is available: %w", err)
		}
		// Update args for genisoimage
		genisoArgs[0] = isoCmdName
	}

	fmt.Printf("Creating cloud-init ISO with %s...\n", isoCmdName)
	genisoCmd := exec.CommandContext(ctx, genisoArgs[0], genisoArgs[1:]...)
	genisoCmd.Stdout = os.Stdout
	genisoCmd.Stderr = os.Stderr

	if err := genisoCmd.Run(); err != nil {
		return fmt.Errorf("creating cloud-init ISO: %w", err)
	}

	fmt.Println("Cloud-init configuration created")
	return nil
}
