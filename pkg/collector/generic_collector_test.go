package collector

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenericCollector_Collect tests the Collect method.
func TestGenericCollectorCollect(t *testing.T) {
	collector := NewGenericCollector[int]()

	collector.Collect(1)

	require.Equal(t, []int{1}, collector.Values())
}

// TestGenericCollector_MultipleCollects tests adding multiple batches of values.
func TestGenericCollectorMultipleCollects(t *testing.T) {
	collector := NewGenericCollector[int]()

	collector.Collect(1)
	collector.Collect(5)

	require.Equal(t, []int{1, 5}, collector.Values())
}

// TestGenericCollector_String tests the GenericCollector with string values.
func TestGenericCollectorString(t *testing.T) {
	stringCollector := NewGenericCollector[string]()

	stringCollector.Collect("a")
	stringCollector.Collect("b")
	stringCollector.Collect("c")

	require.Equal(t, []string{"a", "b", "c"}, stringCollector.Values())
}

// TestGenericCollectorEmpty tests the behavior of Results on an empty collector.
func TestGenericCollectorEmpty(t *testing.T) {
	collector := NewGenericCollector[int]()

	require.Equal(t, []int{}, collector.Values())
}
