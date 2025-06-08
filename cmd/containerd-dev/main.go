package main

import (
	_ "github.com/containerd/containerd/v2/cmd/containerd/builtins"
	slogctx "github.com/veqryn/slog-context"

	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/moby/sys/reexec"

	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/tcontainerd"
)

func init() {
	tcontainerd.ShimReexecInit()

	if reexec.Init() {
		os.Exit(0)
	}
}

var ctrCommands = FlagArray[string]{}

func main() {

	background := false
	debug := true
	json := false
	flag.BoolVar(&background, "background", false, "Run in background (daemon mode)")
	flag.BoolVar(&debug, "debug", true, "Run in debug mode")
	flag.Var(&ctrCommands, "ctr-command", "Command to run in ctr")
	flag.BoolVar(&json, "json", false, "Run in JSON mode")
	flag.Parse()

	var ctx context.Context

	if json {
		ctx = logging.SetupSlogSimpleToWriterWithProcessNameJSON(context.Background(), os.Stdout, debug, "containerd")
	} else {
		ctx = logging.SetupSlogSimpleToWriterWithProcessName(context.Background(), os.Stdout, debug, "containerd")
	}

	ctx = slogctx.Append(ctx, slog.String("process", "containerd"), slog.String("pid", strconv.Itoa(os.Getpid())))

	slog.InfoContext(ctx, "Starting development containerd instance",
		"background", background,
		"debug", debug,
		"ctr-commands", ctrCommands.Get())

	server, err := tcontainerd.NewDevContainerdServer(ctx, debug)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create containerd server", "error", err)
		os.Exit(1)
	}

	if len(ctrCommands.values) > 0 {
		slog.InfoContext(ctx, "Running ctr command", "command", ctrCommands.values)
		if err := server.StartBackground(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to start containerd in background", "error", err)
			server.Stop(ctx)
			os.Exit(1)
		}

		for _, command := range ctrCommands.values {
			if err := tcontainerd.RunCtrCommand(ctx, strings.Split(command, " ")...); err != nil {
				slog.ErrorContext(ctx, "Failed to run ctr command", "error", err)
				server.Stop(ctx)
				os.Exit(1)
			}
		}

		server.Stop(ctx)
		os.Exit(0)
	}

	if background {
		slog.InfoContext(ctx, "Starting containerd in background mode")
		if err := server.StartBackground(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to start containerd in background", "error", err)
			os.Exit(1)
		}

		// Print connection info and exit
		server.PrintConnectionInfoBackground()

	} else {
		// Foreground mode - handle signals gracefully
		slog.InfoContext(ctx, "Starting containerd in foreground mode")

		server.PrintConnectionInfoForground()

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Start containerd in goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Start(ctx)
		}()

		// Wait for either signal or error
		select {
		case sig := <-sigChan:
			slog.InfoContext(ctx, "Received signal, shutting down", "signal", sig)
			server.Stop(ctx)
		case err := <-errChan:
			if err != nil {
				slog.ErrorContext(ctx, "Containerd failed", "error", err)
				os.Exit(1)
			}
		}
	}
}

type FlagArray[T any] struct {
	values []T
}

func (f *FlagArray[T]) String() string {
	return fmt.Sprintf("%v", f.values)
}

func (f *FlagArray[T]) Set(value string) error {

	slog.InfoContext(context.Background(), "Setting flag", "value", value)

	var v T
	switch any(v).(type) {
	case string:
		f.values = append(f.values, any(value).(T))
	case int:
		i, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		f.values = append(f.values, any(i).(T))
	default:
		return fmt.Errorf("unsupported type: %T", v)
	}
	return nil
}

func (f *FlagArray[T]) Get() []T {
	return f.values
}
