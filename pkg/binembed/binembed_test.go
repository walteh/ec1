package binembed

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/mholt/archives"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestXZData creates test data compressed with XZ
func createTestXZData(t *testing.T, data []byte) []byte {
	var buf bytes.Buffer
	xz := &archives.Xz{}
	writer, err := xz.OpenWriter(&buf)
	require.NoError(t, err)
	
	_, err = writer.Write(data)
	require.NoError(t, err)
	
	err = writer.Close()
	require.NoError(t, err)
	
	return buf.Bytes()
}

// createTestXZDataB creates test data compressed with XZ for benchmarks
func createTestXZDataB(b *testing.B, data []byte) []byte {
	var buf bytes.Buffer
	xz := &archives.Xz{}
	writer, err := xz.OpenWriter(&buf)
	require.NoError(b, err)
	
	_, err = writer.Write(data)
	require.NoError(b, err)
	
	err = writer.Close()
	require.NoError(b, err)
	
	return buf.Bytes()
}

func TestRegisterXZ(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for compression")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-1"
	
	RegisterXZ(checksum, compressedData)
	
	assert.True(t, IsRegistered(checksum))
	
	checksums := GetRegisteredChecksums()
	assert.Contains(t, checksums, checksum)
	assert.Len(t, checksums, 1)
}

func TestGetDecompressed(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for decompression")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-2"
	
	RegisterXZ(checksum, compressedData)
	
	reader, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	decompressedData, err := io.ReadAll(reader)
	require.NoError(t, err)
	
	assert.Equal(t, testData, decompressedData)
}

func TestGetDecompressedNotFound(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	_, err := GetDecompressed("non-existent-checksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary not found")
}

func TestMustGetDecompressed(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for must get")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-3"
	
	RegisterXZ(checksum, compressedData)
	
	reader := MustGetDecompressed(checksum)
	decompressedData, err := io.ReadAll(reader)
	require.NoError(t, err)
	
	assert.Equal(t, testData, decompressedData)
}

func TestMustGetDecompressedPanic(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	assert.Panics(t, func() {
		MustGetDecompressed("non-existent-checksum")
	})
}

func TestConcurrentAccess(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for concurrent access")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-concurrent"
	
	RegisterXZ(checksum, compressedData)
	
	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make([][]byte, numGoroutines)
	errors := make([]error, numGoroutines)
	
	// Launch multiple goroutines to access the same binary concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			reader, err := GetDecompressed(checksum)
			errors[index] = err
			if err == nil {
				data, readErr := io.ReadAll(reader)
				if readErr != nil {
					errors[index] = readErr
				} else {
					results[index] = data
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify all goroutines succeeded and got the same data
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "goroutine %d should not have error", i)
		assert.Equal(t, testData, results[i], "goroutine %d should get correct data", i)
	}
}

func TestLazyDecompression(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for lazy decompression")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-lazy"
	
	RegisterXZ(checksum, compressedData)
	
	// Verify the binary is registered but not yet decompressed
	registryMutex.RLock()
	entry := registry[checksum]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry)
	assert.Nil(t, entry.decompressed, "should not be decompressed yet")
	
	// First access should trigger decompression
	reader1, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	data1, err := io.ReadAll(reader1)
	require.NoError(t, err)
	assert.Equal(t, testData, data1)
	
	// Verify decompression happened and data is cached
	registryMutex.RLock()
	entry = registry[checksum]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry.decompressed, "should be decompressed now")
	assert.Equal(t, testData, entry.decompressed)
	
	// Second access should use cached data
	reader2, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	data2, err := io.ReadAll(reader2)
	require.NoError(t, err)
	assert.Equal(t, testData, data2)
}

func TestPreloadAsync(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for preload async")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-preload"
	
	RegisterXZ(checksum, compressedData)
	
	// Start preloading
	PreloadAsync(checksum)
	
	// Give some time for background decompression
	time.Sleep(100 * time.Millisecond)
	
	// Verify decompression happened in background
	registryMutex.RLock()
	entry := registry[checksum]
	registryMutex.RUnlock()
	
	// The decompression should have completed by now
	assert.NotNil(t, entry.decompressed, "should be decompressed by preload")
	assert.Equal(t, testData, entry.decompressed)
	
	// Subsequent access should be immediate
	reader, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

func TestPreloadAsyncAll(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Register multiple binaries
	testData1 := []byte("test data 1")
	testData2 := []byte("test data 2")
	compressedData1 := createTestXZData(t, testData1)
	compressedData2 := createTestXZData(t, testData2)
	checksum1 := "test-checksum-all-1"
	checksum2 := "test-checksum-all-2"
	
	RegisterXZ(checksum1, compressedData1)
	RegisterXZ(checksum2, compressedData2)
	
	// Preload all
	PreloadAsyncAll()
	
	// Give some time for background decompression
	time.Sleep(200 * time.Millisecond)
	
	// Verify both binaries were decompressed
	registryMutex.RLock()
	entry1 := registry[checksum1]
	entry2 := registry[checksum2]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry1.decompressed, "binary 1 should be decompressed")
	assert.NotNil(t, entry2.decompressed, "binary 2 should be decompressed")
	assert.Equal(t, testData1, entry1.decompressed)
	assert.Equal(t, testData2, entry2.decompressed)
}

func TestIsRegistered(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for is registered")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-registered"
	
	assert.False(t, IsRegistered(checksum), "should not be registered initially")
	
	RegisterXZ(checksum, compressedData)
	
	assert.True(t, IsRegistered(checksum), "should be registered after RegisterXZ")
	assert.False(t, IsRegistered("non-existent"), "non-existent should not be registered")
}

func TestGetRegisteredChecksums(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	assert.Empty(t, GetRegisteredChecksums(), "should be empty initially")
	
	testData1 := []byte("test data 1")
	testData2 := []byte("test data 2")
	compressedData1 := createTestXZData(t, testData1)
	compressedData2 := createTestXZData(t, testData2)
	checksum1 := "test-checksum-list-1"
	checksum2 := "test-checksum-list-2"
	
	RegisterXZ(checksum1, compressedData1)
	RegisterXZ(checksum2, compressedData2)
	
	checksums := GetRegisteredChecksums()
	assert.Len(t, checksums, 2)
	assert.Contains(t, checksums, checksum1)
	assert.Contains(t, checksums, checksum2)
}

// Benchmark tests to verify performance improvements
func BenchmarkGetDecompressedConcurrent(b *testing.B) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := bytes.Repeat([]byte("benchmark test data "), 1000) // ~20KB
	compressedData := createTestXZDataB(b, testData)
	checksum := "benchmark-checksum"
	
	RegisterXZ(checksum, compressedData)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader, err := GetDecompressed(checksum)
			if err != nil {
				b.Fatal(err)
			}
			_, err = io.ReadAll(reader)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGetDecompressedSequential(b *testing.B) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := bytes.Repeat([]byte("benchmark test data "), 1000) // ~20KB
	compressedData := createTestXZDataB(b, testData)
	checksum := "benchmark-checksum-seq"
	
	RegisterXZ(checksum, compressedData)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := GetDecompressed(checksum)
		if err != nil {
			b.Fatal(err)
		}
		_, err = io.ReadAll(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPreloadSync(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for preload sync")
	compressedData := createTestXZData(t, testData)
	checksum := "test-checksum-preload-sync"
	
	RegisterXZ(checksum, compressedData)
	
	// Verify the binary is registered but not yet decompressed
	registryMutex.RLock()
	entry := registry[checksum]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry)
	assert.Nil(t, entry.decompressed, "should not be decompressed yet")
	
	// Synchronously preload the binary
	err := PreloadSync(checksum)
	require.NoError(t, err, "PreloadSync should succeed")
	
	// Verify decompression happened immediately
	registryMutex.RLock()
	entry = registry[checksum]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry.decompressed, "should be decompressed after PreloadSync")
	assert.Equal(t, testData, entry.decompressed)
	
	// Subsequent access should be immediate
	reader, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Equal(t, testData, data)
}

func TestPreloadSyncNotFound(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	err := PreloadSync("non-existent-checksum")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary not found")
}

func TestPreloadAllSync(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Register multiple binaries
	testData1 := []byte("test data 1 for sync all")
	testData2 := []byte("test data 2 for sync all")
	testData3 := []byte("test data 3 for sync all")
	compressedData1 := createTestXZData(t, testData1)
	compressedData2 := createTestXZData(t, testData2)
	compressedData3 := createTestXZData(t, testData3)
	checksum1 := "test-checksum-sync-all-1"
	checksum2 := "test-checksum-sync-all-2"
	checksum3 := "test-checksum-sync-all-3"
	
	RegisterXZ(checksum1, compressedData1)
	RegisterXZ(checksum2, compressedData2)
	RegisterXZ(checksum3, compressedData3)
	
	// Verify none are decompressed yet
	registryMutex.RLock()
	entry1 := registry[checksum1]
	entry2 := registry[checksum2]
	entry3 := registry[checksum3]
	registryMutex.RUnlock()
	
	assert.Nil(t, entry1.decompressed, "binary 1 should not be decompressed yet")
	assert.Nil(t, entry2.decompressed, "binary 2 should not be decompressed yet")
	assert.Nil(t, entry3.decompressed, "binary 3 should not be decompressed yet")
	
	// Measure time for concurrent preloading
	startTime := time.Now()
	err := PreloadAllSync()
	duration := time.Since(startTime)
	
	require.NoError(t, err, "PreloadAllSync should succeed")
	t.Logf("Concurrent preload took: %v", duration)
	
	// Verify all binaries were decompressed
	registryMutex.RLock()
	entry1 = registry[checksum1]
	entry2 = registry[checksum2]
	entry3 = registry[checksum3]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry1.decompressed, "binary 1 should be decompressed")
	assert.NotNil(t, entry2.decompressed, "binary 2 should be decompressed")
	assert.NotNil(t, entry3.decompressed, "binary 3 should be decompressed")
	assert.Equal(t, testData1, entry1.decompressed)
	assert.Equal(t, testData2, entry2.decompressed)
	assert.Equal(t, testData3, entry3.decompressed)
	
	// Verify all can be accessed immediately
	for i, checksum := range []string{checksum1, checksum2, checksum3} {
		reader, err := GetDecompressed(checksum)
		require.NoError(t, err, "should be able to get binary %d", i+1)
		
		data, err := io.ReadAll(reader)
		require.NoError(t, err, "should be able to read binary %d", i+1)
		
		expectedData := [][]byte{testData1, testData2, testData3}[i]
		assert.Equal(t, expectedData, data, "binary %d should have correct data", i+1)
	}
}

func TestPreloadAllSyncConcurrentPerformance(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Create larger test data to make timing differences more apparent
	largeTestData := bytes.Repeat([]byte("performance test data "), 5000) // ~100KB
	
	// Register multiple binaries
	const numBinaries = 5
	checksums := make([]string, numBinaries)
	for i := 0; i < numBinaries; i++ {
		checksum := fmt.Sprintf("perf-test-checksum-%d", i)
		checksums[i] = checksum
		compressedData := createTestXZData(t, largeTestData)
		RegisterXZ(checksum, compressedData)
	}
	
	// Test concurrent preloading
	startTime := time.Now()
	err := PreloadAllSync()
	concurrentDuration := time.Since(startTime)
	
	require.NoError(t, err, "PreloadAllSync should succeed")
	t.Logf("Concurrent preload of %d binaries took: %v", numBinaries, concurrentDuration)
	
	// Clear registry and register again for sequential test
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	for i := 0; i < numBinaries; i++ {
		checksum := fmt.Sprintf("perf-test-checksum-seq-%d", i)
		compressedData := createTestXZData(t, largeTestData)
		RegisterXZ(checksum, compressedData)
	}
	
	// Test sequential preloading for comparison
	startTime = time.Now()
	registryMutex.RLock()
	seqChecksums := make([]string, 0, len(registry))
	for checksum := range registry {
		seqChecksums = append(seqChecksums, checksum)
	}
	registryMutex.RUnlock()
	
	for _, checksum := range seqChecksums {
		err := PreloadSync(checksum)
		require.NoError(t, err)
	}
	sequentialDuration := time.Since(startTime)
	
	t.Logf("Sequential preload of %d binaries took: %v", numBinaries, sequentialDuration)
	
	// Concurrent should be faster than sequential (with some tolerance for test variance)
	// We expect at least some speedup, but allow for overhead and test environment variance
	if concurrentDuration < sequentialDuration {
		t.Logf("✓ Concurrent preload was faster: %v vs %v (%.1fx speedup)", 
			concurrentDuration, sequentialDuration, 
			float64(sequentialDuration)/float64(concurrentDuration))
	} else {
		t.Logf("⚠ Concurrent preload was not faster: %v vs %v (this may be due to test environment)", 
			concurrentDuration, sequentialDuration)
		// Don't fail the test as this could be due to test environment limitations
	}
}

func TestPreloadAllSyncEmpty(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Should succeed even with no registered binaries
	err := PreloadAllSync()
	assert.NoError(t, err, "PreloadAllSync should succeed with empty registry")
}

func TestPreloadAllSyncWithError(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Register a binary with invalid compressed data
	invalidData := []byte("this is not valid XZ data")
	checksum := "test-checksum-invalid"
	
	RegisterXZ(checksum, invalidData)
	
	// PreloadAllSync should fail
	err := PreloadAllSync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "preloading binary")
	assert.Contains(t, err.Error(), checksum)
}

// Benchmark tests to verify performance of sync functions
func BenchmarkPreloadSync(b *testing.B) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := bytes.Repeat([]byte("benchmark sync test data "), 1000) // ~25KB
	compressedData := createTestXZDataB(b, testData)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checksum := fmt.Sprintf("benchmark-sync-%d", i)
		RegisterXZ(checksum, compressedData)
		
		err := PreloadSync(checksum)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPreloadAllSync(b *testing.B) {
	testData := bytes.Repeat([]byte("benchmark sync all test data "), 1000) // ~30KB
	compressedData := createTestXZDataB(b, testData)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Clear registry for each iteration
		registryMutex.Lock()
		registry = make(map[string]*binaryEntry)
		registryMutex.Unlock()
		
		// Register multiple binaries
		for j := 0; j < 5; j++ {
			checksum := fmt.Sprintf("benchmark-sync-all-%d-%d", i, j)
			RegisterXZ(checksum, compressedData)
		}
		
		err := PreloadAllSync()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestRegisterRaw(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for raw registration")
	checksum := "test-checksum-raw"
	
	RegisterRaw(checksum, testData)
	
	assert.True(t, IsRegistered(checksum))
	
	// Verify the entry is marked as uncompressed
	registryMutex.RLock()
	entry := registry[checksum]
	registryMutex.RUnlock()
	
	assert.NotNil(t, entry)
	assert.False(t, entry.isCompressed, "should be marked as uncompressed")
	assert.Equal(t, testData, entry.decompressed, "should have data immediately available")
	
	checksums := GetRegisteredChecksums()
	assert.Contains(t, checksums, checksum)
}

func TestGetDecompressedRaw(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	testData := []byte("test data for raw decompression")
	checksum := "test-checksum-raw-decomp"
	
	RegisterRaw(checksum, testData)
	
	reader, err := GetDecompressed(checksum)
	require.NoError(t, err)
	
	decompressedData, err := io.ReadAll(reader)
	require.NoError(t, err)
	
	assert.Equal(t, testData, decompressedData)
}

func TestMixedCompressedAndRaw(t *testing.T) {
	// Clear registry for test isolation
	registryMutex.Lock()
	registry = make(map[string]*binaryEntry)
	registryMutex.Unlock()
	
	// Register both compressed and raw data
	compressedData := []byte("compressed test data")
	rawData := []byte("raw test data")
	
	compressedXZ := createTestXZData(t, compressedData)
	compressedChecksum := "test-mixed-compressed"
	rawChecksum := "test-mixed-raw"
	
	RegisterXZ(compressedChecksum, compressedXZ)
	RegisterRaw(rawChecksum, rawData)
	
	// Test compressed data
	reader1, err := GetDecompressed(compressedChecksum)
	require.NoError(t, err)
	
	result1, err := io.ReadAll(reader1)
	require.NoError(t, err)
	assert.Equal(t, compressedData, result1)
	
	// Test raw data
	reader2, err := GetDecompressed(rawChecksum)
	require.NoError(t, err)
	
	result2, err := io.ReadAll(reader2)
	require.NoError(t, err)
	assert.Equal(t, rawData, result2)
	
	// Verify both are registered
	checksums := GetRegisteredChecksums()
	assert.Len(t, checksums, 2)
	assert.Contains(t, checksums, compressedChecksum)
	assert.Contains(t, checksums, rawChecksum)
} 