package vmm

import (
	"bytes"
	"context"
	"io"
	"sync"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"

	"github.com/walteh/ec1/gen/harpoon/harpoon_harpoond_arm64"
)

var (
	// Embedded, statically‑compiled init+gRPC binary (as XZ data)
	decompressedInitBin = []byte{}
	initMutex           = sync.Mutex{}
)

func LoadInitBinToMemory(ctx context.Context) ([]byte, error) {
	initMutex.Lock()
	defer initMutex.Unlock()

	if len(decompressedInitBin) > 0 {
		return decompressedInitBin, nil
	}

	arc, err := (&archives.Xz{}).OpenReader(bytes.NewReader(harpoon_harpoond_arm64.BinaryXZ))
	if err != nil {
		return nil, errors.Errorf("opening xz reader: %w", err)
	}

	decompressedInitBind, err := io.ReadAll(arc)
	if err != nil {
		return nil, errors.Errorf("reading uncompressed init bin: %w", err)
	}

	decompressedInitBin = decompressedInitBind

	return decompressedInitBin, nil
}
