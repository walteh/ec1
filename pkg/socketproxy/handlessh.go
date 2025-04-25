package socketproxy

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"slices"
	"sync"

	"github.com/walteh/ec1/pkg/socketproxy/config"
	"golang.org/x/crypto/ssh"
)

func handleSSHConnection(cfg *config.Config, conn net.Conn, sshServer *ssh.Conn) {
	// 1. Check client IP allowlist (similar to HTTP)
	clientAddr := conn.RemoteAddr().String()
	allowedIP, err := isAllowedClient(cfg, clientAddr)
	if err != nil {
		slog.Warn("cannot get valid IP address for client allowlist check", "reason", err, "client", clientAddr)
	}
	if !allowedIP {
		slog.Info("blocked SSH connection", "reason", "forbidden IP", "client", clientAddr)
		conn.Close()
		return
	}

	// 2. Create SSH server config for this connection
	sshConfig := &ssh.ServerConfig{
		// Define authentication callbacks
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			username := conn.User()
			keyString := string(ssh.MarshalAuthorizedKey(key))

			// Check if this user+key combination is allowed
			if !isAllowedSSHUser(cfg, username, keyString) {
				slog.Info("blocked SSH auth", "reason", "unauthorized user or key", "user", username)
				return nil, fmt.Errorf("permission denied")
			}

			slog.Debug("allowed SSH auth", "user", username, "client", clientAddr)
			return &ssh.Permissions{}, nil
		},
	}

	// 3. Create SSH connection and handle
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		slog.Warn("failed SSH handshake", "reason", err, "client", clientAddr)
		return
	}
	defer sshConn.Close()

	// 4. Forward connection to the backend SSH server
	// This is where you'd implement command/operation filtering
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		// Check if channel type is allowed
		if !isAllowedChannelType(cfg, newChannel.ChannelType()) {
			newChannel.Reject(ssh.Prohibited, "channel type not allowed")
			continue
		}

		// Accept and forward the channel
		forwardSSHChannel(newChannel, sshServer)
	}
}

// Check if user+key is allowed
func isAllowedSSHUser(cfg *config.Config, username, publicKey string) bool {
	// Check against allowed users and their keys
	allowedKeys, exists := cfg.AllowedSSHUsers[username]
	if !exists {
		return false
	}

	return slices.Contains(allowedKeys, publicKey)
}

// Check if the SSH channel type is allowed
func isAllowedChannelType(cfg *config.Config, channelType string) bool {
	// Check channel types like "session", "direct-tcpip", etc.
	for _, allowed := range cfg.AllowedSSHChannelTypes {
		if allowed == channelType {
			return true
		}
	}
	return false
}

// Forward SSH channel to backend server
func forwardSSHChannel(newChannel ssh.NewChannel, backendAddr string) {
	// Accept the channel from the client
	channel, requests, err := newChannel.Accept()
	if err != nil {
		slog.Warn("could not accept channel", "reason", err)
		return
	}
	defer channel.Close()

	// Connect to the backend SSH server
	backendConn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		slog.Error("failed to connect to backend SSH server", "error", err)
		return
	}
	defer backendConn.Close()

	// Set up bidirectional copying
	var once sync.Once
	closeFunc := func() {
		channel.Close()
		backendConn.Close()
	}

	// Copy from client to backend
	go func() {
		io.Copy(backendConn, channel)
		once.Do(closeFunc)
	}()

	// Copy from backend to client
	go func() {
		io.Copy(channel, backendConn)
		once.Do(closeFunc)
	}()

	// Forward all channel requests
	go func() {
		for req := range requests {
			// You could filter certain requests here
			payload := req.Payload
			ok, reply, err := forwardRequest(backendConn, req.Type, req.WantReply, payload)
			if req.WantReply {
				req.Reply(ok, reply)
			}
		}
	}()
}
