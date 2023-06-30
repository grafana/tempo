package overrides

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoesNotContain(t *testing.T) {
	tc := []struct {
		l        []string
		item     string
		expected bool
	}{
		{
			[]string{},
			"test",
			true,
		},
		{
			[]string{"test-1", "test-2"},
			"test",
			true,
		},
		{
			[]string{"test-1", "test-2"},
			"test-2",
			false,
		},
	}
	for _, testCase := range tc {
		t.Run(fmt.Sprintf("%v contains %v", testCase.l, testCase.item), func(t *testing.T) {
			assert.Equal(t, testCase.expected, doesNotContain(testCase.l, testCase.item))
		})
	}
}
