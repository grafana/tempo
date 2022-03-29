package registry

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_hashLabelValues(t *testing.T) {
	testCases := []struct {
		v1, v2 []string
	}{
		{[]string{"foo"}, []string{"bar"}},
		{[]string{"foo", "bar"}, []string{"foob", "ar"}},
		{[]string{"foo", "bar"}, []string{"foobar", ""}},
		{[]string{"foo", "bar"}, []string{"foo\nbar", ""}},
		{[]string{"foo_", "bar"}, []string{"foo", "_bar"}},
		{[]string{"123", "456"}, []string{"1234", "56"}},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("%s - %s", strings.Join(testCase.v1, ","), strings.Join(testCase.v2, ",")), func(t *testing.T) {
			assert.NotEqual(t, hashLabelValues(testCase.v1), hashLabelValues(testCase.v2))
		})
	}
}
