package backend

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type marshalTest struct {
	Test Encoding
}

func TestUnmarshalMarshalYaml(t *testing.T) {
	for _, enc := range SupportedEncoding {
		expected := marshalTest{
			Test: enc,
		}
		actual := marshalTest{}

		buff, err := yaml.Marshal(expected)
		assert.NoError(t, err)
		err = yaml.Unmarshal(buff, &actual)
		assert.NoError(t, err)

		assert.Equal(t, expected, actual)
	}
}

func TestUnmarshalMarshalJson(t *testing.T) {
	for _, enc := range SupportedEncoding {
		expected := marshalTest{
			Test: enc,
		}
		actual := marshalTest{}

		buff, err := json.Marshal(expected)
		assert.NoError(t, err)
		err = json.Unmarshal(buff, &actual)
		assert.NoError(t, err)

		assert.Equal(t, expected, actual)
	}
}
