package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/walteh/ec1/pkg/streamexec"
	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

const (
	// VMADDR constants from linux/vm_sockets.h
	VMADDR_CID_ANY        = 0xFFFFFFFF
	VMADDR_CID_HYPERVISOR = 0
	VMADDR_CID_LOCAL      = 1
	VMADDR_CID_HOST       = 2
	VMADDR_CID_GUEST      = 3
	// Default message buffer size
	DEFAULT_BUFFER_SIZE = 2048
)

func main() {
	// Default values
	defaultCID := VMADDR_CID_HOST
	defaultPort := 1024

	// Parse command line arguments
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [CID[:PORT]]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  CID:       Context ID (defaults to %d/VMADDR_CID_HOST)\n", defaultCID)
		fmt.Fprintf(os.Stderr, "  PORT:      Port number (defaults to %d)\n", defaultPort)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s                 # Listen on host on port 1024\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 3               # Listen on CID 3 on port 1024\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s 3:2048          # Listen on CID 3 on port 2048\n", os.Args[0])
	}
	flag.Parse()

	// Process arguments
	cid := defaultCID
	port := defaultPort

	args := flag.Args()
	if len(args) > 0 {
		param := args[0]
		if strings.Contains(param, ":") {
			parts := strings.Split(param, ":")
			cidStr, portStr := parts[0], parts[1]

			if c, err := strconv.Atoi(cidStr); err == nil {
				if isValidCID(uint32(c)) {
					cid = c
				}
			}

			if p, err := strconv.Atoi(portStr); err == nil {
				if p >= 0 && p <= 65535 {
					port = p
				}
			}
		} else {
			if c, err := strconv.Atoi(param); err == nil {
				if isValidCID(uint32(c)) {
					cid = c
				}
			}
		}
	}

	// Create transport
	transportObj := transport.NewVSockTransport(uint32(cid), uint32(port))

	// Create executor
	exec := executor.NewStreamingExecutor(DEFAULT_BUFFER_SIZE)

	// Create protocol function
	protoFunc := func(conn io.ReadWriter) protocol.Protocol {
		return protocol.NewFramedProtocol(conn)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create server
	server := streamexec.NewServer(ctx, transportObj, exec, protoFunc)

	// Start the server
	if err := server.Serve(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	fmt.Printf("Server listening on CID %d, port %d\n", cid, port)

	// Wait for interrupt signal to gracefully shut down the server
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down server...")

	// Give 5 seconds for shutdown - don't use the original context because its likely already cancelled
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	fmt.Println("Server stopped")
}

// isValidCID checks if the provided CID is one of the predefined values
func isValidCID(cid uint32) bool {
	return cid == VMADDR_CID_ANY || cid == VMADDR_CID_HYPERVISOR ||
		cid == VMADDR_CID_LOCAL || cid == VMADDR_CID_HOST || cid > 2
}
