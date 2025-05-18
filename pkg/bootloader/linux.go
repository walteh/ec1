package bootloader

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/mholt/archives"
	"github.com/u-root/u-root/pkg/cpio"
	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"

	"github.com/walteh/ec1/gen/binembed/lgia_linux_arm64"
)

func init() {
	initramfs.CompressReaders[initramfs.Xz] = func(r io.Reader) (io.Reader, error) {
		return (archives.Xz{}).OpenReader(r)
	}

	initramfs.CompressReaders[initramfs.Gzip] = func(r io.Reader) (io.Reader, error) {
		return (archives.Gz{}).OpenReader(r)
	}
}

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

var (
	// Embedded, statically‑compiled init+gRPC binary (as XZ data)
	initBin             = lgia_linux_arm64.BinaryXZ
	uncompressedInitBin = []byte{}
)

func uncompressInitBin(ctx context.Context) ([]byte, error) {
	if len(uncompressedInitBin) > 0 {
		return uncompressedInitBin, nil
	}

	fmt, rdr, err := archives.Identify(ctx, "", bytes.NewReader(initBin))
	if err != nil {
		return nil, errors.Errorf("identifying init bin: %w", err)
	}

	c, ok := fmt.(archives.Compression)
	if !ok {
		return nil, errors.Errorf("init bin is not a compression format: %s", fmt)
	}

	initBinXZ, err := c.OpenReader(rdr)
	if err != nil {
		return nil, errors.Errorf("opening xz reader: %w", err)
	}

	uncompressedInitBin, err = io.ReadAll(initBinXZ)
	if err != nil {
		return nil, errors.Errorf("reading uncompressed init bin: %w", err)
	}

	return uncompressedInitBin, nil
}

func PrepareInitramfsCpio(ctx context.Context, initramfsPath string) (string, error) {
	// Create a new file next to the original with .ec1 suffix
	outputPath := initramfsPath + ".ec1"

	// Open the CPIO initramfs file for reading
	cpioFile, err := os.Open(initramfsPath)
	if err != nil {
		return "", errors.Errorf("opening initramfs file %s: %w", initramfsPath, err)
	}
	defer cpioFile.Close()

	// Create the output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer outputFile.Close()

	gzr, err := (archives.Gz{}).OpenReader(cpioFile)
	if err != nil {
		return "", errors.Errorf("opening gzip reader: %w", err)
	}

	defer gzr.Close()

	gzw, err := (archives.Gz{}).OpenWriter(outputFile)
	if err != nil {
		return "", errors.Errorf("creating gzip writer: %w", err)
	}

	defer gzw.Close()

	buff := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buff, gzr)
	if err != nil {
		return "", errors.Errorf("copying cpio file: %w", err)
	}

	// convert gzr to readerat

	// Create CPIO reader and writer using the Newc format
	cpioReader := cpio.Newc.Reader(bytes.NewReader(buff.Bytes()))
	cpioWriter := cpio.Newc.Writer(gzw)

	// Get the uncompressed init binary
	uncompressedInitBinData, err := uncompressInitBin(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init binary: %w", err)
	}

	// Path for our custom init
	const customInitPath = "init"

	// Read all records from the input CPIO
	records, err := cpio.ReadAllRecords(cpioReader)
	if err != nil {
		return "", errors.Errorf("reading CPIO records: %w", err)
	}

	// Filter out any existing init.ec1 files
	var filteredRecords []cpio.Record
	for _, rec := range records {
		// if rec.Name == customInitPath {
		// 	continue
		// }
		if rec.Name == "init" {
			rec.Name = "init.real"
		}
		filteredRecords = append(filteredRecords, rec)
	}

	// Add our custom init file
	customInitRecord := cpio.StaticFile(customInitPath, string(uncompressedInitBinData), 0755)
	filteredRecords = append(filteredRecords, customInitRecord)

	// Write all records to the output file
	if err := cpio.WriteRecords(cpioWriter, filteredRecords); err != nil {
		return "", errors.Errorf("writing CPIO records: %w", err)
	}

	// Write trailer to finalize the archive
	if err := cpio.WriteTrailer(cpioWriter); err != nil {
		return "", errors.Errorf("writing CPIO trailer: %w", err)
	}

	slog.InfoContext(ctx, "custom init added to initramfs", "customInitPath", "/"+customInitPath, "outputPath", outputPath)

	return outputPath, nil
}

// PrepareRootFS opens the image file, adds our custom binary as /init.ec1.
// The original /sbin/init is left untouched. Creates a new file with .ec1 suffix.
func PrepareRootFS(ctx context.Context, imagePath string) (string, error) {
	// Create a new file next to the original with .ec1 suffix
	outputPath := imagePath + ".ec1"

	// Copy the original file to our new file
	srcFile, err := os.Open(imagePath)
	if err != nil {
		return "", errors.Errorf("opening source file %s: %w", imagePath, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", errors.Errorf("copying disk image: %w", err)
	}

	// We need to close the file before diskfs can open it
	destFile.Close()

	// Open the disk image
	d, err := diskfs.Open(outputPath, diskfs.WithOpenMode(diskfs.ReadWriteExclusive))
	if err != nil {
		return "", errors.Errorf("opening disk image %s: %w", outputPath, err)
	}
	defer d.Close()

	// Get the filesystem - we'll try partition 1 first
	// In a real-world scenario, you might need to identify the correct partition
	var fs filesystem.FileSystem

	// Try to get the filesystem from the first partition
	fs, err = d.GetFilesystem(1)
	if err != nil {
		// If that fails, try to get the filesystem from the entire disk (partition 0)
		fs, err = d.GetFilesystem(0)
		if err != nil {
			return "", errors.Errorf("getting filesystem: %w", err)
		}
	}

	// Path for our custom init
	const customInitPath = "/init.ec1"

	// Write the custom init
	newInitFile, err := fs.OpenFile(customInitPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return "", errors.Errorf("opening custom init file for writing: %w", err)
	}

	uncompressedInitBinData, err := uncompressInitBin(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init bin: %w", err)
	}

	_, err = newInitFile.Write(uncompressedInitBinData)
	if err != nil {
		newInitFile.Close()
		return "", errors.Errorf("writing custom init: %w", err)
	}
	newInitFile.Close()

	// Log what we did
	slog.InfoContext(ctx, "custom init added to rootfs", "customInitPath", customInitPath, "outputPath", outputPath)

	return outputPath, nil
}

// func identifyBootInitSystem(ctx context.Context, imagePath filesystem.FileSystem) (string, error) {

// 	// Pseudocode
// 	d, err := imagePath.OpenFile("/etc/os-release", os.O_RDONLY)
// 	if err != nil {
// 		return "", errors.Errorf("reading os-release: %w", err)
// 	}
// 	osrel, err := io.ReadAll(d)
// 	if err != nil {
// 		return "", errors.Errorf("reading os-release: %w", err)
// 	}
// 	if strings.Contains(string(osrel), "ID=alpine") {
// 		return "openrc", nil
// 	} else if strings.Contains(string(osrel), "ID=fedora") || strings.Contains(string(osrel), "ID=\"fedora\"") {
// 		return "systemd", nil
// 	} else if strings.Contains(string(osrel), "ID=ubuntu") || strings.Contains(string(osrel), "ID=debian") {
// 		// Distinguish Debian sysv vs systemd by checking for /sbin/init symlink
// 		fi, err := imagePath.OpenFile("/sbin/init", os.O_RDONLY)
// 		if err != nil {
// 			return "", errors.Errorf("statting init: %w", err)
// 		}
// 		if fi.IsSymlink() {
// 			target, err := imagePath.Readlink("/sbin/init")
// 			if err != nil {
// 				return "", errors.Errorf("reading init symlink: %w", err)
// 			}
// 			if strings.Contains(target, "systemd") {
// 				return "systemd", nil
// 			} else if strings.Contains(target, "busybox") {
// 				return "busybox", nil
// 			} else {
// 				return "sysvinit", nil // could be SysVinit's own binary
// 			}
// 		} else {
// 			// Not a symlink (could be an actual binary)
// 			data, err := io.ReadAll(fi)
// 			if err != nil {
// 				return "", errors.Errorf("reading init: %w", err)
// 			}
// 			if bytes.Contains(data, []byte("systemd")) {
// 				return "systemd", nil
// 			} else if bytes.Contains(data, []byte("BusyBox")) {
// 				return "busybox", nil
// 			} else if bytes.Contains(data, []byte("OpenRC")) {
// 				return "openrc", nil // if openrc-init has a signature
// 			} else {
// 				return "sysvinit", nil
// 			}
// 		}
// 	}

// }
