package testdataembed

import (
	"embed"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func MustCreateTmpFileFor(t *testing.T, data *embed.FS, name string) string {
	tmpFile, err := CreateTmpFileFor(t, data, name)
	require.NoError(t, err)
	return tmpFile
}

func CreateTmpFileFor(t *testing.T, data *embed.FS, name string) (string, error) {
	content, err := data.ReadFile(name)
	if err != nil {
		return "", err
	}

	tmpDir := t.TempDir()

	tmpFile, err := os.Create(filepath.Join(tmpDir, name))
	if err != nil {
		return "", err
	}

	_, err = tmpFile.Write(content)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

var tmpAssetsDirOnce sync.Once
var tmpAssetsDir string

func CreateTmpFilesOnce(t *testing.T, data *embed.FS) (string, error) {
	tmpAssetsDirOnce.Do(func() {
		var err error
		tmpAssetsDir, err = CreateTmpFiles(t, data)
		require.NoError(t, err)

	})
	return tmpAssetsDir, nil
}

func CreateTmpFiles(t *testing.T, data *embed.FS) (string, error) {
	files, err := data.ReadDir(".")
	if err != nil {
		return "", err
	}

	tmpDir := t.TempDir()

	for _, file := range files {
		content, err := data.ReadFile(file.Name())
		if err != nil {
			return "", err
		}

		tmpFile, err := os.Create(filepath.Join(tmpDir, file.Name()))
		if err != nil {
			return "", err
		}

		_, err = tmpFile.Write(content)
		if err != nil {
			return "", err
		}

	}

	return tmpDir, nil

}
