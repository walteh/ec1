package gvproxy

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/containers/gvisor-tap-vsock/pkg/net/stdio"
	"github.com/containers/gvisor-tap-vsock/pkg/sshclient"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/dustin/go-humanize"
	slogctx "github.com/veqryn/slog-context"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

// var (
// 	debug            bool
// 	mtu              int
// 	endpoints        arrayFlags
// 	vpnkitSocket     string
// 	qemuSocket       string
// 	bessSocket       string
// 	stdioSocket      string
// 	vfkitSocket      string
// 	forwardSocket    arrayFlags
// 	forwardDest      arrayFlags
// 	forwardUser      arrayFlags
// 	forwardIdentify  arrayFlags
// 	sshPort          int
// 	pidFile          string
// 	exitCode         int
// 	logFile          string
// 	servicesEndpoint string
// )

const (
	gatewayIP   = "192.168.127.1"
	sshHostPort = "192.168.127.2:22"
	hostIP      = "192.168.127.254"
	host        = "host"
	gateway     = "gateway"
)

// flag.Var(&endpoints, "listen", "control endpoint")
// flag.BoolVar(&debug, "debug", false, "Print debug info")
// flag.IntVar(&mtu, "mtu", 1500, "Set the MTU")
// flag.IntVar(&sshPort, "ssh-port", 2222, "Port to access the guest virtual machine. Must be between 1024 and 65535")
// flag.StringVar(&vpnkitSocket, "listen-vpnkit", "", "VPNKit socket to be used by Hyperkit")
// flag.StringVar(&qemuSocket, "listen-qemu", "", "Socket to be used by Qemu")
// flag.StringVar(&bessSocket, "listen-bess", "", "unixpacket socket to be used by Bess-compatible applications")
// flag.StringVar(&stdioSocket, "listen-stdio", "", "accept stdio pipe")
// flag.StringVar(&vfkitSocket, "listen-vfkit", "", "unixgram socket to be used by vfkit-compatible applications")
// flag.Var(&forwardSocket, "forward-sock", "Forwards a unix socket to the guest virtual machine over SSH")
// flag.Var(&forwardDest, "forward-dest", "Forwards a unix socket to the guest virtual machine over SSH")
// flag.Var(&forwardUser, "forward-user", "SSH user to use for unix socket forward")
// flag.Var(&forwardIdentify, "forward-identity", "Path to SSH identity key for forwarding")
// flag.StringVar(&pidFile, "pid-file", "", "Generate a file with the PID in it")
// flag.StringVar(&logFile, "log-file", "", "Output log messages (logrus) to a given file path")
// flag.StringVar(&servicesEndpoint, "services", "", "t")
// flag.Parse()

type Forward struct {
	Socket   string
	Dest     string
	User     string
	Identify string
}

type GvproxyConfig struct {
	debug            bool      // if true, print debug info
	mtu              int       // set the MTU
	endpoints        []string  // control endpoint
	vpnkitSocket     string    // VPNKit socket to be used by Hyperkit
	qemuSocket       string    // Socket to be used by Qemu
	bessSocket       string    // unixpacket socket to be used by Bess-compatible applications
	stdioSocket      string    // accept stdio pipe
	vfkitSocket      string    // unixgram socket to be used by vfkit-compatible applications
	forwardSocket    []Forward // unix socket to be forwarded to the guest virtual machine over SSH
	sshPort          int       // port to access the guest virtual machine, must be between 1024 and 65535
	sshHostPort      string    // host port to access the guest virtual machine
	pidFile          string    // path to pid file
	exitCode         int       // exit code
	logFile          string    // path to log file
	servicesEndpoint string    // Exposes the same HTTP API as the --listen flag, without the /connect endpoint
}

func GvproxyVersion() string {
	return types.NewVersion("gvproxy").String()
}

func Main(ctx context.Context, cfg *GvproxyConfig) error {

	ctx = slogctx.WithGroup(ctx, "gvproxy")

	// log.Info(version.String())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Make this the last defer statement in the stack
	// defer os.Exit(exitCode)

	groupErrs, ctx := errgroup.WithContext(ctx)

	// Make sure the qemu socket provided is valid syntax
	if len(cfg.qemuSocket) > 0 {
		uri, err := url.Parse(cfg.qemuSocket)
		if err != nil || uri == nil {
			return errors.Errorf("invalid value for listen-qemu %w", err)
		}
		if _, err := os.Stat(uri.Path); err == nil && uri.Scheme == "unix" {
			return errors.Errorf("%q already exists", uri.Path)
		}
	}
	if len(cfg.bessSocket) > 0 {
		uri, err := url.Parse(cfg.bessSocket)
		if err != nil || uri == nil {
			return errors.Errorf("invalid value for listen-bess %w", err)
		}
		if uri.Scheme != "unixpacket" {
			return errors.New("listen-bess must be unixpacket:// address")
		}
		if _, err := os.Stat(uri.Path); err == nil {
			return errors.Errorf("%q already exists", uri.Path)
		}
	}
	if len(cfg.vfkitSocket) > 0 {
		uri, err := url.Parse(cfg.vfkitSocket)
		if err != nil || uri == nil {
			return errors.Errorf("invalid value for listen-vfkit %w", err)
		}
		if uri.Scheme != "unixgram" {
			return errors.New("listen-vfkit must be unixgram:// address")
		}
		if _, err := os.Stat(uri.Path); err == nil {
			return errors.Errorf("%q already exists", uri.Path)
		}
	}

	if cfg.vpnkitSocket != "" && cfg.qemuSocket != "" {
		return errors.New("cannot use qemu and vpnkit protocol at the same time")
	}
	if cfg.vpnkitSocket != "" && cfg.bessSocket != "" {
		return errors.New("cannot use bess and vpnkit protocol at the same time")
	}
	if cfg.qemuSocket != "" && cfg.bessSocket != "" {
		return errors.New("cannot use qemu and bess protocol at the same time")
	}

	// If the given port is not between the privileged ports
	// and the oft considered maximum port, return an error.
	if cfg.sshPort != -1 && cfg.sshPort < 1024 || cfg.sshPort > 65535 {
		return errors.New("ssh-port value must be between 1024 and 65535")
	}
	protocol := types.HyperKitProtocol
	if cfg.qemuSocket != "" {
		protocol = types.QemuProtocol
	}
	if cfg.bessSocket != "" {
		protocol = types.BessProtocol
	}
	if cfg.vfkitSocket != "" {
		protocol = types.VfkitProtocol
	}

	for _, socket := range cfg.forwardSocket {
		_, err := os.Stat(socket.Identify)
		if err != nil {
			return errors.Errorf("Identity file %s can't be loaded: %w", socket.Identify, err)
		}
	}

	// Create a PID file if requested
	if len(cfg.pidFile) > 0 {
		f, err := os.Create(cfg.pidFile)
		if err != nil {
			return errors.Errorf("creating pid file %s: %w", cfg.pidFile, err)
		}
		// Remove the pid-file when exiting
		defer func() {
			if err := os.Remove(cfg.pidFile); err != nil {
				slog.ErrorContext(ctx, "removing pid file", "error", err)
			}
		}()
		pid := os.Getpid()
		if _, err := f.WriteString(strconv.Itoa(pid)); err != nil {
			return errors.Errorf("writing pid file %s: %w", cfg.pidFile, err)
		}
	}

	dnss, err := searchDomains(ctx)
	if err != nil {
		return errors.Errorf("searching domains: %w", err)
	}

	config := types.Configuration{
		Debug:             cfg.debug,
		CaptureFile:       captureFile(cfg),
		MTU:               cfg.mtu,
		Subnet:            "192.168.127.0/24",
		GatewayIP:         gatewayIP,
		GatewayMacAddress: "5a:94:ef:e4:0c:dd",
		DHCPStaticLeases: map[string]string{
			"192.168.127.2": "5a:94:ef:e4:0c:ee",
		},
		DNS: []types.Zone{
			{
				Name: "containers.internal.",
				Records: []types.Record{
					{
						Name: gateway,
						IP:   net.ParseIP(gatewayIP),
					},
					{
						Name: host,
						IP:   net.ParseIP(hostIP),
					},
				},
			},
			{
				Name: "docker.internal.",
				Records: []types.Record{
					{
						Name: gateway,
						IP:   net.ParseIP(gatewayIP),
					},
					{
						Name: host,
						IP:   net.ParseIP(hostIP),
					},
				},
			},
		},
		DNSSearchDomains: dnss,
		Forwards:         getForwardsMap(cfg.sshPort, cfg.sshHostPort),
		NAT: map[string]string{
			hostIP: "127.0.0.1",
		},
		GatewayVirtualIPs: []string{hostIP},
		VpnKitUUIDMacAddresses: map[string]string{
			"c3d68012-0208-11ea-9fd7-f2189899ab08": "5a:94:ef:e4:0c:ee",
		},
		Protocol: protocol,
	}

	groupErrs.Go(func() error {
		return run(ctx, groupErrs, &config, cfg)
	})

	// // Wait for something to happen
	// groupErrs.Go(func() error {
	// 	select {
	// 	// Catch signals so exits are graceful and defers can run
	// 	case <-sigChan:
	// 		cancel()
	// 		return errors.New("signal caught")
	// 	case <-ctx.Done():
	// 		return nil
	// 	}
	// })
	// Wait for all of the go funcs to finish up
	if err := groupErrs.Wait(); err != nil {
		return errors.Errorf("gvproxy exiting: %v", err)
	}
	return nil
}

func getForwardsMap(sshPort int, sshHostPort string) map[string]string {
	if sshPort == -1 {
		return map[string]string{}
	}
	return map[string]string{
		fmt.Sprintf("127.0.0.1:%d", sshPort): sshHostPort,
	}
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func captureFile(cfg *GvproxyConfig) string {
	if !cfg.debug {
		return ""
	}
	return "capture.pcap"
}

func run(ctx context.Context, g *errgroup.Group, configuration *types.Configuration, cfg *GvproxyConfig) error {
	vn, err := virtualnetwork.New(configuration)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "waiting for clients...")

	for _, endpoint := range cfg.endpoints {
		slog.InfoContext(ctx, "listening", "endpoint", endpoint)
		ln, err := transport.Listen(endpoint)
		if err != nil {
			return errors.Wrap(err, "cannot listen")
		}
		httpServe(ctx, g, ln, withProfiler(vn, cfg))
	}

	if cfg.servicesEndpoint != "" {
		slog.InfoContext(ctx, "enabling services API", "endpoint", cfg.servicesEndpoint)
		ln, err := transport.Listen(cfg.servicesEndpoint)
		if err != nil {
			return errors.Wrap(err, "cannot listen")
		}
		httpServe(ctx, g, ln, vn.ServicesMux())
	}

	ln, err := vn.Listen("tcp", fmt.Sprintf("%s:80", gatewayIP))
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("/services/forwarder/all", vn.Mux())
	mux.Handle("/services/forwarder/expose", vn.Mux())
	mux.Handle("/services/forwarder/unexpose", vn.Mux())
	httpServe(ctx, g, ln, mux)

	if cfg.debug {
		g.Go(func() error {
		debugLog:
			for {
				select {
				case <-time.After(5 * time.Second):
					slog.DebugContext(ctx, "%v sent to the VM, %v received from the VM", humanize.Bytes(vn.BytesSent()), humanize.Bytes(vn.BytesReceived()))
				case <-ctx.Done():
					break debugLog
				}
			}
			return nil
		})
	}

	if cfg.vpnkitSocket != "" {
		vpnkitListener, err := transport.Listen(cfg.vpnkitSocket)
		if err != nil {
			return errors.Wrap(err, "vpnkit listen error")
		}
		g.Go(func() error {
		vpnloop:
			for {
				select {
				case <-ctx.Done():
					break vpnloop
				default:
					// pass through
				}
				conn, err := vpnkitListener.Accept()
				if err != nil {
					slog.ErrorContext(ctx, "vpnkit accept error", "error", err)
					continue
				}
				g.Go(func() error {
					return vn.AcceptVpnKit(conn)
				})
			}
			return nil
		})
	}

	if cfg.qemuSocket != "" {
		qemuListener, err := transport.Listen(cfg.qemuSocket)
		if err != nil {
			return errors.Wrap(err, "qemu listen error")
		}

		g.Go(func() error {
			<-ctx.Done()
			if err := qemuListener.Close(); err != nil {
				slog.ErrorContext(ctx, "error closing", "socket", cfg.qemuSocket, "error", err)
			}
			return os.Remove(cfg.qemuSocket)
		})

		g.Go(func() error {
			conn, err := qemuListener.Accept()
			if err != nil {
				return errors.Wrap(err, "qemu accept error")
			}
			return vn.AcceptQemu(ctx, conn)
		})
	}

	if cfg.bessSocket != "" {
		bessListener, err := transport.Listen(cfg.bessSocket)
		if err != nil {
			return errors.Wrap(err, "bess listen error")
		}

		g.Go(func() error {
			<-ctx.Done()
			if err := bessListener.Close(); err != nil {
				slog.ErrorContext(ctx, "error closing", "socket", cfg.bessSocket, "error", err)
			}
			return os.Remove(cfg.bessSocket)
		})

		g.Go(func() error {
			conn, err := bessListener.Accept()
			if err != nil {
				return errors.Wrap(err, "bess accept error")

			}
			return vn.AcceptBess(ctx, conn)
		})
	}

	if cfg.vfkitSocket != "" {
		conn, err := transport.ListenUnixgram(cfg.vfkitSocket)
		if err != nil {
			return errors.Wrap(err, "vfkit listen error")
		}

		g.Go(func() error {
			<-ctx.Done()
			if err := conn.Close(); err != nil {
				slog.ErrorContext(ctx, "error closing", "socket", cfg.vfkitSocket, "error", err)
			}
			vfkitSocketURI, _ := url.Parse(cfg.vfkitSocket)
			return os.Remove(vfkitSocketURI.Path)
		})

		g.Go(func() error {
			vfkitConn, err := transport.AcceptVfkit(conn)
			if err != nil {
				return errors.Wrap(err, "vfkit accept error")
			}
			return vn.AcceptVfkit(ctx, vfkitConn)
		})
	}

	if cfg.stdioSocket != "" {
		g.Go(func() error {
			conn := stdio.GetStdioConn()
			return vn.AcceptStdio(ctx, conn)
		})
	}

	for _, socket := range cfg.forwardSocket {
		var (
			src *url.URL
			err error
		)
		if strings.Contains(socket.Socket, "://") {
			src, err = url.Parse(socket.Socket)
			if err != nil {
				return err
			}
		} else {
			src = &url.URL{
				Scheme: "unix",
				Path:   socket.Socket,
			}
		}

		dest := &url.URL{
			Scheme: "ssh",
			User:   url.User(socket.User),
			Host:   cfg.sshHostPort,
			Path:   socket.Dest,
		}
		g.Go(func() error {
			defer os.Remove(socket.Socket)
			forward, err := sshclient.CreateSSHForward(ctx, src, dest, socket.Identify, vn)
			if err != nil {
				return err
			}
			go func() {
				<-ctx.Done()
				// Abort pending accepts
				forward.Close()
			}()
		loop:
			for {
				select {
				case <-ctx.Done():
					break loop
				default:
					// proceed
				}
				err := forward.AcceptAndTunnel(ctx)
				if err != nil {
					slog.WarnContext(ctx, "Error occurred handling ssh forwarded connection", "error", err)
				}
			}
			return nil
		})
	}

	return nil
}

func httpServe(ctx context.Context, g *errgroup.Group, ln net.Listener, mux http.Handler) {
	g.Go(func() error {
		<-ctx.Done()
		return ln.Close()
	})
	g.Go(func() error {
		s := &http.Server{
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		err := s.Serve(ln)
		if err != nil {
			if err != http.ErrServerClosed {
				return err
			}
			return err
		}
		return nil
	})
}

func withProfiler(vn *virtualnetwork.VirtualNetwork, cfg *GvproxyConfig) http.Handler {
	mux := vn.Mux()
	if cfg.debug {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
	return mux
}

func searchDomains(ctx context.Context) ([]string, error) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		f, err := os.Open("/etc/resolv.conf")
		if err != nil {
			return nil, errors.Errorf("opening resolv.conf: %w", err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		searchPrefix := "search "
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), searchPrefix) {
				return parseSearchString(ctx, sc.Text(), searchPrefix), nil
			}
		}
		if err := sc.Err(); err != nil {
			return nil, errors.Errorf("scanning resolv.conf: %w", err)
		}
	}
	return nil, errors.New("only Linux and macOS are supported currently")
}

// Parse and sanitize search list
// macOS has limitation on number of domains (6) and general string length (256 characters)
// since glibc 2.26 Linux has no limitation on 'search' field
func parseSearchString(ctx context.Context, text, searchPrefix string) []string {
	// macOS allow only 265 characters in search list
	if runtime.GOOS == "darwin" && len(text) > 256 {
		slog.WarnContext(ctx, "Search domains list is too long, it should not exceed 256 chars on macOS - truncating", "length", len(text))
		text = text[:256]
		lastSpace := strings.LastIndex(text, " ")
		if lastSpace != -1 {
			text = text[:lastSpace]
		}
	}

	searchDomains := strings.Split(strings.TrimPrefix(text, searchPrefix), " ")
	slog.DebugContext(ctx, "Using search domains", "domains", searchDomains)

	// macOS allow only 6 domains in search list
	if runtime.GOOS == "darwin" && len(searchDomains) > 6 {
		slog.WarnContext(ctx, "Search domains list is too long, it should not exceed 6 domains on macOS - truncating", "length", len(searchDomains))
		searchDomains = searchDomains[:6]
	}

	return searchDomains
}
