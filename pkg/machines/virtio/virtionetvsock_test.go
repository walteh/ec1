package virtio_test

// func sourceSocketPath(t *testing.T, sourcePathLen int) (string, func()) {
// 	// the 't.sock' name is chosen to be shorter than what
// 	// localUnixSocketPath will generate so that the source socket path
// 	// will not exceed the 104 byte limit while the destination socket path
// 	// will, and will trigger an error
// 	const sourceSocketName = "t.sock"
// 	tmpDir := "/tmp"
// 	subDirLen := sourcePathLen - len(tmpDir) - 2*len("/") - len(sourceSocketName) - 1
// 	subDir := filepath.Join(tmpDir, strings.Repeat("a", subDirLen))
// 	err := os.Mkdir(subDir, 0700)
// 	require.NoError(t, err)
// 	unixSocketPath := filepath.Join(subDir, sourceSocketName)
// 	require.Equal(t, len(unixSocketPath), sourcePathLen-1)
// 	return unixSocketPath, func() { os.RemoveAll(subDir) }

// }
// func testConnectUnixgram(t *testing.T, ctx context.Context, sourcePathLen int) error {
// 	t.Helper()
// 	unixSocketPath, closer := sourceSocketPath(t, sourcePathLen)
// 	defer closer()

// 	addr, err := net.ResolveUnixAddr("unixgram", unixSocketPath)
// 	require.NoError(t, err)

// 	l, err := net.ListenUnixgram("unixgram", addr)
// 	require.NoError(t, err)

// 	defer l.Close()

// 	dev := &virtio.VirtioNetViaUnixSocket{
// 		UnixSocketPath: unixSocketPath,
// 	}

// 	_, err = dev.TransformToVirtioNet(ctx)
// 	return err
// }

// func TestConnectUnixPath(t *testing.T) {
// 	ctx := t.Context()
// 	t.Run("Successful connection - no error", func(t *testing.T) {
// 		// 50 is an arbitrary number, small enough for the 104 bytes limit not to be exceeded
// 		err := testConnectUnixgram(t, ctx, 50)
// 		require.NoError(t, err)
// 	})

// 	t.Run("Failed connection - End socket longer than 104 bytes", func(t *testing.T) {
// 		err := testConnectUnixgram(t, ctx, virtio.MaxUnixgramPathLen)
// 		// It should return an error
// 		require.Error(t, err)
// 		require.ErrorContains(t, err, "is too long")
// 	})
// }

// func TestLocalUnixSocketPath(t *testing.T) {
// 	t.Run("Success case - Creates temporary socket path", func(t *testing.T) {
// 		// Retrieve HOME env variable (used by the os.UserHomeDir)
// 		socketDir := t.TempDir()

// 		path, err := virtio.LocalUnixSocketPath(socketDir)

// 		// Assert successful execution
// 		require.NoError(t, err)

// 		// Check if path starts with the expected prefix
// 		require.Truef(t, strings.HasPrefix(path, socketDir), "Path doesn't start with expected prefix: %v", path)

// 		// Check if path ends with a socket extension
// 		require.Equalf(t, ".sock", filepath.Ext(path), "Path doesn't end with .sock extension: %v, ext is %v", path, filepath.Ext(path))
// 	})
// }
