package streamexec

import (
	"context"
	"io"
	"log"
	"net"
	"sync"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/streamexec/executor"
	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

// Server represents the server side of the stream execution protocol
type Server struct {
	transport  transport.Transport
	executor   executor.CommandExecutor
	protocol   protocol.ProtocolConnFunc
	bufferSize int
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewServer creates a new Server
func NewServer(ctx context.Context, transport transport.Transport, executor executor.CommandExecutor, proto protocol.ProtocolConnFunc) *Server {

	ctx, cancel := context.WithCancel(ctx)

	return &Server{
		transport: transport,
		executor:  executor,
		protocol:  proto,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// ListenAndServe starts a server on the given listener
func (s *Server) Serve() error {

	listener, err := s.transport.Listen()
	if err != nil {
		return errors.Errorf("listening: %w", err)
	}

	go func() {
		<-s.ctx.Done()
		listener.Close()
	}()

	// Accept connections in a goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if the server is shutting down
				select {
				case <-s.ctx.Done():
					return
				default:
					log.Printf("Error accepting connection: %v", err)
					continue
				}
			}

			// Handle each connection in a separate goroutine
			s.wg.Add(1)
			go func(c net.Conn) {
				defer s.wg.Done()
				defer c.Close()

				if err := s.handleConnection(c); err != nil {
					log.Printf("Error handling connection: %v", err)
				}
			}(conn)
		}
	}()

	s.wg.Wait()

	return nil
}

// handleConnection processes a single client connection
func (s *Server) handleConnection(conn io.ReadWriteCloser) error {
	// Create a protocol handler for this connection
	proto := s.protocol(conn)

	// Process commands until the connection is closed
	for {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			msgType, data, err := proto.ReadMessage()
			if err != nil {
				if err == io.EOF {
					return nil // Connection closed normally
				}
				return errors.Errorf("reading message: %w", err)
			}

			// Only handle command messages
			if msgType != protocol.Command {
				log.Printf("Received non-command message type: %d", msgType)
				continue
			}

			command := string(data)
			log.Printf("Executing command: %s", command)
			cmdCtx, cmdCancel := context.WithCancel(s.ctx)

			go func() {
				defer cmdCancel()

				// Execute the command in a separate context that can be canceled when the server shuts down
				err := s.executor.ExecuteCommand(cmdCtx, proto, command)

				if err != nil {
					log.Printf("Error executing command: %v", err)
					// Continue processing commands despite errors
				}
			}()

		}
	}
}

// HandleSingleCommand handles a single command from the connection and then closes it
func (s *Server) HandleSingleCommand(conn io.ReadWriteCloser) error {
	// Create a protocol handler for this connection
	proto := s.protocol(conn)
	defer conn.Close()

	msgType, data, err := proto.ReadMessage()
	if err != nil {
		return errors.Errorf("reading message: %w", err)
	}

	// Only handle command messages
	if msgType != protocol.Command {
		return errors.Errorf("expected command message, got type: %d", msgType)
	}

	command := string(data)
	log.Printf("Executing command: %s", command)

	// Execute the command
	cmdCtx, cmdCancel := context.WithCancel(s.ctx)
	defer cmdCancel()

	return s.executor.ExecuteCommand(cmdCtx, proto, command)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	// Signal all goroutines to stop
	s.cancel()

	// Wait for all connections to finish with a timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
