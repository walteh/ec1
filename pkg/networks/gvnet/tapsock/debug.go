package tapsock

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

// debugLogConn is a net.Conn wrapper that logs all reads and writes
type debugLogConn struct {
	net.Conn
	name string
	ctx  context.Context
}

func (d *debugLogConn) Read(b []byte) (n int, err error) {
	n, err = d.Conn.Read(b)
	if err != nil && err != io.EOF {
		slog.ErrorContext(d.ctx, "debugLogConn read error", "name", d.name, "error", err)
	} else if n > 0 {
		slog.InfoContext(d.ctx, "debugLogConn read data", "name", d.name, "bytes", n, "data", b[:n])
	}
	return
}

func (d *debugLogConn) Write(b []byte) (n int, err error) {
	slog.InfoContext(d.ctx, "debugLogConn write attempt", "name", d.name, "bytes", len(b))
	n, err = d.Conn.Write(b)
	if err != nil {
		slog.ErrorContext(d.ctx, "debugLogConn write error", "name", d.name, "error", err)
	} else {
		slog.InfoContext(d.ctx, "debugLogConn wrote data", "name", d.name, "bytes", n)
	}
	return
}

// testNetConnDirectly sends a simple test message through the connection
func testNetConnDirectly(ctx context.Context, conn net.Conn) {
	testData := []byte("EC1_DIRECT_NETCONN_TEST")
	slog.InfoContext(ctx, "testing direct netConn writing before Accept")

	n, err := conn.Write(testData)
	if err != nil {
		slog.ErrorContext(ctx, "failed to write test data to netConn",
			"error", err,
			"error_type", fmt.Sprintf("%T", err))
	} else {
		slog.InfoContext(ctx, "wrote test data to netConn", "bytes", n)
	}
}

// sendTestPackets sends test packets in both directions to verify socket connectivity
func sendTestPackets(ctx context.Context, hostConn, vmConn *net.UnixConn) {
	// Wait a bit for everything to initialize
	time.Sleep(500 * time.Millisecond)

	// Create a simple Ethernet frame (14 byte header + payload)
	// Destination MAC: ff:ff:ff:ff:ff:ff (broadcast)
	// Source MAC: 00:00:00:00:00:01 (made-up source)
	// EtherType: 0x0800 (IPv4)
	// Then some dummy payload
	ethernetHeader := []byte{
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, // Destination MAC: broadcast
		0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // Source MAC: made-up
		0x08, 0x00, // EtherType: IPv4
	}

	// Create test packets of different sizes
	smallTestPayload := []byte("EC1_TEST_PACKET_FOR_SOCKET_DEBUGGING")
	largeTestPayload := make([]byte, 4096) // 4KB payload
	for i := range largeTestPayload {
		largeTestPayload[i] = byte(i % 256) // Fill with pattern
	}

	// Copy SSH header pattern for testing
	sshTestPayload := []byte("SSH-2.0-OpenSSH_8.9 EC1_TEST_SSH_PACKET")

	// Combine header with payloads
	smallTestPacket := append(ethernetHeader, smallTestPayload...)
	largeTestPacket := append(ethernetHeader, largeTestPayload...)
	sshTestPacket := append(ethernetHeader, sshTestPayload...)

	slog.InfoContext(ctx, "created test packets",
		"small_size", len(smallTestPacket),
		"large_size", len(largeTestPacket),
		"ssh_size", len(sshTestPacket))

	// Set up a ticker to send packets every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send initial test packets
	sendTestPacketBatch(ctx, hostConn, vmConn, smallTestPacket, largeTestPacket, sshTestPacket)

	// Keep sending test packets at regular intervals
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stopping test packet sender due to context cancellation")
			return
		case <-ticker.C:
			sendTestPacketBatch(ctx, hostConn, vmConn, smallTestPacket, largeTestPacket, sshTestPacket)
		}
	}
}

// sendTestPacketBatch sends a batch of test packets in both directions
func sendTestPacketBatch(ctx context.Context, hostConn, vmConn *net.UnixConn, smallPacket, largePacket, sshPacket []byte) {
	sendPacket := func(conn *net.UnixConn, packet []byte, direction, packetType string) {
		slog.InfoContext(ctx, "sending test packet",
			"direction", direction,
			"type", packetType,
			"size", len(packet))

		// Set a write deadline
		err := conn.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		if err != nil {
			slog.ErrorContext(ctx, "failed to set write deadline",
				"direction", direction,
				"error", err)
			return
		}

		n, err := conn.Write(packet)

		// Clear the deadline
		conn.SetWriteDeadline(time.Time{})

		if err != nil {
			slog.ErrorContext(ctx, "failed to send test packet",
				"direction", direction,
				"type", packetType,
				"error", err)
		} else {
			slog.InfoContext(ctx, "sent test packet successfully",
				"direction", direction,
				"type", packetType,
				"bytes", n)
		}
	}

	// Host to VM direction
	sendPacket(hostConn, smallPacket, "host->vm", "small")
	time.Sleep(100 * time.Millisecond)
	sendPacket(hostConn, sshPacket, "host->vm", "ssh")
	time.Sleep(100 * time.Millisecond)
	sendPacket(hostConn, largePacket, "host->vm", "large")

	time.Sleep(500 * time.Millisecond)

	// VM to Host direction
	sendPacket(vmConn, smallPacket, "vm->host", "small")
	time.Sleep(100 * time.Millisecond)
	sendPacket(vmConn, sshPacket, "vm->host", "ssh")
	time.Sleep(100 * time.Millisecond)
	sendPacket(vmConn, largePacket, "vm->host", "large")
}

func unixConnLogProxy(ctx context.Context, name string, from *net.UnixConn, to *net.UnixConn) (int, error) {
	slog.InfoContext(ctx, "=== PROXY STARTED ===", "name", name,
		"from", from.LocalAddr().String(),
		"to", to.LocalAddr().String())

	var totalBytes int
	var readAttempts int
	var writeAttempts int
	var readSuccess int
	var writeSuccess int

	// Create a buffer with a reasonable size for network packets
	// SSH packets can be larger, so increase the buffer size
	buf := make([]byte, 65536) // 64KB buffer for better performance

	for {
		readAttempts++

		// Check if context is done before each read
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "Context done in proxy", "name", name)
			return totalBytes, nil
		default:
			// Continue with the normal flow
		}

		// Set a shorter read deadline to prevent blocking for too long
		readDeadlineErr := from.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		if readDeadlineErr != nil {
			if isClosedConnError(readDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONNECTION CLOSED) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to set read deadline", "name", name, "error", readDeadlineErr)
		}

		// Add blocking indicator so we know if we're stuck on read
		slog.DebugContext(ctx, "waiting to read data", "name", name, "attempt", readAttempts)

		n, err := from.Read(buf)

		// Very detailed error logging
		if err != nil {
			errStr := fmt.Sprintf("%v", err)
			errType := fmt.Sprintf("%T", err)

			// Handle timeout by continuing
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Just a timeout, not an error - continue with the loop
				continue
			}

			// Handle closed connection gracefully
			if isClosedConnError(err) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED DURING READ) ===",
					"name", name,
					"read_attempts", readAttempts,
					"read_success", readSuccess,
					"write_attempts", writeAttempts,
					"write_success", writeSuccess)
				return totalBytes, nil // Return nil error since this is expected
			}

			slog.ErrorContext(ctx, "=== PROXY ERROR (READ) ===",
				"name", name,
				"error", err,
				"error_type", errType,
				"error_string", errStr)

			// Report the real error for debugging but don't propagate
			return totalBytes, fmt.Errorf("reading from unix conn (%s): %w", name, err)
		}

		readSuccess++

		// Clear the deadline after successful read
		clearDeadlineErr := from.SetReadDeadline(time.Time{})
		if clearDeadlineErr != nil {
			if isClosedConnError(clearDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED WHEN CLEARING DEADLINE) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to clear read deadline", "name", name, "error", clearDeadlineErr)
		}

		slog.InfoContext(ctx, "!!! PACKET RECEIVED !!!",
			"name", name,
			"bytes", n,
			"first_bytes_hex", fmt.Sprintf("%x", buf[:min(n, 16)])) // Log at most first 16 bytes as hex

		// Check for SSH protocol signature in the first few bytes
		if n >= 4 {
			// SSH protocol starts with "SSH-" (0x5353482D)
			if string(buf[:4]) == "SSH-" {
				slog.InfoContext(ctx, "!!! SSH PACKET DETECTED !!!",
					"name", name,
					"bytes", n,
					"data", string(buf[:min(n, 64)]))
			}
		}

		writeAttempts++

		// Set a write deadline to avoid blocking on write
		writeDeadlineErr := to.SetWriteDeadline(time.Now().Add(500 * time.Millisecond))
		if writeDeadlineErr != nil {
			if isClosedConnError(writeDeadlineErr) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED WHEN SETTING WRITE DEADLINE) ===", "name", name)
				return totalBytes, nil
			}
			slog.WarnContext(ctx, "failed to set write deadline", "name", name, "error", writeDeadlineErr)
		}

		written, err := to.Write(buf[:n])

		// Clear write deadline
		to.SetWriteDeadline(time.Time{})

		if err != nil {
			// Handle closed connection gracefully
			if isClosedConnError(err) {
				slog.InfoContext(ctx, "=== PROXY FINISHED (CONN CLOSED DURING WRITE) ===", "name", name)
				return totalBytes, nil
			}

			slog.ErrorContext(ctx, "=== PROXY ERROR (WRITE) ===", "name", name, "error", err)
			// Report the real error for debugging but don't propagate
			return totalBytes, fmt.Errorf("writing to unix conn (%s): %w", name, err)
		}

		writeSuccess++
		slog.InfoContext(ctx, "!!! PACKET FORWARDED !!!", "name", name, "bytes", written)

		totalBytes += n
	}
}

// isClosedConnError returns true if the error is related to using a closed connection
func isClosedConnError(err error) bool {
	if err == nil {
		return false
	}

	// Check for standard EOF
	if err == io.EOF {
		return true
	}

	// Check for "use of closed network connection"
	errStr := err.Error()
	return errStr == "use of closed network connection" ||
		errStr == "read unixgram ->: use of closed network connection" ||
		errStr == "write unixgram ->: use of closed network connection"
}
