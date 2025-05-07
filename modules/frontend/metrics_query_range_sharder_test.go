package frontend

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzExemplarsPerShard(f *testing.F) {
	f.Add(uint32(1), uint32(10))  // total = 1, exemplars = 10
	f.Add(uint32(100), uint32(1)) // total = 100, exemplars = 1
	f.Add(uint32(10), uint32(0))  // total = 10, exemplars = 0

	s := &queryRangeSharder{}

	f.Fuzz(func(t *testing.T, total uint32, exemplars uint32) {
		result := s.exemplarsPerShard(total, exemplars)

		if exemplars == 0 || total == 0 {
			assert.Equal(t, uint32(0), result, "if exemplars is 0 or total is 0, result should be 0")
		} else {
			assert.Greater(t, result, uint32(0), "result should be greater than 0")
		}
	})
}
