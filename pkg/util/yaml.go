package util

import (
	"bytes"
	"io"

	"go.yaml.in/yaml/v3"
)

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

// YAMLUnmarshalStrict unmarshals YAML while rejecting unknown struct fields.
func YAMLUnmarshalStrict(in []byte, out interface{}) error {
	return YAMLNewStrictDecoder(bytes.NewReader(in)).Decode(out)
}

// YAMLNewStrictDecoder returns a YAML decoder configured to reject unknown struct fields.
func YAMLNewStrictDecoder(r io.Reader) *yaml.Decoder {
	decoder := yaml.NewDecoder(r)
	decoder.KnownFields(true)
	return decoder
}
