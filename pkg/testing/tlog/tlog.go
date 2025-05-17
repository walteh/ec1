package tlog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func init() {
	logrusshim.ForwardLogrusToSlogGlobally()
}

func SetupSlogForTestWithContext(t *testing.T, ctx context.Context) context.Context {

	simpctx := logging.SetupSlogSimple(ctx)

	cached, err := host.CacheDirPrefix()
	require.NoError(t, err)
	logging.RegisterRedactedLogValue(ctx, os.TempDir()+"/", "[os-tmp-dir]")
	logging.RegisterRedactedLogValue(ctx, cached, "[vm-cache-dir]")
	logging.RegisterRedactedLogValue(ctx, filepath.Dir(t.TempDir()), "[test-tmp-dir]") // higher priority than os-tmp-dir

	return simpctx
}

func SetupSlogForTest(t *testing.T) context.Context {
	return SetupSlogForTestWithContext(t, t.Context())
}
