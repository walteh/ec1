package applevf_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/walteh/ec1/pkg/cloud/hypervisor/applevf"
	"github.com/walteh/ec1/pkg/cloud/hypervisor/applevf/applevftest/testdata"
	"github.com/walteh/ec1/pkg/embedtd"
)

func TestStartIgnitionProvisionerServer(t *testing.T) {
	socketPath := "virtiovsock"
	defer os.Remove(socketPath)

	ignitionData := []byte("ignition configuration")
	ignitionReader := bytes.NewReader(ignitionData)

	// Start the server using the socket so that it can returns the ignition data
	go func() {
		err := applevf.StartIgnitionProvisionerServer(t.Context(), ignitionReader, socketPath)
		require.NoError(t, err)
	}()

	// Wait for the socket file to be created before serving, up to 2 seconds
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Make a request to the server
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(_, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
	resp, err := client.Get("http://unix://" + socketPath)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify the response from the server is actually the ignition data
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, ignitionData, body)
}

func TestGenerateCloudInitImage(t *testing.T) {

	iso, err := applevf.GenerateCloudInitImage(t.Context(), []string{
		embedtd.MustCreateTmpFileFor(t, testdata.FS(), "user-data"),
		embedtd.MustCreateTmpFileFor(t, testdata.FS(), "meta-data"),
	})
	require.NoError(t, err)

	assert.Contains(t, iso, "vfkit-cloudinit")

	_, err = os.Stat(iso)
	require.NoError(t, err)

	err = os.Remove(iso)
	require.NoError(t, err)
}

func TestGenerateCloudInitImageWithMissingFile(t *testing.T) {

	iso, err := applevf.GenerateCloudInitImage(t.Context(), []string{
		embedtd.MustCreateTmpFileFor(t, testdata.FS(), "user-data"),
	})
	require.NoError(t, err)

	assert.Contains(t, iso, "vfkit-cloudinit")

	_, err = os.Stat(iso)
	require.NoError(t, err)

	err = os.Remove(iso)
	require.NoError(t, err)
}

func TestGenerateCloudInitImageWithWrongFile(t *testing.T) {

	iso, err := applevf.GenerateCloudInitImage(t.Context(), []string{
		embedtd.MustCreateTmpFileFor(t, testdata.FS(), "seed.img"),
	})
	assert.Empty(t, iso)
	require.Error(t, err, "cloud-init needs user-data and meta-data files to work")
}

func TestGenerateCloudInitImageWithNoFile(t *testing.T) {
	iso, err := applevf.GenerateCloudInitImage(t.Context(), []string{})
	assert.Empty(t, iso)
	require.NoError(t, err)
}
