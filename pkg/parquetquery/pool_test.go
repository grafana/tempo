package parquetquery

import (
	"testing"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
)

func TestResultPoolRelease(t *testing.T) {
	pool := NewResultPool(5)

	// Get a result and populate it with data
	result1 := pool.Get()
	result1.AppendValue("key1", parquet.ValueOf("value1"))
	result1.AppendValue("key2", parquet.ValueOf(42))
	result1.AppendOtherValue("other1", "othervalue1")
	result1.AppendOtherValue("other2", 123)

	pool.Release(result1)

	result2 := pool.Get()

	// Verify the IteratorResult is properly cleared
	assert.Equal(t, 0, len(result2.Entries), "Entries should be empty in reused result")
	assert.Equal(t, 0, len(result2.OtherEntries), "OtherEntries should be empty in reused result")
}
