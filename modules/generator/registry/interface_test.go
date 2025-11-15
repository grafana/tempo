package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_newLabelValuesWithMax(t *testing.T) {
	labelValues := newLabelValueComboWithMax([]string{"service", "name"}, []string{"abc", "abcdef"}, 5, 5)

	assert.Equal(t, "abc", labelValues.labels.Get("servi"))
	assert.Equal(t, "abcde", labelValues.labels.Get("name"))
}

func Test_newLabelValuesWithMax_zeroLength(t *testing.T) {
	labelValues := newLabelValueComboWithMax([]string{"service", "name"}, []string{"abc", "abcdef"}, 0, 0)

	assert.Equal(t, "abc", labelValues.labels.Get("service"))
	assert.Equal(t, "abcdef", labelValues.labels.Get("name"))
}
