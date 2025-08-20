package work

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

// TestAtomicWriteFileStress applies extreme concurrent pressure to verify robustness
func TestAtomicWriteFileStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	tmpDir := t.TempDir()
	targetFile := filepath.Join(tmpDir, "stress.json")

	// Large data to increase write time and chances of race conditions
	largeData := make([]byte, 1024*10) // 10KB
	for i := range largeData {
		largeData[i] = byte('A' + (i % 26))
	}
	testData := []byte(fmt.Sprintf(`{"large_data": "%s", "id": "test"}`, string(largeData)))

	// High concurrency stress test
	var wg sync.WaitGroup
	numGoroutines := 100
	writesPerGoroutine := 5

	for i := range numGoroutines {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := range writesPerGoroutine {
				// Use different temp prefixes to test CreateTemp uniqueness
				prefix := fmt.Sprintf("stress_%d_%d.json", goroutineID, j)
				err := atomicWriteFile(testData, targetFile, prefix)
				require.NoError(t, err)

				// No sleep - maximum pressure
			}
		}(i)
	}

	wg.Wait()

	// Verify final file integrity
	finalData, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	require.NotEmpty(t, finalData, "Final file should not be empty")

	// Should be valid JSON
	var result map[string]any
	err = jsoniter.Unmarshal(finalData, &result)
	require.NoError(t, err, "Final file should contain valid JSON after stress test")

	// Verify it's our expected data
	require.Equal(t, "test", result["id"], "Final file should contain our test data")

	// Critical: No temp files should remain after stress test
	files, err := filepath.Glob(filepath.Join(tmpDir, "*tmp*"))
	require.NoError(t, err)
	require.Empty(t, files, "No temporary files should remain after stress test - all should be cleaned up")

	t.Logf("Stress test completed: %d goroutines × %d writes = %d total operations",
		numGoroutines, writesPerGoroutine, numGoroutines*writesPerGoroutine)
}
