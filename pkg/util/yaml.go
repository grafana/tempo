package util

import (
	"bytes"

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

// YAMLUnmarshalStrict decodes YAML into out, returning an error if any keys in
// the input are not present in the destination. Replaces yaml.v2's
// UnmarshalStrict, which was removed in v3.
func YAMLUnmarshalStrict(in []byte, out interface{}) error {
	dec := yaml.NewDecoder(bytes.NewReader(in))
	dec.KnownFields(true)
	return dec.Decode(out)
}

// YAMLMarshalIndent2 marshals v to YAML using 2-space indentation, matching
// yaml.v2's default. yaml.v3 defaults to 4 spaces, which would change wire
// formats and snapshot test fixtures.
func YAMLMarshalIndent2(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
