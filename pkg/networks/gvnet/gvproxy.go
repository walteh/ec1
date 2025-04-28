package gvnet

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
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"github.com/containers/gvisor-tap-vsock/pkg/net/stdio"
	"github.com/containers/gvisor-tap-vsock/pkg/sshclient"
	"github.com/containers/gvisor-tap-vsock/pkg/tap"
	"github.com/containers/gvisor-tap-vsock/pkg/transport"
	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/dustin/go-humanize"
	"github.com/soheilhy/cmux"
	slogctx "github.com/veqryn/slog-context"
	"github.com/walteh/ec1/pkg/networks/gvnet/tapsock"
	"github.com/walteh/ec1/pkg/port"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

const (
	VIRTUAL_SUBNET_CIDR = "192.168.127.0/24"
	VIRTUAL_GATEWAY_IP  = "192.168.127.1"
	VIRTUAL_GUEST_IP    = "192.168.127.2"
	VIRUTAL_HOST_IP     = "192.168.127.254"
	VIRTUAL_GUEST_MAC   = "5a:94:ef:e4:0c:ee"
	VIRTUAL_GATEWAY_MAC = "5a:94:ef:e4:0c:dd"
	LOCAL_HOST_IP       = "127.0.0.1"
	host                = "host"
	gateway             = "gateway"
)

type Forward struct {
	Socket    string
	URIPath   string
	User      string
	Password  string
	PublicKey string
}

func (f *Forward) Validate() error {
	if f.User == "" {
		return errors.New("user is required")
	}
	if f.Password == "" && f.PublicKey == "" {
		return errors.New("password or public key is required")
	}
	if f.Socket == "" {
		return errors.New("socket is required")
	}
	if f.URIPath == "" {
		return errors.New("dest is required")
	}

	return nil
}

type GvproxyConfig struct {
	EnableDebug        bool // if true, print debug info
	EnableStdioSocket  bool // accept stdio pipe
	EnableNoConnectAPI bool // enable raw no connect API

	// restEndpoint               string // control endpoint
	// restEndpointWithoutConnect string // Exposes the same HTTP API as the --listen flag, without the /connect endpoint

	MTU      int // set the MTU, default is 1500
	VMSocket tapsock.VMSocket

	// GuestSSHPort int    // port to access the guest virtual machine, must be between 1024 and 65535
	VMHostPort string // host port to access the guest virtual machine, must be between 1024 and 65535
	// guestSSHPortOnHost string // host port to access the guest virtual machine
	// pidFile    string // path to pid file
	// workingDir string // working directory

	WorkingDir string // working directory
	ReadyChan  chan struct{}

	sshConnections []Forward // unix socket to be forwarded to the guest virtual machine over SSH
}

func GvproxyVersion() string {
	return types.NewVersion("gvnet").String()
}

func Proxy(ctx context.Context, cfg *GvproxyConfig) error {

	ctx = slogctx.WithGroup(ctx, "gvnet")

	// if cfg.GuestSSHPort == 0 {
	// 	cfg.GuestSSHPort = 22
	// }

	// log.Info(version.String())
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Make this the last defer statement in the stack
	// defer os.Exit(exitCode)

	groupErrs, ctx := errgroup.WithContext(ctx)

	for _, socket := range cfg.sshConnections {
		if err := socket.Validate(); err != nil {
			return errors.Errorf("ssh connection validation: %w", err)
		}
	}

	if cfg.VMSocket == nil {
		return errors.New("vmFileSocket is required")
	}

	// validate the vmFileSocket
	if err := cfg.VMSocket.Validate(); err != nil {
		return errors.Errorf("vmFileSocket validation: %w", err)
	}

	if cfg.MTU == 0 {
		cfg.MTU = 1500
	}

	if cfg.VMHostPort == "" {
		return errors.New("vmHostPort is required")
	}

	dnss, err := searchDomains(ctx)
	if err != nil {
		return errors.Errorf("searching domains: %w", err)
	}

	m, virtualPortMap, err := buildForwards(ctx, cfg.VMHostPort, groupErrs, map[int]cmux.Matcher{
		22: cmux.PrefixMatcher("SSH-"),
	})
	if err != nil {
		return errors.Errorf("building forwards: %w", err)
	}

	slog.InfoContext(ctx, "forwarders", slog.Any("virtualPortMap", virtualPortMap))

	config := types.Configuration{
		Debug:             cfg.EnableDebug,
		CaptureFile:       captureFile(cfg),
		MTU:               cfg.MTU,
		Subnet:            VIRTUAL_SUBNET_CIDR,
		GatewayIP:         VIRTUAL_GATEWAY_IP,
		GatewayMacAddress: VIRTUAL_GATEWAY_MAC,
		DHCPStaticLeases: map[string]string{
			VIRTUAL_GUEST_IP: VIRTUAL_GUEST_MAC,
		},
		DNS: []types.Zone{
			{
				Name: "containers.internal.",
				Records: []types.Record{

					{
						Name: gateway,
						IP:   net.ParseIP(VIRTUAL_GATEWAY_IP),
					},
					{
						Name: host,
						IP:   net.ParseIP(VIRUTAL_HOST_IP),
					},
				},
			},
			{
				Name: "docker.internal.",
				Records: []types.Record{
					{
						Name: gateway,
						IP:   net.ParseIP(VIRTUAL_GATEWAY_IP),
					},
					{
						Name: host,
						IP:   net.ParseIP(VIRUTAL_HOST_IP),
					},
				},
			},
		},
		DNSSearchDomains: dnss,
		Forwards:         virtualPortMap,
		NAT: map[string]string{
			VIRUTAL_HOST_IP: LOCAL_HOST_IP,
		},
		GatewayVirtualIPs: []string{VIRUTAL_HOST_IP},
		VpnKitUUIDMacAddresses: map[string]string{
			"c3d68012-0208-11ea-9fd7-f2189899ab08": VIRTUAL_GUEST_MAC,
		},
		Protocol: cfg.VMSocket.Protocol(),
	}

	groupErrs.Go(func() error {
		defer func() {
			if cfg.ReadyChan != nil {
				go func() {
					cfg.ReadyChan <- struct{}{}
				}()
			}
		}()
		return run(ctx, groupErrs, &config, cfg, m, virtualPortMap)
	})

	if err := groupErrs.Wait(); err != nil {
		return errors.Errorf("gvnet exiting: %v", err)
	}
	return nil
}

func buildForwards(ctx context.Context, globalHostPort string, groupErrs *errgroup.Group, forwards map[int]cmux.Matcher) (cmux.CMux, map[string]string, error) {
	l, err := transport.Listen(globalHostPort)
	if err != nil {
		return nil, nil, errors.Errorf("listen: %w", err)
	}

	virtualPortMap := make(map[string]string)

	m := cmux.New(l)

	for guestPortTarget, matcher := range forwards {

		listener := m.Match(matcher)

		hostProxyPort, err := port.ReservePort(ctx)
		if err != nil {
			return nil, nil, errors.Errorf("reserving ssh port: %w", err)
		}

		hostProxyPortStr := fmt.Sprintf("%s:%d", LOCAL_HOST_IP, hostProxyPort)
		guestPortTargetStr := fmt.Sprintf("%s:%d", VIRTUAL_GUEST_IP, guestPortTarget)

		groupErrs.Go(func() error {
			return ForwardListenerToPort(ctx, listener, hostProxyPortStr, groupErrs)
		})

		virtualPortMap[hostProxyPortStr] = guestPortTargetStr

	}

	return m, virtualPortMap, nil
}

// func getSSHForwarders(guestSSHPort int, guestSSHPortOnHost string) (map[string]string, error) {
// 	if guestSSHPort < 1024 || guestSSHPort > 65535 {
// 		return nil, errors.New("ssh-port value must be between 1024 and 65535")
// 	}
// 	return map[string]string{
// 		fmt.Sprintf("127.0.0.1:%d", guestSSHPort): VIRTUAL_GUEST_IP,
// 	}, nil
// }

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func captureFile(cfg *GvproxyConfig) string {
	if !cfg.EnableDebug {
		return ""
	}
	return filepath.Join(cfg.WorkingDir, "capture.pcap")
}

func run(ctx context.Context, g *errgroup.Group, configuration *types.Configuration, cfg *GvproxyConfig, cmuxl cmux.CMux, virtualPortMap map[string]string) error {
	vn, err := virtualnetwork.New(configuration)
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "waiting for clients... listening...", "endpoint", cfg.VMHostPort)

	mux := vn.Mux()
	if cfg.EnableDebug {
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
	httpServe(ctx, g, cmuxl.Match(cmux.Any()), mux)

	if cfg.EnableNoConnectAPI {
		slog.InfoContext(ctx, "enabling raw no connect API at /no-connect")
		mux := http.NewServeMux()
		mux.Handle("/no-connect", vn.Mux())
		httpServe(ctx, g, cmuxl.Match(cmux.Any()), mux)
	}

	// setup the gateway cmuxl
	vml, err := vn.Listen("tcp", fmt.Sprintf("%s:80", VIRTUAL_GATEWAY_IP))
	if err != nil {
		return err
	}

	mux = http.NewServeMux()
	mux.Handle("/services/forwarder/all", vn.Mux())
	mux.Handle("/services/forwarder/expose", vn.Mux())
	mux.Handle("/services/forwarder/unexpose", vn.Mux())
	httpServe(ctx, g, vml, mux)

	g.Go(func() error {
		if err := cmuxl.Serve(); err != nil {
			return errors.Errorf("serving cmux: %w", err)
		}
		return nil
	})

	if cfg.EnableDebug {
		g.Go(func() error {
		debugLog:
			for {
				select {
				case <-time.After(5 * time.Second):
					slog.DebugContext(ctx, "virtual network transfers", "sent", humanize.Bytes(vn.BytesSent()), "received", humanize.Bytes(vn.BytesReceived()))
				case <-ctx.Done():
					break debugLog
				}
			}
			return nil
		})
	}

	swtch := GetUnexportedField(reflect.ValueOf(vn).Elem().FieldByName("networkSwitch")).(*tap.Switch)

	// start the vmFileSocket
	if err := cfg.VMSocket.Listen(ctx, g, swtch); err != nil {
		return errors.Errorf("vmFileSocket listen: %w", err)
	}

	if cfg.EnableStdioSocket {
		g.Go(func() error {
			conn := stdio.GetStdioConn()
			return vn.AcceptStdio(ctx, conn)
		})
	}

	if len(cfg.sshConnections) > 0 {
		// i am still not quite sure if we will need this funcitonality, leaving just in case for now
		return errors.New("ssh connections are not supported yet")
	}

	for _, socket := range cfg.sshConnections {
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
			Host:   fmt.Sprintf("%s:22", VIRTUAL_GUEST_IP),
			Path:   socket.URIPath,
		}
		g.Go(func() error {
			defer os.Remove(socket.Socket)
			publicKey, err := makeTmpFileForPublicKey(ctx, socket.PublicKey, g)
			if err != nil {
				return err
			}
			var forward *sshclient.SSHForward
			if socket.Password == "" {
				forward, err = sshclient.CreateSSHForward(ctx, src, dest, publicKey, vn)
				if err != nil {
					return err
				}
			} else {
				forward, err = sshclient.CreateSSHForwardPassphrase(ctx, src, dest, publicKey, socket.Password, vn)
				if err != nil {
					return err
				}
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

func makeTmpFileForPublicKey(ctx context.Context, publicKey string, g *errgroup.Group) (string, error) {
	if publicKey == "" {
		return "", nil
	}

	tempFile, err := os.CreateTemp("", "tmp.pub")
	if err != nil {
		return "", errors.Errorf("creating temp file: %w", err)
	}

	g.Go(func() error {
		<-ctx.Done()
		if err := os.Remove(publicKey); err != nil {
			slog.WarnContext(ctx, "removing public key", "error", err)
		}
		return nil
	})

	if _, err := tempFile.WriteString(publicKey); err != nil {
		return "", errors.Errorf("writing temp file: %w", err)
	}

	return tempFile.Name(), nil
}

func httpServe(ctx context.Context, g *errgroup.Group, ln net.Listener, mux http.Handler) {
	g.Go(func() error {
		<-ctx.Done()
		return errors.Errorf("http serve: %w", ln.Close())
	})
	g.Go(func() error {
		s := &http.Server{
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		err := s.Serve(ln)
		if err != nil {
			// if err != http.ErrServerClosed {
			// 	return errors.Errorf("http serve: %w", err)
			// }
			return errors.Errorf("http serve: %w", err)
		}
		return nil
	})
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

func GetUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

func SetUnexportedField(field reflect.Value, value interface{}) {
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
		Elem().
		Set(reflect.ValueOf(value))
}
