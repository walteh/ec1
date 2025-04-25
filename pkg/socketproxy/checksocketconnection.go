package socketproxy

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"gitlab.com/tozd/go/errors"
)

const dialTimeout = 5 // timeout in seconds for the socket connection

// checkSocketAvailability tries to connect to the socket and returns an error if it fails.
func checkSocketAvailability(socketPath string) error {
	slog.Debug("checking socket availability", "origin", "checkSocketAvailability")
	conn, err := net.DialTimeout("unix", socketPath, dialTimeout*time.Second)
	if err != nil {
		return err
	}
	err = conn.Close()
	if err != nil {
		slog.Error("error closing socket", "origin", "checkSocketAvailability", "error", err)
	}
	return nil
}

// startSocketWatchdog starts a watchdog that checks the socket availability every n seconds.
func startSocketWatchdog(ctx context.Context, socketPath string, interval int64) error {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := checkSocketAvailability(socketPath); err != nil {
				return errors.Errorf("checking socket availability from watchdog: %w", err)
			}
		}
	}
}

// healthCheckServer starts a http server that listens on localhost:55555/health
// and returns 200 if the socket is available, 503 otherwise.
func healthCheckServer(ctx context.Context, socketPath string) error {
	hcMux := http.NewServeMux()
	hcMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		err := checkSocketAvailability(socketPath)
		if err != nil {
			slog.Error("health check failed", "origin", "healthcheck", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	hcSrv := &http.Server{
		Addr:              "127.0.0.1:55555",
		Handler:           hcMux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	if err := hcSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Errorf("serving healthcheck HTTP: %w", err)
	}
	return nil
}
