package applevftest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/walteh/ec1/pkg/hypervisors/vf/config"

	"golang.org/x/crypto/ssh"
)

type OsProvider interface {
	URL() string
	Initialize(ctx context.Context, cacheDir string) error
	ToVirtualMachine(ctx context.Context) (*config.VirtualMachine, error)
	SSHConfig() *ssh.ClientConfig
	ShutdownCommand() string
	Name() string
	Version() string
}

func cacheDir(urld string) (string, error) {
	hrlHasher := sha256.New()
	hrlHasher.Write([]byte(urld))
	hrlHash := hex.EncodeToString(hrlHasher.Sum(nil))

	// parse the url and get the filename
	parsedURL, err := url.Parse(urld)
	if err != nil {
		return "", err
	}

	dirname := fmt.Sprintf("%s_%s", parsedURL.Host, hrlHash[:16])
	userCacheDir, err := cacheDirPrefix()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCacheDir, dirname), nil
}

func cacheDirPrefix() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCacheDir, "vfkit-testing", "cache"), nil
}

func init() {
	const clearCache = false
	if clearCache {
		prefix, err := cacheDirPrefix()
		if err != nil {
			slog.Error("failed to get cache dir prefix", "error", err)
			return
		}
		os.RemoveAll(prefix)
	}
}

func kernelArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	default:
		return "invalid"
	}
}

type SSHAccessMethod struct {
	network string
	port    uint
}

func findFirstFileWithExtension(cacheDir string, extension string) (string, error) {
	filez, err := os.ReadDir(cacheDir)
	if err != nil {
		return "", err
	}
	files := []string{}
	for _, file := range filez {
		files = append(files, filepath.Join(cacheDir, file.Name()))
	}
	for _, f := range files {
		if strings.HasSuffix(f, extension) {
			return f, nil
		}
	}
	return "", fmt.Errorf("could not find %s", extension)
}

func findFile(files []string, filename string) (string, error) {
	for _, f := range files {
		if filepath.Base(f) == filename {
			return f, nil
		}
	}

	return "", fmt.Errorf("could not find %s", filename)
}
