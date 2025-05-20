package host

import (
	"bytes"
	"context"
	"debug/elf"
	"debug/pe"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/mholt/archives"
)

var (
	ErrNoVmlinuxFound = errors.New("no vmlinux found")
)

// magicHeader represents a compression format's signature and decompression function
type magicHeader struct {
	name       string
	signature  []byte
	decompress func(r io.Reader) (io.ReadCloser, error)
}

// List of supported compression formats
var magicHeaders = []magicHeader{
	{
		name:      "gzip",
		signature: []byte{0x1f, 0x8b, 0x08},
		decompress: func(r io.Reader) (io.ReadCloser, error) {
			return (archives.Gz{}).OpenReader(r)
		},
	},
	{
		name:      "xz",
		signature: []byte{0xfd, 0x37, 0x7a, 0x58, 0x5a, 0x00},
		decompress: func(r io.Reader) (io.ReadCloser, error) {
			return (archives.Xz{}).OpenReader(r)
		},
	},
	{
		name:      "bzip2",
		signature: []byte{0x42, 0x5a, 0x68},
		decompress: func(r io.Reader) (io.ReadCloser, error) {
			return (archives.Bz2{}).OpenReader(r)
		},
	},
	{
		name:      "lz4",
		signature: []byte{0x02, 0x21, 0x4c, 0x18},
		decompress: func(r io.Reader) (io.ReadCloser, error) {
			return (archives.Lz4{}).OpenReader(r)
		},
	},
	{
		name:      "zstd",
		signature: []byte{0x28, 0xb5, 0x2f, 0xfd},
		decompress: func(r io.Reader) (io.ReadCloser, error) {
			return (archives.Zstd{}).OpenReader(r)
		},
	},
}

// PE32+ file header magic for EFI applications
var peHeaderMagic = []byte{0x4d, 0x5a} // MZ

// ARM64 boot image header magic: "LINUX ARM64 KERNEL" (with nulls between)
var arm64Magic = []byte{
	0x41, 0x52, 0x4d, 0x64, // ARMd
}

// isELF checks if a reader contains a valid ELF file
func isELF(r io.ReaderAt) bool {
	_, err := elf.NewFile(r)
	return err == nil
}

// isPE checks if the file is a PE32+ EFI application
func isPE(r io.ReaderAt) bool {
	file, err := pe.NewFile(r)
	if err != nil {
		return false
	}
	defer file.Close()

	// Check if it's actually an ARM64 PE binary by examining machine type
	// ARM64 machine type is 0xAA64
	return file.Machine == 0xAA64
}

// isArm64BootImage checks if the file is an ARM64 kernel boot image
func isArm64BootImage(r io.ReaderAt) bool {
	// ARM64 boot images have a specific signature at offset 0x38
	buf := make([]byte, 64)
	n, err := r.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return false
	}
	if n < 64 {
		return false
	}

	// Check for ARM64 identifier at offset 0x38
	return bytes.Equal(buf[0x38:0x38+4], arm64Magic)
}

// isValidKernel checks if the file is a valid kernel (ELF, PE32+ EFI, or ARM64 boot image)
func isValidKernel(r io.ReaderAt) bool {
	return isELF(r) || isArm64BootImage(r) || isPE(r)
}

// isValidKernelWithSize checks if the file is a valid kernel and also verifies
// it's a reasonable size for a kernel
func isValidKernelWithSize(r io.ReaderAt, size int64) bool {
	// For PE files, they can be smaller than typical Linux kernels
	// Let's lower the threshold for these files
	if isPE(r) && size >= 128*1024 { // 128KB for PE files
		return true
	}

	// For other kernel types, check if it's a valid format
	if isELF(r) || isArm64BootImage(r) {
		// Ensure the kernel is a reasonable size
		// Most kernels are at least several megabytes
		return size >= 1*1024*1024 // At least 1MB
	}

	return false
}

// findSignatures efficiently scans a file for magic signatures using parallel processing
func findSignatures(ctx context.Context, file *os.File, header magicHeader, fileSize int64) ([]int64, error) {
	var positions []int64
	var mu sync.Mutex

	// Use a reasonable buffer size - not too large to consume too much memory
	// but large enough to minimize disk reads
	const bufferSize = 4 * 1024 * 1024 // 4MB

	// Number of goroutines to use - use core count but limit to avoid too many goroutines
	workerCount := runtime.NumCPU()
	if workerCount > 16 {
		workerCount = 16
	}

	// Calculate chunk size for each worker
	chunkSize := fileSize / int64(workerCount)
	if chunkSize < bufferSize {
		// For small files, use a single goroutine
		workerCount = 1
		chunkSize = fileSize
	}

	var wg sync.WaitGroup
	wg.Add(workerCount)

	// Process each chunk in a separate goroutine
	for i := 0; i < workerCount; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize
		if i == workerCount-1 || end > fileSize {
			end = fileSize
		}

		// Ensure we don't miss signatures at chunk boundaries
		// We need to extend each chunk to include potential signatures that might be split
		if i > 0 {
			start -= int64(len(header.signature) - 1)
		}
		if start < 0 {
			start = 0
		}

		go func(startPos, endPos int64) {
			defer wg.Done()

			// Create a buffer for the chunk
			chunkBuffer := make([]byte, endPos-startPos)
			n, err := file.ReadAt(chunkBuffer, startPos)
			if err != nil && err != io.EOF {
				return
			}
			chunkBuffer = chunkBuffer[:n] // Resize to actual bytes read

			// Find all occurrences of the signature in this chunk
			var chunkPositions []int64
			for i := 0; i <= len(chunkBuffer)-len(header.signature); i++ {
				select {
				case <-ctx.Done():
					return
				default:
					if bytes.Equal(chunkBuffer[i:i+len(header.signature)], header.signature) {
						pos := startPos + int64(i)
						chunkPositions = append(chunkPositions, pos)
					}
				}
			}

			// Add found positions to the global list
			if len(chunkPositions) > 0 {
				mu.Lock()
				positions = append(positions, chunkPositions...)
				mu.Unlock()
			}
		}(start, end)
	}

	wg.Wait()
	return positions, nil
}

// simpleDecompress is a simplified version that just pipes data through a decompressor
// and checks periodically if the result is a valid kernel
func simpleDecompress(ctx context.Context, file *os.File, pos int64, header magicHeader) ([]byte, bool) {
	// Create a temporary file for the output
	tmpFile, err := os.CreateTemp("", "vmlinux-*")
	if err != nil {
		slog.ErrorContext(ctx, "failed to create temp file", "error", err)
		return nil, false
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)
	tmpFile.Close()

	// Open the output file for writing
	outFile, err := os.OpenFile(tmpName, os.O_WRONLY, 0644)
	if err != nil {
		slog.ErrorContext(ctx, "error opening output file", "error", err)
		return nil, false
	}

	// Set a timeout context for the entire operation
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second) // Increased timeout
	defer cancel()

	// Channel to signal when decompression is done or failed
	done := make(chan struct{})

	// Variables to track found kernels
	type foundKernel struct {
		data []byte
		size int64
	}
	var foundKernels []foundKernel
	var kernelMutex sync.Mutex

	// Start the decompression process
	go func() {
		defer close(done)
		defer outFile.Close()

		// Seek to the position where we found the magic header
		_, err := file.Seek(pos, 0)
		if err != nil {
			slog.ErrorContext(ctx, "failed to seek to position", "pos", pos, "error", err)
			return
		}

		// Create the decompressor
		decompressor, err := header.decompress(file)
		if err != nil {
			slog.ErrorContext(ctx, "error creating decompressor", "error", err, "format", header.name)
			return
		}
		defer decompressor.Close()

		// We'll read in chunks so we can check periodically
		buffer := make([]byte, 4*1024*1024) // 4MB chunks
		totalWritten := int64(0)

		for {
			select {
			case <-ctx.Done():
				slog.InfoContext(ctx, "decompression stopped due to timeout", "format", header.name)
				return
			default:
				n, err := decompressor.Read(buffer)
				if n > 0 {
					if _, err := outFile.Write(buffer[:n]); err != nil {
						slog.ErrorContext(ctx, "error writing to output file", "error", err)
						return
					}
					totalWritten += int64(n)

					// Check regularly, but not too often
					if totalWritten%16_000_000 < 4*1024*1024 && totalWritten > 0 {
						outFile.Sync()

						// Check if we have a valid kernel
						checkFile, err := os.Open(tmpName)
						if err == nil {
							fileInfo, _ := checkFile.Stat()
							fileSize := fileInfo.Size()

							if isValidKernel(checkFile) {
								// We found something that looks like a kernel...
								slog.InfoContext(ctx, "found potential kernel during decompression",
									"format", header.name, "bytes", totalWritten, "type", getKernelType(checkFile))

								// Check if it's a valid kernel with appropriate size
								checkFile.Seek(0, 0) // Rewind for next check
								if isValidKernelWithSize(checkFile, fileSize) {
									// Read the data for this potentially valid kernel
									checkFile.Seek(0, 0)
									data, err := io.ReadAll(checkFile)
									if err == nil {
										kernelMutex.Lock()
										foundKernels = append(foundKernels, foundKernel{
											data: data,
											size: fileSize,
										})
										kernelMutex.Unlock()

										slog.InfoContext(ctx, "saved kernel candidate",
											"format", header.name, "size_mb", fileSize/1024/1024,
											"type", getKernelType(checkFile))
									}
								}
							}
							checkFile.Close()
						}
					}
				}

				if err != nil {
					if err != io.EOF {
						slog.ErrorContext(ctx, "error reading from decompressor", "error", err, "bytes_decompressed", totalWritten)
					} else {
						slog.InfoContext(ctx, "finished decompression", "format", header.name, "bytes_decompressed", totalWritten)
					}
					return
				}
			}
		}
	}()

	// Wait for the decompression to complete or timeout
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			slog.InfoContext(ctx, "decompression timed out", "format", header.name)
		}
		return nil, false
	case <-done:
		// Decompression completed, check if we got a valid kernel
	}

	// If we found any kernel candidates, return the largest one
	kernelMutex.Lock()
	defer kernelMutex.Unlock()

	if len(foundKernels) > 0 {
		// Find the largest kernel
		var bestKernel foundKernel
		for _, k := range foundKernels {
			if k.size > bestKernel.size {
				bestKernel = k
			}
		}

		slog.InfoContext(ctx, "selected largest kernel candidate",
			"format", header.name, "size_mb", bestKernel.size/1024/1024,
			"candidates", len(foundKernels))

		return bestKernel.data, true
	}

	// Check if the final output is a valid kernel, as a fallback
	checkFile, err := os.Open(tmpName)
	if err != nil {
		return nil, false
	}
	defer checkFile.Close()

	fileInfo, _ := checkFile.Stat()
	fileSize := fileInfo.Size()

	if isValidKernelWithSize(checkFile, fileSize) {
		slog.InfoContext(ctx, "found valid kernel in final check",
			"format", header.name, "size_mb", fileSize/1024/1024,
			"type", getKernelType(checkFile))

		// Read the data and return it
		checkFile.Seek(0, 0)
		data, err := io.ReadAll(checkFile)
		if err != nil {
			slog.ErrorContext(ctx, "error reading valid kernel data", "error", err)
			return nil, false
		}
		return data, true
	}

	slog.InfoContext(ctx, "no valid kernel found in decompressed data", "format", header.name)
	return nil, false
}

// getKernelType returns a string describing the detected kernel type
func getKernelType(r io.ReaderAt) string {
	if isPE(r) {
		return "PE32+ EFI"
	} else if isELF(r) {
		return "ELF"
	} else if isArm64BootImage(r) {
		return "ARM64 Boot Image"
	}
	return "Unknown"
}

// extractVmlinuxNative attempts to extract vmlinux from a kernel image
func ExtractVmlinuxNative(ctx context.Context, imagePath string) (io.ReadCloser, error) {
	// Open the image file
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("opening image file: %w", err)
	}
	defer file.Close()

	// Get file size
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("getting file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// First check if the file is already a valid kernel
	if fileSize > 0 {
		fileReader := io.NewSectionReader(file, 0, fileSize)
		if isValidKernel(fileReader) {
			// Check for appropriate size too
			fileReader.Seek(0, 0) // Rewind for next check
			if isValidKernelWithSize(fileReader, fileSize) {
				slog.InfoContext(ctx, "file is already a valid kernel, returning as-is",
					"size_mb", fileSize/1024/1024, "type", getKernelType(fileReader))
				// File is already a valid kernel, read and return it
				data := make([]byte, fileSize)
				_, err := file.ReadAt(data, 0)
				if err != nil && err != io.EOF {
					return nil, fmt.Errorf("reading uncompressed file: %w", err)
				}
				return io.NopCloser(bytes.NewReader(data)), nil
			} else {
				slog.InfoContext(ctx, "file looks like a kernel but is too small, trying decompression",
					"size_kb", fileSize/1024)
			}
		}
	}

	// Create channels for parallelizing format checks
	resultChan := make(chan []byte, len(magicHeaders))
	errorChan := make(chan error, len(magicHeaders))
	doneChan := make(chan struct{})

	// Create a cancellable context to abort other goroutines when we find a valid kernel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Process all formats in parallel
	var wg sync.WaitGroup
	for _, header := range magicHeaders {
		wg.Add(1)
		go func(hdr magicHeader) {
			defer wg.Done()

			// Find all occurrences of this format's signature
			positions, err := findSignatures(ctx, file, hdr, fileSize)
			if err != nil {
				select {
				case errorChan <- fmt.Errorf("error finding %s signatures: %w", hdr.name, err):
				case <-ctx.Done():
				}
				return
			}

			slog.InfoContext(ctx, "found signature positions", "format", hdr.name, "count", len(positions))

			// Try to decompress at each position found
			for _, pos := range positions {
				select {
				case <-ctx.Done():
					return
				default:
					data, found := simpleDecompress(ctx, file, pos, hdr)
					if found {
						// Found a valid kernel image, signal to stop other goroutines
						select {
						case resultChan <- data:
							cancel()
						case <-ctx.Done():
						}
						return
					}
				}
			}
		}(header)
	}

	// Wait for either a result or all goroutines to finish
	go func() {
		wg.Wait()
		close(doneChan)
	}()

	// Wait for result, error, or completion
	select {
	case result := <-resultChan:
		return io.NopCloser(bytes.NewReader(result)), nil
	case err := <-errorChan:
		return nil, err
	case <-doneChan:
		return nil, fmt.Errorf("no valid kernel found in %s", imagePath)
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			// This might be because we found a result
			select {
			case result := <-resultChan:
				return io.NopCloser(bytes.NewReader(result)), nil
			default:
				return nil, fmt.Errorf("operation cancelled")
			}
		}
		return nil, fmt.Errorf("operation timed out")
	}
}
