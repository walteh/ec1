package containerd

import (
	"context"
	"io"
	"log/slog"
	"os"
	"syscall"

	"github.com/containerd/fifo"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/hack"
)

type stdio struct {
	stdinPath  string
	stdoutPath string
	stderrPath string

	stdinFD  int
	stdoutFD int
	stderrFD int

	stdin  io.ReadCloser
	stdout io.WriteCloser
	stderr io.WriteCloser
}

func openFifo(ctx context.Context, path string, flags int) (uintptr, io.ReadWriteCloser, error) {

	if _, err := os.Stat(path); err != nil {
		return 0, nil, errors.Errorf("stat: %w", err)
	}

	fo, err := fifo.OpenFifo(ctx, path, flags, 0)
	if err != nil {
		return 0, nil, err
	}

	fd := hack.GetUnexportedFieldOf(fo, "file").(*os.File).Fd()
	// sc, ok := fo.(syscall.Conn)
	// if !ok {
	// 	return 0, nil, errors.Errorf("fifo is not a syscall.Conn")
	// }

	// slog.InfoContext(ctx, "fifo opened, starting syscall.Conn", "path", path)

	// rc, err := sc.SyscallConn()
	// if err != nil {
	// 	return 0, nil, errors.Errorf("getting syscall.Conn: %w", err)
	// }

	// slog.InfoContext(ctx, "fifo opened, got syscall.Conn - getting fd", "path", path)

	// var fdr uintptr

	// err = rc.Control(func(fd uintptr) {
	// 	fdr = fd
	// })
	// if err != nil {
	// 	return fdr, nil, errors.Errorf("setting fd: %w", err)
	// }

	slog.InfoContext(ctx, "fifo opened, got fd", "path", path)

	return fd, fo, nil
}

func setupIO(ctx context.Context, stdin, stdout, stderr string) (io stdio, _ error) {
	io.stdinPath = stdin
	io.stdoutPath = stdout
	io.stderrPath = stderr

	if stdin != "" {

		fd, fo, err := openFifo(ctx, stdin, syscall.O_RDONLY|syscall.O_NONBLOCK)
		if err != nil {
			return io, errors.Errorf("opening stdin fifo: %w", err)
		}
		io.stdinFD = int(fd)
		io.stdin = fo
	}

	if stdout != "" {
		fd, fo, err := openFifo(ctx, stdout, syscall.O_WRONLY)
		if err != nil {
			return io, errors.Errorf("opening stdout fifo: %w", err)
		}
		io.stdoutFD = int(fd)
		io.stdout = fo
	}
	if stderr != "" {
		fd, fo, err := openFifo(ctx, stderr, syscall.O_WRONLY)
		if err != nil {
			return io, errors.Errorf("opening stderr fifo: %w", err)
		}
		io.stderrFD = int(fd)
		io.stderr = fo
	}

	return io, nil
}

func (s stdio) Close() error {
	if s.stdin != nil {
		_ = s.stdin.Close()
	}
	if s.stdout != nil {
		_ = s.stdout.Close()
	}
	if s.stderr != nil {
		_ = s.stderr.Close()
	}
	return nil
}
