package libkrun

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLibkrunBasicOperations(t *testing.T) {
	// Setup context with logger
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	// Test setting log level
	t.Run("SetLogLevel", func(t *testing.T) {
		err := SetLogLevel(ctx, 3) // Info level
		// Note: This might fail if libkrun is not installed, which is expected
		if err != nil {
			t.Skipf("libkrun not available: %v", err)
		}
	})

	// Test creating and freeing context
	t.Run("CreateAndFreeContext", func(t *testing.T) {
		kctx, err := CreateContext(ctx)
		if err != nil {
			t.Skipf("libkrun not available: %v", err)
		}
		require.NoError(t, err)
		require.NotNil(t, kctx)

		// Test freeing the context
		err = kctx.Free(ctx)
		assert.NoError(t, err)
	})

	// Test VM configuration
	t.Run("VMConfiguration", func(t *testing.T) {
		kctx, err := CreateContext(ctx)
		if err != nil {
			t.Skipf("libkrun not available: %v", err)
		}
		require.NoError(t, err)
		defer func() {
			_ = kctx.Free(ctx)
		}()

		// Test setting VM config
		err = kctx.SetVMConfig(ctx, 1, 512) // 1 vCPU, 512 MiB RAM
		assert.NoError(t, err)

		// Test setting root path (using /tmp as a safe test path)
		err = kctx.SetRoot(ctx, "/tmp")
		assert.NoError(t, err)

		// Test setting exec (simple echo command)
		err = kctx.SetExec(ctx, "/bin/echo", []string{"echo", "hello"}, []string{"PATH=/bin:/usr/bin"})
		assert.NoError(t, err)
	})
}

// TestLibkrunAvailability tests if libkrun is available on the system
func TestLibkrunAvailability(t *testing.T) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	_, err := CreateContext(ctx)
	if err != nil {
		t.Logf("libkrun is not available on this system: %v", err)
		t.Skip("libkrun not available - this is expected during development")
	} else {
		t.Log("libkrun is available on this system")
	}
}

// BenchmarkContextCreation benchmarks the context creation and cleanup
func BenchmarkContextCreation(b *testing.B) {
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	// Check if libkrun is available
	testCtx, err := CreateContext(ctx)
	if err != nil {
		b.Skipf("libkrun not available: %v", err)
	}
	_ = testCtx.Free(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		kctx, err := CreateContext(ctx)
		if err != nil {
			b.Fatal(err)
		}
		err = kctx.Free(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}
