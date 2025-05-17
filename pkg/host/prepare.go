package host

// func ShortTestTempDir(t *testing.T) string {
// 	// hash the test name, take the first 8 characters
// 	hash := sha256.Sum256([]byte(t.Name()))
// 	tmpdir := os.TempDir()
// 	testTmpDir := t.TempDir()
// 	dir := filepath.Join(tmpdir, fmt.Sprintf("t%x", hash[:4]), filepath.Base(testTmpDir))
// 	slog.InfoContext(t.Context(), "creating short test temp dir", "dir", dir)
// 	// clean the dir
// 	if _, err := os.Stat(dir); err == nil {
// 		os.RemoveAll(dir)
// 	}

// 	err := os.MkdirAll(dir, 0755)
// 	if err != nil {
// 		t.Fatalf("creating temp dir: %s", err)
// 	}
// 	t.Cleanup(func() {
// 		os.RemoveAll(dir)
// 	})
// 	RegisterRedactedLogValue(t, dir, "[short-tmp-dir]")
// 	return dir
// }

// func FullSetupOS(t *testing.T, prov OsProvider) *testVM {
// 	tmpDir := ShortTestTempDir(t)
// 	err := SetupOS(t, prov, tmpDir)
// 	if err != nil {
// 		t.Fatalf("setting up os: %s", err)
// 	}
// 	return NewTestVM(t, prov, tmpDir)
// }
