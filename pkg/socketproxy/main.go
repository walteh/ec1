package socketproxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/walteh/ec1/pkg/socketproxy/config"
	"gitlab.com/tozd/go/errors"
	"golang.org/x/sync/errgroup"
)

const (
	programURL   = "github.com/wollomatic/socket-proxy"
	logAddSource = false // set to true to log the source position (file and line) of the log message
)

var (
	version     = "dev" // will be overwritten by build system
	socketProxy *httputil.ReverseProxy
	cfg         *config.Config
)

func Run(ctx context.Context, cfg *config.Config) error {

	// print configuration
	slog.Info("starting socket-proxy", "version", version, "os", runtime.GOOS, "arch", runtime.GOARCH, "runtime", runtime.Version(), "URL", programURL)
	if cfg.ProxySocketEndpoint == "" {
		slog.Info("configuration info", "socketpath", cfg.SocketPath, "listenaddress", cfg.ListenAddress, "allowfrom", cfg.AllowFrom, "shutdowngracetime", cfg.ShutdownGraceTime)
	} else {
		slog.Info("configuration info", "socketpath", cfg.SocketPath, "proxysocketendpoint", cfg.ProxySocketEndpoint, "proxysocketendpointfilemode", cfg.ProxySocketEndpointFileMode, "allowfrom", cfg.AllowFrom, "shutdowngracetime", cfg.ShutdownGraceTime)
		slog.Info("proxysocketendpoint is set, so the TCP listener is deactivated")
	}
	if cfg.WatchdogInterval > 0 {
		slog.Info("watchdog enabled", "interval", cfg.WatchdogInterval, "stoponwatchdog", cfg.StopOnWatchdog)
	} else {
		slog.Info("watchdog disabled")
	}

	// print request allow list
	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		for method, regex := range cfg.AllowedRequests {
			slog.DebugContext(ctx, "configured allowed request", "method", method, "regex", regex)
		}
	} else {
		// don't use slog here, as we want to print the regexes as they are
		// see https://github.com/wollomatic/socket-proxy/issues/11
		fmt.Printf("Request allowlist:\n   %-8s %s\n", "Method", "Regex")
		for method, regex := range cfg.AllowedRequests {
			fmt.Printf("   %-8s %s\n", method, regex)
		}
	}

	// check if the socket is available
	err := checkSocketAvailability(cfg.SocketPath)
	if err != nil {
		slog.Error("socket not available", "error", err)
		os.Exit(2)
	}

	// define the reverse proxy
	socketURLDummy, _ := url.Parse("http://localhost") // dummy URL - we use the unix socket
	socketProxy = httputil.NewSingleHostReverseProxy(socketURLDummy)
	socketProxy.Transport = &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", cfg.SocketPath)
		},
	}

	var l net.Listener
	if cfg.ProxySocketEndpoint != "" {
		if _, err = os.Stat(cfg.ProxySocketEndpoint); err == nil {
			slog.WarnContext(ctx, fmt.Sprintf("%s already exists, removing existing file", cfg.ProxySocketEndpoint))
			if err = os.Remove(cfg.ProxySocketEndpoint); err != nil {
				return errors.Errorf("removing existing socket file: %w", err)
			}
		}
		l, err = net.Listen("unix", cfg.ProxySocketEndpoint)
		if err != nil {
			return errors.Errorf("creating socket: %w", err)
		}
		if err = os.Chmod(cfg.ProxySocketEndpoint, cfg.ProxySocketEndpointFileMode); err != nil {
			return errors.Errorf("setting socket file permissions: %w", err)
		}
	} else {
		l, err = net.Listen("tcp", cfg.ListenAddress())
		if err != nil {
			return errors.Errorf("listening to address %s: %w", cfg.ListenAddress(), err)
		}
	}

	srv := &http.Server{ // #nosec G112 -- intentionally do not time out the client
		Handler: http.HandlerFunc(handleHTTPRequest), // #nosec G112
	} // #nosec G112

	grp, ctx := errgroup.WithContext(ctx)

	// start the server in a goroutine
	grp.Go(func() error {
		if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return errors.Errorf("serving HTTP: %w", err)
		}
		return nil
	})

	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			slog.WarnContext(ctx, "proxy server shutdown failed", "error", err)
		}
	}()

	slog.Info("socket-proxy running and listening...")

	// start the watchdog if configured
	if cfg.WatchdogInterval > 0 {
		grp.Go(func() error {
			return startSocketWatchdog(ctx, cfg.SocketPath, int64(cfg.WatchdogInterval)) // #nosec G115 - we validated the integer size in config.go
		})
		slog.DebugContext(ctx, "watchdog running")
	}

	// start the health check server if configured
	if cfg.AllowHealthcheck {
		grp.Go(func() error {
			return healthCheckServer(ctx, cfg.SocketPath)
		})
		slog.DebugContext(ctx, "healthcheck ready")
	}

	// Try to shut down gracefully
	ctx, cancel := context.WithTimeout(ctx, time.Duration(int64(cfg.ShutdownGraceTime))*time.Second) // #nosec G115 - we validated the integer size in config.go
	defer cancel()

	if err := grp.Wait(); err != nil {
		return errors.Errorf("running server: %w", err)
	}
	return nil

}
