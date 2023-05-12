package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newLabelValuesWithMax(t *testing.T) {
	labelValues := newLabelValueComboWithMax([]string{"service", "name"}, []string{"abc", "abcdef"}, 5, 5)

	assert.Equal(t, []string{"abc", "abcde"}, labelValues.getValues())
	assert.Equal(t, []string{"servi", "name"}, labelValues.getNames())
}

func Test_newLabelValuesWithMax_zeroLength(t *testing.T) {
	labelValues := newLabelValueComboWithMax([]string{"service", "name"}, []string{"abc", "abcdef"}, 0, 0)

	assert.Equal(t, []string{"abc", "abcdef"}, labelValues.getValues())
	assert.Equal(t, []string{"service", "name"}, labelValues.getNames())
}
