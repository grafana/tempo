package collector

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistinctStringCollector(t *testing.T) {
	d := NewDistinctString(10)

	d.Collect("123")
	d.Collect("4567")
	d.Collect("890")
	d.Collect("11")

	require.True(t, d.Exceeded())
	require.Equal(t, []string{"123", "4567", "890"}, d.Strings())
}

func TestDistinctStringCollectorDiff(t *testing.T) {
	d := NewDistinctString(0)

	d.Collect("123")
	d.Collect("4567")

	require.Equal(t, []string{"123", "4567"}, d.Diff())
	require.Equal(t, []string{}, d.Diff())

	d.Collect("123")
	d.Collect("890")

	require.Equal(t, []string{"890"}, d.Diff())
	require.Equal(t, []string{}, d.Diff())
}

func TestDistinctStringCollectorIsSafe(t *testing.T) {
	d := NewDistinctString(0) // no limit
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				d.Collect(fmt.Sprintf("goroutine-%d-string-%d", id, j))
			}
		}(i)
	}
	wg.Wait()

	require.Equal(t, len(d.Strings()), 10*100)
	require.False(t, d.Exceeded())
}
