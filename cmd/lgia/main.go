package main

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

const (
	vsockPort = 2019
	// vsockExecPath = "/usr/local/bin/vsock_exec"
	realInitPath = "/sbin/init.real"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pid := os.Getpid()
	if pid == 1 {

		// make an ec1 directory
		err := os.Mkdir("ec1", 0755)
		if err != nil {
			log.Fatalf("Failed to create ec1 directory: %v", err)
		}

		// make a pid file, stdout and
		pidFile, err := os.Create("ec1/init.pid")
		if err != nil {
			log.Fatalf("Failed to create pid file: %v", err)
		}
		handleFile, err := os.Create("ec1/init.handle")
		if err != nil {
			log.Fatalf("Failed to create handle file: %v", err)
		}
		stdout, err := os.Create("ec1/init.stdout")
		if err != nil {
			log.Fatalf("Failed to create stdout file: %v", err)
		}
		stderr, err := os.Create("ec1/init.stderr")
		if err != nil {
			log.Fatalf("Failed to create stderr file: %v", err)
		}

		pid, h, err := syscall.StartProcess(os.Args[0], os.Args[1:], &syscall.ProcAttr{
			Env:   os.Environ(),
			Files: []uintptr{os.Stdin.Fd(), stdout.Fd(), stderr.Fd()},
		})
		if err != nil {
			log.Fatalf("Failed to start process: %v", err)
		}

		pidFile.WriteString(strconv.Itoa(pid))
		pidFile.Close()

		handleFile.WriteString(strconv.Itoa(int(h)))
		handleFile.Close()

		if err := syscall.Exec(realInitPath, os.Args[1:], os.Environ()); err != nil {
			log.Fatalf("Failed to exec original init: %v", err)
		}
	} else {

		log.Printf("Serving vsock on port %d, pid %d", vsockPort, pid)

		defer func() {
			log.Printf("Shutting down vsock server")
		}()

		err := serveRawVsock(ctx, vsockPort)
		if err != nil {
			log.Fatalf("Failed to serve vsock: %v", err)
		}
	}

}

func serveRawVsock(ctx context.Context, port int) error {

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutor(1024)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}
