package util

import (
	"bytes"

	"go.yaml.in/yaml/v3"
)

// UnmarshalYAMLStrict unmarshals YAML while rejecting unknown struct fields.
func UnmarshalYAMLStrict(in []byte, out interface{}) error {
	decoder := yaml.NewDecoder(bytes.NewReader(in))
	decoder.KnownFields(true)
	return decoder.Decode(out)
}

// YAMLMarshalUnmarshal utility function that converts a YAML interface in a map
// doing marshal and unmarshal of the parameter
func YAMLMarshalUnmarshal(in interface{}) (map[interface{}]interface{}, error) {
	yamlBytes, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	object := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(yamlBytes, object); err != nil {
		return nil, err
	}

	return object, nil
}
