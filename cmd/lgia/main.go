package main

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

const (
	vsockPort = 2019
	// vsockExecPath = "/usr/local/bin/vsock_exec"
	realInitPath = "/init.real"
)

func main() {

	log.Printf("Starting lgia")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logging.SetupSlogSimpleNoColor(ctx)

	pid := os.Getpid()
	if pid == 1 {

		// make an ec1 directory
		err := os.Mkdir("ec1", 0755)
		if err != nil {
			log.Fatalf("Failed to create ec1 directory: %v", err)
		}

		// // make a pid file, stdout and
		// pidFile, err := os.Create("ec1/init.pid")
		// if err != nil {
		// 	log.Fatalf("Failed to create pid file: %v", err)
		// }
		// handleFile, err := os.Create("ec1/init.handle")
		// if err != nil {
		// 	log.Fatalf("Failed to create handle file: %v", err)
		// }
		// stdout, err := os.Create("ec1/init.stdout")
		// if err != nil {
		// 	log.Fatalf("Failed to create stdout file: %v", err)
		// }
		// stderr, err := os.Create("ec1/init.stderr")
		// if err != nil {
		// 	log.Fatalf("Failed to create stderr file: %v", err)
		// }

		pid, hf, err := syscall.StartProcess(os.Args[0], os.Args[1:], &syscall.ProcAttr{
			Env:   os.Environ(),
			Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		})
		if err != nil {
			log.Fatalf("Failed to start process: %v", err)
		}

		log.Printf("Starting lgia at %s, pid %d, hf %d", os.Args[0], pid, hf)

		// pidFile.WriteString(strconv.Itoa(pid))
		// pidFile.Close()

		// handleFile.WriteString(strconv.Itoa(int(h)))
		// handleFile.Close()

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
			log.Printf("Failed to serve vsock: %v", err)

		}

		log.Printf("Shutting down lgia")
	}

}

func serveRawVsock(ctx context.Context, port int) error {

	// // wait until rootfs is mounted
	// for {
	// 	if _, err := os.Stat("/init.ec1"); err == nil {
	// 		log.Printf("Rootfs is mounted, continuing")
	// 		break
	// 	}
	// 	time.Sleep(1 * time.Second)
	// }

	os.Mkdir("/ec1", 0755)

	// // make a pid file, stdout and
	pidFile, err := os.Create("/ec1/init.pid")
	if err != nil {
		log.Fatalf("Failed to create pid file: %v", err)
	}

	// stdout, err := os.Create("/ec1/init.stdout")
	// if err != nil {
	// 	log.Fatalf("Failed to create stdout file: %v", err)
	// }
	// stderr, err := os.Create("/ec1/init.stderr")
	// if err != nil {
	// 	log.Fatalf("Failed to create stderr file: %v", err)
	// }

	// os.Stdout = stdout
	// os.Stderr = stderr

	// defer func() {
	// 	stdout.Close()
	// 	stderr.Close()
	// }()

	pidFile.WriteString(strconv.Itoa(os.Getpid()))
	pidFile.Close()

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutor(1024)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}
