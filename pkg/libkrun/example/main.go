package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog"

	"github.com/walteh/ec1/pkg/libkrun"
)

func main() {
	// Setup context with logger
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.DebugLevel)
	ctx = logger.WithContext(ctx)

	logger.Info().Msg("libkrun example starting")

	// Set log level for libkrun
	if err := libkrun.SetLogLevel(ctx, 3); err != nil {
		logger.Warn().Err(err).Msg("failed to set libkrun log level")
	}

	// Create a new libkrun context
	kctx, err := libkrun.CreateContext(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create libkrun context")
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nThis is expected if libkrun is not installed on your system.")
		fmt.Println("To install libkrun:")
		fmt.Println("  - On macOS: brew install libkrun")
		fmt.Println("  - On Linux: check your distribution's package manager")
		fmt.Println("\nThis example demonstrates the API structure even without libkrun installed.")
		return
	}

	// Ensure we clean up the context
	defer func() {
		if err := kctx.Free(ctx); err != nil {
			logger.Error().Err(err).Msg("failed to free libkrun context")
		}
	}()

	logger.Info().Msg("libkrun context created successfully")

	// Configure the VM
	if err := kctx.SetVMConfig(ctx, 1, 512); err != nil {
		logger.Error().Err(err).Msg("failed to set VM config")
		return
	}
	logger.Info().Msg("VM configuration set: 1 vCPU, 512 MiB RAM")

	// Set root filesystem (example - would need to be a real path)
	rootPath := "/tmp"
	if err := kctx.SetRoot(ctx, rootPath); err != nil {
		logger.Error().Err(err).Msg("failed to set VM root")
		return
	}
	logger.Info().Str("root_path", rootPath).Msg("VM root path set")

	// Set executable to run (example - simple echo command)
	execPath := "/bin/echo"
	argv := []string{"echo", "Hello from libkrun!"}
	envp := []string{"PATH=/bin:/usr/bin", "HOME=/tmp"}

	if err := kctx.SetExec(ctx, execPath, argv, envp); err != nil {
		logger.Error().Err(err).Msg("failed to set VM executable")
		return
	}
	logger.Info().
		Str("exec_path", execPath).
		Strs("argv", argv).
		Msg("VM executable configured")

	logger.Info().Msg("libkrun configuration completed successfully")
	fmt.Println("\n✅ libkrun PoC completed successfully!")
	fmt.Println("All configuration functions executed without errors.")
	fmt.Println("\nNote: StartEnter() was not called as it would actually start the VM.")
	fmt.Println("In a real application, you would call kctx.StartEnter(ctx) to start the microVM.")
}
