package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newLabelValuesWithMax(t *testing.T) {
	labelValues := newLabelValuesWithMax([]string{"abc", "abcdef"}, 5)

	assert.Equal(t, []string{"abc", "abcde"}, labelValues.getValues())
}

func Test_newLabelValuesWithMax_zeroLength(t *testing.T) {
	labelValues := newLabelValuesWithMax([]string{"abc", "abcdef"}, 0)

	assert.Equal(t, []string{"abc", "abcdef"}, labelValues.getValues())
}
