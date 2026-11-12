package attributes

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestSpaceSavingExactUnderCapacity(t *testing.T) {
	s := newSpaceSaving(8)
	for v, n := range map[string]int{"a": 5, "b": 3, "c": 3, "d": 1} {
		for range n {
			s.add(v, 1)
		}
	}

	require.False(t, s.evicted)
	require.Equal(t, 4, s.distinct())
	require.Equal(t, []valueCount{{"a", 5}, {"b", 3}, {"c", 3}, {"d", 1}}, s.top(10))
	require.Equal(t, []valueCount{{"a", 5}, {"b", 3}}, s.top(2))
}

// Past capacity a value carried by more than total/capacity must survive, with
// its count over-stated by at most that much.
func TestSpaceSavingKeepsHeavyHittersPastCapacity(t *testing.T) {
	const capacity, total = 4, 1000
	s := newSpaceSaving(capacity)
	for i := range total {
		if i%2 == 0 {
			s.add("hot", 1)
		} else {
			s.add(fmt.Sprintf("cold-%d", i), 1)
		}
	}

	require.True(t, s.evicted)
	top := s.top(1)
	require.Equal(t, "hot", top[0].value)
	require.GreaterOrEqual(t, top[0].count, uint64(total/2))
	require.LessOrEqual(t, top[0].count, uint64(total/2+total/capacity))
}

// A kept value must not share memory with the string it was added from.
func TestSpaceSavingCopiesKeys(t *testing.T) {
	s := newSpaceSaving(4)
	buf := []byte("abc")
	s.add(unsafe.String(&buf[0], len(buf)), 1)
	copy(buf, "xyz")

	require.Equal(t, []valueCount{{"abc", 1}}, s.top(1))
}
