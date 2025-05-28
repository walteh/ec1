package bootloader

import (
	_ "unsafe"

	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kdomanski/iso9660"
	"github.com/mholt/archives"
	"github.com/u-root/u-root/pkg/cpio"
	"gitlab.com/tozd/go/errors"
	"go.pdmccormick.com/initramfs"

	"github.com/walteh/ec1/gen/harpoon/harpoon_harpoond_arm64"
	"github.com/walteh/ec1/pkg/host"
)

func init() {
	initramfs.CompressReaders[initramfs.Zstd] = func(r io.Reader) (io.Reader, error) {
		return (archives.Zstd{}).OpenReader(r)
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
	initBin             = harpoon_harpoond_arm64.BinaryXZ
	uncompressedInitBin = []byte{}
)

func UncompressInitBin(ctx context.Context) ([]byte, error) {
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

type discarder struct {
	r   io.Reader
	pos int64
	dup *os.File
}

func NewDiscarder(r io.Reader) *discarder {
	df, err := os.CreateTemp("", "initramfs-discarder")
	if err != nil {
		panic(err)
	}
	return &discarder{
		r:   r,
		pos: 0,
		dup: df,
	}
}

// ReadAt implements ReadAt for a discarder.
// It is an error for the offset to be negative.
func (r *discarder) ReadAt(p []byte, off int64) (int, error) {

	if off-r.pos < 0 {
		return r.dup.ReadAt(p, off)
		// return 0, fmt.Errorf("negative seek on discarder not allowed: off=%d, pos=%d", off, r.pos)
	}
	if off != r.pos {
		i, err := io.Copy(r.dup, io.LimitReader(r.r, off-r.pos))
		if err != nil || i != off-r.pos {
			return 0, err
		}
		r.pos += i
	}
	n, err := io.ReadFull(io.TeeReader(r.r, r.dup), p)
	if err != nil {
		return n, err
	}
	r.pos += int64(n)
	return n, err
}

type InMemoryReaderAt struct {
	reader io.Reader
	buffer []byte
	offset int64
	mutex  sync.Mutex
}

func NewInMemoryReaderAt(r io.Reader) *InMemoryReaderAt {
	return &InMemoryReaderAt{
		reader: r,
		buffer: make([]byte, 1024),
		offset: 0,
	}
}

func (r *InMemoryReaderAt) ReadAt(p []byte, off int64) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if off < 0 {
		return 0, fmt.Errorf("negative offset not allowed: %d", off)
	}

	if off >= int64(len(r.buffer)) {
		return 0, io.EOF
	}

	copy(p, r.buffer[off:])
	return len(p), nil
}

func PrepareEmptyIso(ctx context.Context, dir string) (string, error) {
	iso, err := iso9660.NewWriter()
	if err != nil {
		return "", errors.Errorf("creating iso writer: %w", err)
	}
	uncompressedInitBinData, err := UncompressInitBin(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init binary: %w", err)
	}

	err = iso.AddFile(bytes.NewReader(uncompressedInitBinData), "init")
	if err != nil {
		return "", errors.Errorf("adding init file to iso: %w", err)
	}

	tmp, err := os.Create(filepath.Join(dir, "init.ec1.iso"))
	if err != nil {
		return "", errors.Errorf("creating temp file: %w", err)
	}
	defer tmp.Close()

	err = iso.WriteTo(tmp, "init.ec1.volume")
	if err != nil {
		return "", errors.Errorf("writing iso: %w", err)
	}

	return tmp.Name(), nil
}

func PrepareEmptyInitramfs(ctx context.Context, dir string) (string, error) {

	uncompressedInitBinData, err := UncompressInitBin(ctx)
	if err != nil {
		return "", errors.Errorf("uncompressing init binary: %w", err)
	}

	f, err := os.Create(filepath.Join(dir, "initramfs.cpio.gz"))
	if err != nil {
		return "", errors.Errorf("creating initramfs.cpio.gz: %w", err)
	}
	defer f.Close()

	img := cpio.Newc.Writer(f) // wrap gzip+cpio internally

	records := []cpio.Record{}
	customInitRecord := cpio.StaticFile("init", string(uncompressedInitBinData), 0755)
	records = append(records, customInitRecord)

	// Write all records to the output file
	if err := cpio.WriteRecords(img, records); err != nil {
		return "", errors.Errorf("writing CPIO records: %w", err)
	}

	if err := cpio.WriteTrailer(img); err != nil {
		return "", errors.Errorf("writing CPIO trailer: %w", err)
	}

	return f.Name(), nil

}

func PrepareInitramfsCpio(ctx context.Context, rdr io.Reader) (io.ReadCloser, error) {
	// Create a new file next to the original with .ec1 suffix

	// Open the CPIO initramfs file for reading
	inputFile, err := os.CreateTemp("", "initramfs.cpio.gz")
	if err != nil {
		return nil, errors.Errorf("opening initramfs file %s: %w", inputFile.Name(), err)
	}
	defer inputFile.Close()

	_, err = io.Copy(inputFile, rdr)
	if err != nil {
		return nil, errors.Errorf("copying initramfs: %w", err)
	}

	// defer outputFile.Close()

	format, rdr, err := archives.Identify(ctx, "", inputFile)
	if err != nil && err != archives.NoMatch {
		return nil, errors.Errorf("identifying initramfs file %s: %w", inputFile.Name(), err)
	}

	noMatch := err == archives.NoMatch
	if !noMatch {
		if format, ok := format.(archives.Compression); ok {
			slog.InfoContext(ctx, "initramfs is compressed", "format", fmt.Sprintf("%T", format))
			rdrz, err := format.OpenReader(rdr)
			if err != nil {
				return nil, errors.Errorf("opening compression reader: %w", err)
			}
			defer rdrz.Close()
			rdr = rdrz
		} else {
			return nil, errors.Errorf("initramfs file %s is not a compression format: %s", inputFile.Name(), format)
		}
	}

	// ok := initramfs.NewReader(rdr)
	// ok.WriteTo()

	// Create CPIO reader and writer using the Newc format
	cpioReader := cpio.Newc.Reader(NewDiscarder(rdr))

	// Get the uncompressed init binary
	uncompressedInitBinData, err := UncompressInitBin(ctx)
	if err != nil {
		return nil, errors.Errorf("uncompressing init binary: %w", err)
	}

	// Path for our custom init
	const customInitPath = "init"

	// Read all records from the input CPIO
	records, err := cpio.ReadAllRecords(cpioReader)
	if err != nil {
		return nil, errors.Errorf("reading CPIO records: %w", err)
	}

	// Filter out any existing init.ec1 files
	var filteredRecords []cpio.Record
	for _, rec := range records {
		// fmt.Println("rec.Name", rec.Name)
		if rec.Name == "init" {
			rec.Name = "init.real"
		}
		filteredRecords = append(filteredRecords, rec)
	}

	// Add our custom init file
	customInitRecord := cpio.StaticFile(customInitPath, string(uncompressedInitBinData), 0755)
	filteredRecords = append(filteredRecords, customInitRecord)

	outputFile, err := os.CreateTemp("", "initramfs.cpio.gz.ec1")
	if err != nil {
		return nil, errors.Errorf("creating output file %s: %w", outputFile.Name(), err)
	}
	defer outputFile.Close()

	outputWriter := io.Writer(outputFile)

	if _, ok := format.(archives.Compression); ok {
		outputWriterd, err := (&archives.Gz{}).OpenWriter(outputWriter)
		if err != nil {
			return nil, errors.Errorf("opening compression writer: %w", err)
		}
		defer outputWriterd.Close()
		outputWriter = outputWriterd
	}

	cpioWriter := cpio.Newc.Writer(outputWriter)

	// archive := cpio.ArchiveFromRecords(filteredRecords)

	// Write all records to the output file
	if err := cpio.WriteRecords(cpioWriter, filteredRecords); err != nil {
		return nil, errors.Errorf("writing CPIO records: %w", err)
	}

	// Write trailer to finalize the archive
	if err := cpio.WriteTrailer(cpioWriter); err != nil {
		return nil, errors.Errorf("writing CPIO trailer: %w", err)
	}

	open, err := os.Open(outputFile.Name())
	if err != nil {
		return nil, errors.Errorf("opening output file %s: %w", outputFile.Name(), err)
	}

	slog.InfoContext(ctx, "custom init added to initramfs", "customInitPath", "/"+customInitPath, "outputPath", outputFile)

	return open, nil
}

func PrepareKernel(ctx context.Context, kernelPath string) (string, error) {
	// Create a new file next to the original with .ec1 suffix
	outputPath := kernelPath + ".ec1"

	fle, err := os.Open(kernelPath)
	if err != nil {
		return "", errors.Errorf("opening kernel: %w", err)
	}
	defer fle.Close()

	// just check and decompress if needed

	// ext, err := vmlinuz.NewKernelExtractor(fle, false)
	// rdr, err := ext.ExtractAll(ctx)
	// if err != nil {
	// 	return "", errors.Errorf("extracting kernel: %w", err)
	// }

	rdr, err := host.ExtractVmlinuxNative(ctx, kernelPath)
	if err != nil {
		return "", errors.Errorf("extracting kernel: %w", err)
	}

	// Copy the kernel to the output file
	dstFile, err := os.Create(outputPath)
	if err != nil {
		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, rdr)
	if err != nil {
		return "", errors.Errorf("copying kernel: %w", err)
	}

	return outputPath, nil
}

// PrepareRootFS opens the image file, adds our custom binary as /init.ec1.
// The original /sbin/init is left untouched. Creates a new file with .ec1 suffix.
// func PrepareRootFS(ctx context.Context, imagePath string) (string, error) {
// 	// Create a new file next to the original with .ec1 suffix
// 	outputPath := imagePath + ".ec1"

// 	// Copy the original file to our new file
// 	srcFile, err := os.Open(imagePath)
// 	if err != nil {
// 		return "", errors.Errorf("opening source file %s: %w", imagePath, err)
// 	}
// 	defer srcFile.Close()

// 	destFile, err := os.Create(outputPath)
// 	if err != nil {
// 		return "", errors.Errorf("creating output file %s: %w", outputPath, err)
// 	}
// 	defer destFile.Close()

// 	_, err = io.Copy(destFile, srcFile)
// 	if err != nil {
// 		return "", errors.Errorf("copying disk image: %w", err)
// 	}

// 	// We need to close the file before diskfs can open it
// 	destFile.Close()

// 	backend, err := file.OpenFromPath(outputPath, false)
// 	if err != nil {
// 		return "", errors.Errorf("opening disk image %s: %w", outputPath, err)
// 	}
// 	defer backend.Close()

// 	d, err := diskfs.OpenBackend(backend, diskfs.WithOpenMode(diskfs.ReadWrite))
// 	if err != nil {
// 		return "", errors.Errorf("opening disk image %s: %w", outputPath, err)
// 	}
// 	defer d.Close()

// 	// // Open the disk image
// 	// d, err := diskfs.Open(outputPath, diskfs.WithOpenMode(diskfs.ReadWriteExclusive))
// 	// if err != nil {
// 	// 	return "", errors.Errorf("opening disk image %s: %w", outputPath, err)
// 	// }
// 	// defer d.Close()

// 	// // Get the filesystem - we'll try partition 1 first
// 	// // In a real-world scenario, you might need to identify the correct partition
// 	// var fs filesystem.FileSystem

// 	// for i, p := range d.Table.GetPartitions() {
// 	// 	if p.(*gpt.Partition).Name == "p.lxroot" {
// 	// 		fs, err = d.GetFilesystem(i)
// 	// 		if err != nil {
// 	// 			return "", errors.Errorf("getting filesystem: %w", err)
// 	// 		}
// 	// 		break
// 	// 	}
// 	// }
// 	err = d.ReReadPartitionTable()
// 	if err != nil {
// 		return "", errors.Errorf("re-reading partition table: %w", err)
// 	}

// 	slog.InfoContext(ctx, "partition table",
// 		"table", d.Table,
// 		"data", d.Size,
// 		"physical", d.PhysicalBlocksize,
// 		"logical", d.LogicalBlocksize,
// 	)

// 	var exsitingFs filesystem.FileSystem

// 	if d.Table == nil {
// 		fs, err := ext4.Read(backend, d.Size, 0, 0)
// 		if err != nil {
// 			return "", errors.Errorf("reading squashfs: %w", err)
// 		}
// 		exsitingFs = fs

// 		// ok, err := d.GetFilesystem(0)
// 		// if err != nil {
// 		// 	return "", errors.Errorf("getting filesystem: %w", err)
// 		// }
// 		// if ok != nil {
// 		// 	return "", errors.Errorf("disk image has no partition table")
// 		// }
// 		// exsitingFs = ok
// 	} else {

// 		// Find the ext4 partition (e.g., partition 3)
// 		for i, part := range d.Table.GetPartitions() {
// 			if part.(*gpt.Partition).Type == gpt.LinuxLVM || part.(*gpt.Partition).Type == gpt.LinuxFilesystem {
// 				fsTemp, err := d.GetFilesystem(i + 1) // partitions are 1-based
// 				if err != nil {
// 					continue
// 				}
// 				slog.InfoContext(ctx, "partition found", "type", fsTemp.Type())
// 				if e4, ok := fsTemp.(*ext4.FileSystem); ok {
// 					exsitingFs = e4
// 					break
// 				}
// 			} else {
// 				slog.InfoContext(ctx, "partition found", "type", part.(*gpt.Partition).Type)
// 			}
// 		}
// 		if exsitingFs == nil {
// 			return "", errors.Errorf("ext4 partition not found")
// 		}
// 	}

// 	// var newFs filesystem.FileSystem
// 	// if exsitingFs.Type() == filesystem.TypeSquashfs {
// 	// 	newFs, err = squashfs.Create(backend, d.Size, 0, 0)
// 	// 	if err != nil {
// 	// 		return "", errors.Errorf("creating squashfs: %w", err)
// 	// 	}
// 	// } else if exsitingFs.Type() == filesystem.TypeExt4 {
// 	// 	newFs, err = ext4.Create(backend, d.Size, 0, 0, nil)
// 	// 	if err != nil {
// 	// 		return "", errors.Errorf("creating ext4: %w", err)
// 	// 	}
// 	// } else {
// 	// 	return "", errors.Errorf("unsupported filesystem type: %s", exsitingFs.Type())
// 	// }

// 	// fse := fs.(*ext4.FileSystem)

// 	dirents, err := exsitingFs.ReadDir("/")
// 	if err != nil {
// 		slog.ErrorContext(ctx, "reading root directory", "error", err)
// 	} else {

// 		for _, dirent := range dirents {
// 			slog.InfoContext(ctx, "directory entry", "name", dirent.Name(), "isDir", dirent.IsDir())
// 		}
// 	}

// 	// file, err := exsitingFs.OpenFile("/etc/os-release", os.O_RDONLY)
// 	// if err != nil {
// 	// 	slog.ErrorContext(ctx, "opening os-release", "error", err)
// 	// } else {
// 	// 	defer file.Close()
// 	// 	data := make([]byte, 1024)
// 	// 	n, err := file.Read(data)
// 	// 	if err != nil {
// 	// 		slog.ErrorContext(ctx, "reading os-release", "error", err)
// 	// 	} else {
// 	// 		slog.InfoContext(ctx, "os-release", "data", string(data[:n]))
// 	// 	}
// 	// }

// 	// slog.InfoContext(ctx, "init", "path", "/sbin/init", "size", len(rdat))

// 	// // Try to get the filesystem from the first partition
// 	// fs, err = d.GetFilesystem(1)
// 	// if err != nil {
// 	// 	// If that fails, try to get the filesystem from the entire disk (partition 0)
// 	// 	fs, err = d.GetFilesystem(0)
// 	// 	if err != nil {
// 	// 		return "", errors.Errorf("getting filesystem: %w", err)
// 	// 	}
// 	// }

// 	// Path for our custom init
// 	const customInitPath = "/init"

// 	// if true {
// 	// 	return "", errors.New("not implemented")
// 	// }

// 	// dirents, err := exsitingFs.ReadDir("/")
// 	// if err != nil {
// 	// 	slog.ErrorContext(ctx, "reading root directory", "error", err)
// 	// } else {
// 	// 	for _, dirent := range dirents {
// 	// 		if dirent.Name() == "sbin" {
// 	// 			if dirent.IsDir() {
// 	// 				slog.InfoContext(ctx, "found sbin directory", "path", dirent.Name())
// 	// 			} else {
// 	// 				slog.InfoContext(ctx, "found sbin file", "path", dirent.Name())
// 	// 			}
// 	// 		}
// 	// 	// }
// 	// }

// 	// realInitPath := ""
// 	// potentialInitPaths := []string{"/init", "/sbin/init", "/usr/sbin/init", "/usr/lib/systemd/systemd", "/usr/lib/systemd/systemd-journald"}
// 	// for _, path := range potentialInitPaths {
// 	// 	existingInit, err := exsitingFs.OpenFile(path, os.O_RDONLY)
// 	// 	if err != nil {
// 	// 		if strings.Contains(err.Error(), "inode is type 3,") {
// 	// 			// we gots a symlink
// 	// 			realInitPath = path
// 	// 			break
// 	// 		}
// 	// 	} else {
// 	// 		existingInit.Close()
// 	// 		realInitPath = path
// 	// 		break
// 	// 	}
// 	// }

// 	// if /sbin is a file its prob a symlink

// 	// prevInit := "/sbin/init"
// 	// // move the existing init to /init.real
// 	// existingInit, err := exsitingFs.OpenFile("/sbin/init", os.O_RDONLY)
// 	// if err != nil && !os.IsNotExist(err) {
// 	// 	existingInit, err = exsitingFs.OpenFile("/usr/sbin/init", os.O_RDONLY)
// 	// 	if err != nil && !strings.Contains(err.Error(), "no such file or directory") {
// 	// 		return "", errors.Errorf("opening existing init: %w", err)
// 	// 	}
// 	// 	prevInit = "/usr/sbin/init"
// 	// }

// 	// if existingInit != nil {
// 	// 	existingInit.Close()
// 	// }

// 	// opt

// 	oldInitFile, err := exsitingFs.OpenFile(realInitPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
// 	if err != nil {
// 		return "", errors.Errorf("opening init: %w", err)
// 	}
// 	defer oldInitFile.Close()

// 	_, err = io.Copy(oldInitFile, existingInit)
// 	if err != nil {
// 		return "", errors.Errorf("copying init: %w", err)
// 	}

// 	err = oldInitFile.Close()
// 	if err != nil {
// 		return "", errors.Errorf("closing init: %w", err)
// 	}

// 	// Write the custom init
// 	newInitFile, err := exsitingFs.OpenFile(customInitPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
// 	if err != nil {
// 		return "", errors.Errorf("opening custom init file for writing: %w", err)
// 	}

// 	uncompressedInitBinData, err := UncompressInitBin(ctx)
// 	if err != nil {
// 		return "", errors.Errorf("uncompressing init bin: %w", err)
// 	}

// 	_, err = newInitFile.Write(uncompressedInitBinData)
// 	if err != nil {
// 		newInitFile.Close()
// 		return "", errors.Errorf("writing custom init: %w", err)
// 	}
// 	newInitFile.Close()

// 	// Log what we did
// 	slog.InfoContext(ctx, "custom init added to rootfs", "customInitPath", customInitPath, "outputPath", outputPath)

// 	return outputPath, nil
// }

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

type bufferedReaderAt struct {
	reader io.Reader
	buffer []byte
	offset int64
	mutex  sync.Mutex
}

func NewBufferedReaderAt(r io.Reader, bufferSize int) *bufferedReaderAt {
	return &bufferedReaderAt{
		reader: r,
		buffer: make([]byte, bufferSize),
	}
}

func (b *bufferedReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if off < 0 {
		return 0, nil
	}

	if off < b.offset {
		return 0, nil
	}

	if off >= b.offset+int64(len(b.buffer)) {
		b.offset = off
		read, err := io.ReadFull(b.reader, b.buffer)
		if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
			return 0, err
		}
		if read == 0 {
			return 0, io.EOF
		}
	}

	start := off - b.offset
	available := len(b.buffer) - int(start)

	if available <= 0 {
		return 0, io.EOF
	}

	readLen := len(p)
	if readLen > available {
		readLen = available
	}

	copy(p, b.buffer[start:start+int64(readLen)])
	return readLen, nil
}
