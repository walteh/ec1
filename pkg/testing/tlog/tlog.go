package tlog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/walteh/ec1/pkg/host"
	"github.com/walteh/ec1/pkg/logging"
	"github.com/walteh/ec1/pkg/logging/logrusshim"
)

func init() {
	logrusshim.ForwardLogrusToSlogGlobally()
}

func SetupSlogForTestWithContext(t testing.TB, ctx context.Context) context.Context {

	simpctx := logging.SetupSlogSimple(ctx)

	cached, err := host.CacheDirPrefix()
	require.NoError(t, err)
	logging.RegisterRedactedLogValue(ctx, os.TempDir()+"/", "[os-tmp-dir]")
	logging.RegisterRedactedLogValue(ctx, cached, "[vm-cache-dir]")
	logging.RegisterRedactedLogValue(ctx, filepath.Dir(t.TempDir()), "[test-tmp-dir]") // higher priority than os-tmp-dir

	return simpctx
}

func SetupSlogForTest(t testing.TB) context.Context {
	return SetupSlogForTestWithContext(t, t.Context())
}

func TeeToDownloadsFolder(rdr io.Reader, filename string) (io.Reader, io.Closer) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	fle, err := os.Create(filepath.Join(homeDir, "Downloads", fmt.Sprintf("golang-test.%d.%s", time.Now().Unix(), filename)))
	if err != nil {
		panic(err)
	}

	return io.TeeReader(rdr, fle), fle
}

type BCompare struct {
	A io.Reader
	B io.Reader
}

func NewBComare(t testing.TB, a, b io.Reader) (*BCompare, io.ReadCloser, io.ReadCloser) {
	bwA := bytes.NewBuffer(nil)
	bwB := bytes.NewBuffer(nil)
	trA := io.TeeReader(a, bwA)
	trB := io.TeeReader(b, bwB)
	return &BCompare{
		A: bwA,
		B: bwB,
	}, io.NopCloser(trA), io.NopCloser(trB)
}

func (b *BCompare) TeeA(rdr io.Reader) io.Reader {
	bw := bytes.NewBuffer(nil)
	tr := io.TeeReader(rdr, bw)
	b.A = tr
	return tr
}

func (b *BCompare) TeeB(rdr io.Reader) io.Reader {
	bw := bytes.NewBuffer(nil)
	tr := io.TeeReader(rdr, bw)
	b.B = tr
	return tr
}

func (b *BCompare) Close() error {
	return nil
}

func (b *BCompare) Compare(t testing.TB) error {

	allA, err := io.ReadAll(b.A)
	if err != nil {
		return err
	}

	allB, err := io.ReadAll(b.B)
	if err != nil {
		return err
	}

	require.Equal(t, allA, allB)

	return nil
}
