//go:build linux

package main

import (
	"context"
	"io"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/mdlayher/vsock"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

const (
	vsockPort = 2019
	// vsockExecPath = "/usr/local/bin/vsock_exec"
	realInitPath = "/iniz"
)

func main() {

	log.Printf("lgia wrapper called with cmdline=%s args: %v", os.Args[0], os.Args[1:])

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = logging.SetupSlogSimpleNoColor(ctx)

	pid := os.Getpid()
	log.Printf("Starting lgia as pid %d", pid)

	if pid == 1 {

		// make an ec1 directory
		err := os.Mkdir("ec1", 0755)
		if err != nil {
			log.Fatalf("Failed to create ec1 directory: %v", err)
		}

		pid, hf, err := syscall.StartProcess(os.Args[0], os.Args[1:], &syscall.ProcAttr{
			Env:   os.Environ(),
			Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
		})
		if err != nil {
			log.Fatalf("Failed to start process: %v", err)
		}

		log.Printf("Starting lgia at %s, pid %d, hf %d", os.Args[0], pid, hf)

		// // check if realInitPath exists
		// if _, err := os.Stat(realInitPath); os.IsNotExist(err) {
		// 	// read the command line

		// 	// parse the kernel command line, get the rootfs path
		// 	// kernelCmdLine, err := os.ReadFile("/proc/cmdline")
		// 	// if err != nil {
		// 	// 	log.Fatalf("Failed to read kernel command line: %v", err)
		// 	// }
		// 	kernelCmdLineStr := strings.Join(os.Args[1:], " ")
		// 	kernelCmdLineStr = strings.TrimSpace(kernelCmdLineStr)

		// 	// parse the kernel command line, get the rootfs path
		// 	rootfsPath := "/dev/nvme0n1p1"
		// 	initCommand := "/sbin/init"
		// 	for _, arg := range strings.Split(kernelCmdLineStr, " ") {
		// 		if strings.HasPrefix(arg, "root=") {
		// 			rootfsPath = arg[6:]
		// 			break
		// 		}
		// 		if strings.HasPrefix(arg, "init=") {
		// 			initCommand = arg[5:]
		// 			break
		// 		}
		// 	}

		// 	// 1) Mount real root
		// 	err = syscall.Mount(rootfsPath, rootfsPath, "", syscall.MS_BIND|syscall.MS_REC, "")
		// 	if err != nil {
		// 		log.Fatalf("mount failed: %v", err)
		// 	}
		// 	// 2) pivot_root into new root
		// 	if err := syscall.PivotRoot(rootfsPath, rootfsPath+"/.oldroot"); err != nil {
		// 		log.Fatalf("pivot_root failed: %v", err)
		// 	}
		// 	// 3) Change working directory and unmount old root
		// 	err = syscall.Chdir("/")
		// 	if err != nil {
		// 		log.Fatalf("chdir failed: %v", err)
		// 	}
		// 	err = syscall.Unmount("/.oldroot", syscall.MNT_DETACH)
		// 	if err != nil {
		// 		log.Fatalf("unmount failed: %v", err)
		// 	}
		// 	// 4) Exec the real init
		// 	if err := syscall.Exec(initCommand, []string{initCommand}, os.Environ()); err != nil {
		// 		log.Fatalf("Failed to exec real init: %v", err)
		// 	}

		// 	// we need to do the switch to rootfs
		// 	// command := []string{rootfsPath, initCommand}
		// 	// log.Printf("Switching to rootfs: %s, initCommand: %s", rootfsPath, initCommand)
		// 	// if err := syscall.Exec("switch_root", command, os.Environ()); err != nil {
		// 	// 	log.Fatalf("Failed to exec switch_root: %v", err)
		// 	// }
		// }

		// pidFile.WriteString(strconv.Itoa(pid))
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

	// wait until /dev/vsock exists
	// for {
	// 	if _, err := os.Stat("/dev/vsock"); err == nil {
	// 		break
	// 	}
	// 	log.Printf("Waiting for /dev/vsock to exist")
	// 	time.Sleep(1 * time.Second)
	// }

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

	// just keep printing a message
	go func() {
		count := 0
		for {
			count++
			// check if /dev/vsock exists
			l, err := vsock.ListenContextID(3, uint32(2020+count), nil)
			if err != nil {
				log.Printf("Error listening on vsock: %v", err)
			} else {
				log.Printf("Listening on vsock")

			}

			log.Printf("Waiting for z to be ready (listenErr=%v)", err)
			time.Sleep(100 * time.Millisecond)

			if l != nil {
				l.Close()
			}
		}
	}()

	pidFile.WriteString(strconv.Itoa(os.Getpid()))
	pidFile.Close()

	tranport := transport.NewVSockTransport(0, uint32(port))
	executor := executor.NewStreamingExecutor(1024)
	server := streamexec.NewServer(ctx, tranport, executor, func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	})

	return server.Serve()
}
