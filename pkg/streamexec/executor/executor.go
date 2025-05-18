// Package executor defines interfaces and implementations for command execution
package executor

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/pkg/streamexec/protocol"
)

// CommandExecutor is the interface for executing commands
type CommandExecutor interface {
	// ExecuteCommand executes a command and streams stdin/stdout/stderr through the protocol
	ExecuteCommand(ctx context.Context, proto protocol.Protocol, command string) error
}

// StreamingExecutor implements CommandExecutor using a Protocol for streaming
type StreamingExecutor struct {
	bufferSize int
}

// NewStreamingExecutor creates a new StreamingExecutor
func NewStreamingExecutor(bufferSize int) *StreamingExecutor {
	return &StreamingExecutor{
		bufferSize: bufferSize,
	}
}

// ExecuteCommand executes a command and streams stdin/stdout/stderr through the protocol
func (e *StreamingExecutor) ExecuteCommand(ctx context.Context, proto protocol.Protocol, command string) error {
	// Split the command into parts
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New("empty command")
	}

	// First try direct execution
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	err := e.setupAndRunCommand(ctx, proto, cmd)

	// If direct execution failed, try with shell
	if err != nil {
		log.Printf("Direct execution failed, trying with shell: %v", err)
		cmd = exec.CommandContext(ctx, "/bin/sh", "-c", command)

		return e.setupAndRunCommand(ctx, proto, cmd)
	}

	return nil
}

// setupAndRunCommand sets up pipes for a command and runs it
func (e *StreamingExecutor) setupAndRunCommand(ctx context.Context, proto protocol.Protocol, cmd *exec.Cmd) error {
	// Create pipes for stdout, stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Errorf("creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.Errorf("creating stderr pipe: %w", err)
	}

	// TODO: We could add stdin support by reading Stdin messages from the protocol
	// and writing them to the stdin pipe

	// Start the command
	if err := cmd.Start(); err != nil {
		return errors.Errorf("starting command: %w", err)
	}

	// Create a wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(2) // for stdout and stderr goroutines

	// Stream stdout to the protocol
	go func() {
		defer wg.Done()
		e.streamOutput(stdout, protocol.Stdout, proto)
	}()

	// Stream stderr to the protocol
	go func() {
		defer wg.Done()
		e.streamOutput(stderr, protocol.Stderr, proto)
	}()

	// Wait for the command to complete
	cmdErr := cmd.Wait()

	// Wait for stdout/stderr goroutines to finish
	wg.Wait()

	// Send exit status
	var exitMsg string
	if cmdErr != nil {
		exitMsg = fmt.Sprintf("Command exited with error: %v", cmdErr)
		proto.WriteMessage(protocol.Exit, []byte(exitMsg))
		return errors.Errorf("command execution error: %w", cmdErr)
	}

	exitMsg = "Command completed successfully"
	proto.WriteMessage(protocol.Exit, []byte(exitMsg))
	return nil
}

// streamOutput reads from a reader and sends the data through the protocol
func (e *StreamingExecutor) streamOutput(r io.Reader, msgType protocol.MessageType, proto protocol.Protocol) {
	buf := make([]byte, e.bufferSize)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if err := proto.WriteMessage(msgType, buf[:n]); err != nil {
				log.Printf("Error sending output: %v", err)
				break
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading output: %v", err)
			}
			break
		}
	}
}
