package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericCollector(t *testing.T) {
	collector := NewGenericCollector[int]()

	require.Equal(t, []int{}, collector.Values())

	collector.Collect(1)
	require.Equal(t, []int{1}, collector.Values())

	collector.Collect(5)
	collector.Collect(5)
	require.Equal(t, []int{1, 5, 5}, collector.Values())
}

func TestGenericCollectorString(t *testing.T) {
	collector := NewGenericCollector[string]()

	require.Equal(t, []string{}, collector.Values())

	collector.Collect("a")
	require.Equal(t, []string{"a"}, collector.Values())

	collector.Collect("b")
	collector.Collect("c")
	require.Equal(t, []string{"a", "b", "c"}, collector.Values())
}
