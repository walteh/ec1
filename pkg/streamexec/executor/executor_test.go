package executor

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/walteh/ec1/pkg/streamexec/protocol"
	"github.com/walteh/ec1/pkg/testutils"
)

func TestStreamingExecutor_ExecuteCommand(t *testing.T) {

	tests := []struct {
		name           string
		command        string
		expectError    bool
		expectStdout   string
		expectStderr   string
		expectExitCode bool
	}{
		{
			name:           "Echo command",
			command:        "echo hello world",
			expectStdout:   "hello world",
			expectExitCode: true,
		},
		{
			name:           "Command with stderr",
			command:        "echo error message >&2",
			expectStderr:   "error message",
			expectExitCode: true,
		},
		{
			name:           "Command not found",
			command:        "nonexistentcommand",
			expectError:    true,
			expectStderr:   "not found",
			expectExitCode: true,
		},
		{
			name:        "Empty command",
			command:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			ctx := t.Context()

			ctx = testutils.SetupSlog(t, ctx)

			// Create mock protocol
			proto := protocol.NewMockProtocol()

			// Create executor
			executor := NewStreamingExecutor(1024)

			// Execute command
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := executor.ExecuteCommand(ctx, proto, tt.command)

			// Check error
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check stdout
			var foundStdout bool
			if tt.expectStdout != "" {
				for _, msg := range proto.ReceivedMessages {
					if msg.Type == protocol.Stdout && bytes.Contains(msg.Data, []byte(tt.expectStdout)) {
						foundStdout = true
						break
					}
				}
				if !foundStdout {
					t.Errorf("Expected stdout to contain %q, but it didn't", tt.expectStdout)
				}
			}

			// Check stderr
			var foundStderr bool
			if tt.expectStderr != "" {
				for _, msg := range proto.ReceivedMessages {
					if msg.Type == protocol.Stderr && bytes.Contains(msg.Data, []byte(tt.expectStderr)) {
						foundStderr = true
						break
					}
				}
				if !foundStderr {
					t.Errorf("Expected stderr to contain %q, but it didn't", tt.expectStderr)
				}
			}

			// Check for exit message
			var foundExit bool
			if tt.expectExitCode {
				for _, msg := range proto.ReceivedMessages {
					if msg.Type == protocol.Exit {
						foundExit = true
						break
					}
				}
				if !foundExit {
					t.Errorf("Expected exit code message, but it wasn't sent")
				}
			}
		})
	}
}
