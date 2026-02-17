package traceql

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNaNHandling tests that NaN values don't corrupt aggregation results.
// This can happen because NaN + 1 is NaN
func TestNaNHandling(t *testing.T) {
	t.Run("sumOverTime skips NaN", func(t *testing.T) {
		sumFunc := sumOverTime()

		// Start with NaN (initial state)
		result := sumFunc(math.NaN(), 10.0)
		assert.Equal(t, 10.0, result)

		// Add valid value
		result = sumFunc(result, 20.0)
		assert.Equal(t, 30.0, result)

		// Add NaN - should be skipped
		result = sumFunc(result, math.NaN())
		assert.Equal(t, 30.0, result)

		// Add another valid value
		result = sumFunc(result, 5.0)
		assert.Equal(t, 35.0, result)
	})

	t.Run("minOverTime handles NaN", func(t *testing.T) {
		minFunc := minOverTime()

		// Start with NaN
		result := minFunc(math.NaN(), 10.0)
		assert.Equal(t, 10.0, result)

		// NaN comparison should not affect result
		result = minFunc(result, math.NaN())
		assert.Equal(t, 10.0, result)

		// Normal min operation
		result = minFunc(result, 5.0)
		assert.Equal(t, 5.0, result)
	})

	t.Run("maxOverTime handles NaN", func(t *testing.T) {
		maxFunc := maxOverTime()

		// Start with NaN
		result := maxFunc(math.NaN(), 10.0)
		assert.Equal(t, 10.0, result)

		// NaN comparison should not affect result
		result = maxFunc(result, math.NaN())
		assert.Equal(t, 10.0, result)

		// Normal max operation
		result = maxFunc(result, 15.0)
		assert.Equal(t, 15.0, result)
	})
}
