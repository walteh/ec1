package host

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	const clearCache = false
	if clearCache {
		prefix, err := CacheDirPrefix()
		if err != nil {
			slog.Error("failed to get cache dir prefix", "error", err)
			return
		}
		os.RemoveAll(prefix)
	}
}

func CacheDirForURL(urld string) (string, error) {
	hrlHasher := sha256.New()
	hrlHasher.Write([]byte(urld))
	hrlHash := hex.EncodeToString(hrlHasher.Sum(nil))

	// parse the url and get the filename
	parsedURL, err := url.Parse(urld)
	if err != nil {
		return "", err
	}

	dirname := fmt.Sprintf("%s_%s", parsedURL.Host, hrlHash[:16])
	userCacheDir, err := CacheDirPrefix()
	if err != nil {
		return "", err
	}

	return filepath.Join(userCacheDir, "downloads", dirname), nil
}

func CacheDirPrefix() (string, error) {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userCacheDir, "ec1", "cache"), nil
}

func TempDir(ctx context.Context) string {
	tmp, err := os.MkdirTemp(filepath.Join(os.TempDir(), "ec1"), "hostfs-*")
	if err != nil {
		panic(fmt.Sprintf("failed to create temp dir - should not happen: %s", err))
	}

	go func() {
		<-ctx.Done()
		if err := os.RemoveAll(tmp); err != nil {
			slog.Error("failed to remove temp dir", "error", err)
		}
	}()

	return tmp
}

func FindFirstFileWithExtension(cacheDir string, extension string) (string, error) {
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

func FindFile(files []string, filename string) (string, error) {
	for _, f := range files {
		if filepath.Base(f) == filename {
			return f, nil
		}
	}

	return "", fmt.Errorf("could not find %s", filename)
}
