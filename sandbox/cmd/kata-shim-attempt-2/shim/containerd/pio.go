package containerd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/containerd/fifo"
	"github.com/containerd/log"
)

// ProcessIO holds process I/O information for a container process
type ProcessIO struct {
	stdin  io.ReadCloser  // We read from this (containerd writes to it)
	stdout io.WriteCloser // We write to this (containerd reads from it)
	stderr io.WriteCloser // We write to this (containerd reads from it)

	stdinPath  string
	stdoutPath string
	stderrPath string

	logFile *os.File
}

// NewProcessIO creates new process IO for a container task
func NewProcessIO(ctx context.Context, stdin, stdout, stderr, logFilePath string) (*ProcessIO, error) {
	var sio ProcessIO

	sio.stdinPath = stdin
	sio.stdoutPath = stdout
	sio.stderrPath = stderr

	// Set up logging if a log file path was provided
	if logFilePath != "" {
		// Ensure directory exists
		logDir := filepath.Dir(logFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		var err error
		sio.logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, err
		}

		log.L.WithField("log_path", logFilePath).Debug("Created log file for container output")

		// Write header to log file
		fmt.Fprintf(sio.logFile, "--- CONTAINER I/O LOG ---\n")
	}

	// Handle stdin if requested
	if stdin != "" {
		if _, err := os.Stat(stdin); err == nil {
			// Open the FIFO for reading (containerd writes to it, we read from it)
			stdinFifo, err := fifo.OpenFifo(ctx, stdin, syscall.O_RDONLY, 0)
			if err != nil {
				if sio.logFile != nil {
					sio.logFile.Close()
				}
				return nil, err
			}

			// If we have a log file, create a tee reader for stdin
			if sio.logFile != nil {
				// Use pipe to allow forwarding while logging
				pipeReader, pipeWriter := io.Pipe()
				sio.stdin = pipeReader

				// Start goroutine to read from FIFO, log, and forward to pipe
				go func() {
					defer pipeWriter.Close()
					defer stdinFifo.Close()

					buf := make([]byte, 4096)
					for {
						n, err := stdinFifo.Read(buf)
						if n > 0 {
							// Log to file with STDIN prefix
							sio.logFile.Write([]byte("STDIN: "))
							sio.logFile.Write(buf[:n])
							if buf[n-1] != '\n' {
								sio.logFile.Write([]byte("\n"))
							}

							// Forward to pipe
							if _, werr := pipeWriter.Write(buf[:n]); werr != nil {
								log.L.WithError(werr).Error("Failed to forward stdin")
								break
							}
						}
						if err != nil {
							if err != io.EOF {
								log.L.WithError(err).Error("Error reading from stdin")
							}
							break
						}
					}
					log.L.Debug("Stdin forwarding completed")
				}()

				log.L.Debug("Opened stdin FIFO for reading with logging")
			} else {
				// No logging, use FIFO directly
				sio.stdin = stdinFifo
				log.L.Debug("Opened stdin FIFO for reading")
			}
		}
	}

	// Handle stdout if requested
	if stdout != "" {
		if _, err := os.Stat(stdout); err == nil {
			// Open the FIFO for writing (containerd reads from it, we write to it)
			stdoutFifo, err := fifo.OpenFifo(ctx, stdout, syscall.O_WRONLY, 0)
			if err != nil {
				if sio.stdin != nil {
					sio.stdin.Close()
				}
				if sio.logFile != nil {
					sio.logFile.Close()
				}
				return nil, err
			}

			// If we have a log file, create a writer that adds prefixes
			if sio.logFile != nil {
				sio.stdout = &prefixedWriter{
					w:       stdoutFifo,
					logFile: sio.logFile,
					prefix:  "STDOUT: ",
					closer: func() {
						stdoutFifo.Close()
					},
				}
				log.L.Debug("Opened stdout FIFO for writing with logging")
			} else {
				// Otherwise just use the FIFO directly
				sio.stdout = stdoutFifo
				log.L.Debug("Opened stdout FIFO for writing")
			}
		}
	}

	// Handle stderr if requested
	if stderr != "" {
		if _, err := os.Stat(stderr); err == nil {
			// Open the FIFO for writing (containerd reads from it, we write to it)
			stderrFifo, err := fifo.OpenFifo(ctx, stderr, syscall.O_WRONLY, 0)
			if err != nil {
				if sio.stdin != nil {
					sio.stdin.Close()
				}
				if sio.stdout != nil {
					sio.stdout.Close()
				}
				if sio.logFile != nil {
					sio.logFile.Close()
				}
				return nil, err
			}

			// If we have a log file, create a writer that adds prefixes
			if sio.logFile != nil {
				sio.stderr = &prefixedWriter{
					w:       stderrFifo,
					logFile: sio.logFile,
					prefix:  "STDERR: ",
					closer: func() {
						stderrFifo.Close()
					},
				}
				log.L.Debug("Opened stderr FIFO for writing with logging")
			} else {
				// Otherwise just use the FIFO directly
				sio.stderr = stderrFifo
				log.L.Debug("Opened stderr FIFO for writing")
			}
		}
	}

	return &sio, nil
}

// prefixedWriter is a Writer that writes to the primary writer and logs with a prefix
type prefixedWriter struct {
	w       io.Writer
	logFile *os.File
	prefix  string
	closer  func()
	mu      sync.Mutex
}

func (pw *prefixedWriter) Write(p []byte) (n int, err error) {
	// Write to the primary writer first
	n, err = pw.w.Write(p)
	if err != nil {
		return n, err
	}

	// Log with prefix
	pw.mu.Lock()
	defer pw.mu.Unlock()

	pw.logFile.Write([]byte(pw.prefix))
	pw.logFile.Write(p)
	if len(p) > 0 && p[len(p)-1] != '\n' {
		pw.logFile.Write([]byte("\n"))
	}

	return n, nil
}

func (pw *prefixedWriter) Close() error {
	if pw.closer != nil {
		pw.closer()
	}
	return nil
}

// Close closes all process streams
func (p *ProcessIO) Close() error {
	var err error

	if p.stdin != nil {
		if cerr := p.stdin.Close(); cerr != nil {
			err = cerr
		}
	}

	if p.stdout != nil {
		if cerr := p.stdout.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}

	if p.stderr != nil {
		if cerr := p.stderr.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}

	if p.logFile != nil {
		// Write footer to log file
		fmt.Fprintf(p.logFile, "--- END OF CONTAINER I/O LOG ---\n")
		if cerr := p.logFile.Close(); cerr != nil && err == nil {
			err = cerr
		}
		p.logFile = nil
	}

	return err
}
