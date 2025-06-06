package containerd

import (
	"context"
	"io"
	"os"
	"syscall"

	"github.com/containerd/fifo"
	"gitlab.com/tozd/go/errors"
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
	sc, ok := fo.(syscall.Conn)
	if !ok {
		return 0, nil, errors.Errorf("fifo is not a syscall.Conn")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return 0, nil, errors.Errorf("getting syscall.Conn: %w", err)
	}

	var fdr uintptr

	err = rc.Control(func(fd uintptr) {
		fdr = fd
	})
	if err != nil {
		return fdr, nil, errors.Errorf("setting fd: %w", err)
	}

	return fdr, fo, nil
}

func setupIO(ctx context.Context, stdin, stdout, stderr string) (io stdio, _ error) {
	io.stdinPath = stdin
	io.stdoutPath = stdout
	io.stderrPath = stderr

	fd, fo, err := openFifo(ctx, stdin, syscall.O_RDONLY|syscall.O_NONBLOCK)
	if err != nil {
		return io, err
	}
	io.stdinFD = int(fd)
	io.stdin = fo

	fd, fo, err = openFifo(ctx, stdout, syscall.O_WRONLY)
	if err != nil {
		return io, err
	}
	io.stdoutFD = int(fd)
	io.stdout = fo

	fd, fo, err = openFifo(ctx, stderr, syscall.O_WRONLY)
	if err != nil {
		return io, err
	}
	io.stderrFD = int(fd)
	io.stderr = fo

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
