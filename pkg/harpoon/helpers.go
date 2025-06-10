package harpoon

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gitlab.com/tozd/go/errors"
	"go.bug.st/serial"
)

func ptr[T any](v T) *T { return &v }

func ExecCmdForwardingStdio(ctx context.Context, cmds ...string) error {
	if len(cmds) == 0 {
		return errors.Errorf("no command to execute")
	}

	argc := "/bin/busybox"
	if strings.HasPrefix(cmds[0], "/") {
		argc = cmds[0]
		cmds = cmds[1:]
	}
	argv := cmds

	slog.DebugContext(ctx, "executing command", "argc", argc, "argv", argv)
	cmd := exec.CommandContext(ctx, argc, argv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Cloneflags: syscall.CLONE_NEWNS,
	}

	path := os.Getenv("PATH")

	cmd.Env = append([]string{"PATH=" + path + ":/hbin"}, os.Environ()...)

	cmd.Stdin = bytes.NewBuffer(nil) // set to avoid reading /dev/null since it may not be mounted
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return errors.Errorf("running busybox command (stdio was copied to the parent process): %v: %w", cmds, err)
	}

	return nil
}

func OpenSerialPort(ctx context.Context, portName string) (io.ReadWriteCloser, error) {

	// List available ports to find your virtio console
	ports, err := serial.GetPortsList()
	if err != nil {
		return nil, errors.Errorf("getting ports list: %w", err)
	}

	fmt.Println("Available ports:")
	for _, port := range ports {
		fmt.Printf("  %s\n", port)
	}

	mode := &serial.Mode{
		BaudRate: 115200, // Virtio consoles typically use high baud rates
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, errors.Errorf("opening port %s: %w", portName, err)
	}

	return port, nil
}
