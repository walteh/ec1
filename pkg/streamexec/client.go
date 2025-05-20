package streamexec

import (
	"context"
	"io"
	"log"
	"sync"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/streamexec/transport"
)

// Client represents the client side of the stream execution protocol
type Client struct {
	transport transport.Transport
	protocol  protocol.ProtocolConnFunc
	conn      io.ReadWriteCloser
	mu        sync.Mutex
}

// NewClient creates a new Client with the given transport
func NewClient(transport transport.Transport, protocol protocol.ProtocolConnFunc) *Client {
	return &Client{
		transport: transport,
		protocol:  protocol,
	}
}

// Connect establishes a connection to the server
func (c *Client) Connect(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}

	var err error
	c.conn, err = c.transport.Dial()
	if err != nil {
		return errors.Errorf("dialing transport: %w", err)
	}

	// c.proto = c.protocolBuilder(c.conn)
	return nil
}

// Close closes the connection to the server
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SendCommand sends a command to the server
func (c *Client) SendCommand(ctx context.Context, command string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("not connected")
	}

	conn := c.protocol(c.conn)

	return conn.WriteMessage(protocol.Command, []byte(command))
}

// ReadOutput reads output from the server until the connection is closed or an error occurs
// It calls the provided callbacks for each message type
func (c *Client) ReadOutput(
	ctx context.Context,
	onStdout func([]byte),
	onStderr func([]byte),
	onExit func([]byte),
) error {
	if c.conn == nil {
		return errors.New("not connected")
	}

	conn := c.protocol(c.conn)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return errors.Errorf("reading message: %w", err)
			}

			switch msgType {
			case protocol.Stdout:
				if onStdout != nil {
					onStdout(data)
				}
			case protocol.Stderr:
				if onStderr != nil {
					onStderr(data)
				}
			case protocol.Exit:
				if onExit != nil {
					onExit(data)
				}
				return nil // Exit message indicates the command is done
			default:
				log.Printf("Received unknown message type: %d", msgType)
			}
		}
	}
}

// ExecuteCommand is a convenience method that sends a command and waits for all output
// It returns the stdout, stderr, and exit message
func (c *Client) ExecuteCommand(ctx context.Context, command string) (stdout, stderr, exit []byte, err error) {
	var stdoutBuf, stderrBuf, exitBuf []byte

	err = c.SendCommand(ctx, command)
	if err != nil {
		return nil, nil, nil, errors.Errorf("sending command: %w", err)
	}

	err = c.ReadOutput(
		ctx,
		func(data []byte) { stdoutBuf = append(stdoutBuf, data...) },
		func(data []byte) { stderrBuf = append(stderrBuf, data...) },
		func(data []byte) { exitBuf = append(exitBuf, data...) },
	)

	return stdoutBuf, stderrBuf, exitBuf, err
}

// ExecuteCommandWithStreams is a convenience method that sends a command and streams the output
func (c *Client) ExecuteCommandWithStreams(ctx context.Context, command string, stdout, stderr io.Writer) error {
	err := c.SendCommand(ctx, command)
	if err != nil {
		return errors.Errorf("sending command: %w", err)
	}

	return c.ReadOutput(
		ctx,
		func(data []byte) {
			if stdout != nil {
				stdout.Write(data)
			}
		},
		func(data []byte) {
			if stderr != nil {
				stderr.Write(data)
			}
		},
		func(data []byte) {
			// Exit message - we could log this if needed
			log.Printf("Command exited: %s", string(data))
		},
	)
}
