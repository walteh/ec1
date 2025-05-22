package vmm

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/binembed/lgia_linux_arm64"
)

var (
	// Embedded, statically‑compiled init+gRPC binary (as XZ data)
	initBin             = lgia_linux_arm64.BinaryXZ
	decompressedInitBin = []byte{}
	initMutex           = sync.Mutex{}
	customInitPath      = "init"
)

func LoadInitBinToMemory(ctx context.Context) ([]byte, error) {
	initMutex.Lock()
	defer initMutex.Unlock()

	if len(decompressedInitBin) > 0 {
		return decompressedInitBin, nil
	}

	arc, err := (&archives.Xz{}).OpenReader(bytes.NewReader(initBin))
	if err != nil {
		return nil, errors.Errorf("opening xz reader: %w", err)
	}

	decompressedInitBind, err := io.ReadAll(arc)
	if err != nil {
		return nil, errors.Errorf("reading uncompressed init bin: %w", err)
	}

	slog.InfoContext(ctx, "loaded init bin to memory", "size", len(decompressedInitBind))

	decompressedInitBin = decompressedInitBind

	return decompressedInitBin, nil
}
