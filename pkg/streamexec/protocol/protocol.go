// Package protocol defines interfaces and implementations for command streaming protocols
package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	"gitlab.com/tozd/go/errors"
)

// MessageType represents different types of messages in the protocol
type MessageType byte

const (
	// Command represents a command to be executed
	Command MessageType = iota
	// Stdout represents output from a command's stdout
	Stdout
	// Stderr represents output from a command's stderr
	Stderr
	// Stdin represents input for a command's stdin
	Stdin
	// Exit represents a command exit status
	Exit
)

func (m MessageType) String() string {
	switch m {
	case Command:
		return "command"
	case Stdout:
		return "stdout"
	case Stderr:
		return "stderr"
	case Stdin:
		return "stdin"
	case Exit:
		return "exit"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

type ProtocolConnFunc func(conn io.ReadWriter) Protocol

// Protocol handles framing and sending/receiving messages
type Protocol interface {
	// ReadMessage reads a message from the connection
	ReadMessage() (MessageType, []byte, error)

	// WriteMessage writes a message to the connection
	WriteMessage(msgType MessageType, data []byte) error

	// Close closes the underlying connection
	Close() error
}

// FramedProtocol implements the Protocol interface with a simple framing protocol
// where each message is prefixed with a type byte and length (2 bytes)
type FramedProtocol struct {
	conn io.ReadWriter
}

// HeaderSize is the size of the message header (1 byte type + 2 bytes length)
const HeaderSize = 3

// NewFramedProtocol creates a new FramedProtocol
func NewFramedProtocol(conn io.ReadWriter) *FramedProtocol {
	return &FramedProtocol{
		conn: conn,
	}
}

// ReadMessage reads a message from the connection
func (p *FramedProtocol) ReadMessage() (MessageType, []byte, error) {
	// Read header (type + length)
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(p.conn, header); err != nil {
		return 0, nil, errors.Errorf("reading message header: %w", err)
	}

	// Parse header
	msgType := MessageType(header[0])
	msgLength := binary.LittleEndian.Uint16(header[1:3])

	// Read message body
	data := make([]byte, msgLength)
	if _, err := io.ReadFull(p.conn, data); err != nil {
		return 0, nil, errors.Errorf("reading message body: %w", err)
	}

	return msgType, data, nil
}

// WriteMessage writes a message to the connection
func (p *FramedProtocol) WriteMessage(msgType MessageType, data []byte) error {
	// Ensure message length fits in uint16
	if len(data) > 65535 {
		return errors.Errorf("message too large: %d bytes", len(data))
	}

	// Create header
	header := make([]byte, HeaderSize)
	header[0] = byte(msgType)
	binary.LittleEndian.PutUint16(header[1:3], uint16(len(data)))

	// Write header
	if _, err := p.conn.Write(header); err != nil {
		return errors.Errorf("writing message header: %w", err)
	}

	// Write data
	if _, err := p.conn.Write(data); err != nil {
		return errors.Errorf("writing message body: %w", err)
	}

	return nil
}

// Close closes the underlying connection if it implements io.Closer
func (p *FramedProtocol) Close() error {
	if closer, ok := p.conn.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
