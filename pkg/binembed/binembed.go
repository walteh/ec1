package binembed

import (
	"bytes"
	"io"
	"sync"

	"github.com/mholt/archives"
	"gitlab.com/tozd/go/errors"
)

var registry = map[string][]byte{}
var decompressors = map[string]archives.Compression{}

var registryMutex = sync.Mutex{}

var decompressedRegistry = map[string][]byte{}
var decompressedRegistryMutex = sync.Mutex{}

func RegisterXZ(checkSum string, binary []byte) {
	registryMutex.Lock()
	defer registryMutex.Unlock()
	registry[checkSum] = binary
	decompressors[checkSum] = &archives.Xz{}
}

func GetDecompressed(checkSum string) (io.Reader, error) {
	decompressedRegistryMutex.Lock()
	bin, ok := decompressedRegistry[checkSum]
	decompressedRegistryMutex.Unlock()

	if ok {
		return bytes.NewReader(bin), nil
	}

	registryMutex.Lock()
	bin, ok = registry[checkSum]
	registryMutex.Unlock()

	if !ok {
		return nil, errors.Errorf("binary not found: %s", checkSum)
	}

	registryMutex.Lock()
	decompressor, ok := decompressors[checkSum]
	registryMutex.Unlock()

	if !ok {
		return nil, errors.Errorf("decompressor not found: %s", checkSum)
	}

	decompressed, err := decompressor.OpenReader(bytes.NewReader(bin))
	if err != nil {
		return nil, errors.Errorf("decompressing binary: %w", err)
	}

	byt, err := io.ReadAll(decompressed)
	if err != nil {
		return nil, errors.Errorf("reading decompressed binary: %w", err)
	}

	decompressedRegistryMutex.Lock()
	decompressedRegistry[checkSum] = byt
	decompressedRegistryMutex.Unlock()

	return bytes.NewReader(byt), nil
}

func MustGetDecompressed(checkSum string) io.Reader {
	reader, err := GetDecompressed(checkSum)
	if err != nil {
		panic(err)
	}
	return reader
}
