package client

import (
	"encoding/json"
	"fmt"
	"time"

	"go.yaml.in/yaml/v2"
)

type Duration struct {
	time.Duration
}

// MarshalJSON implements json.Marshaler.
func (d *Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(input []byte) error {
	var unmarshalledJSON interface{}

	err := json.Unmarshal(input, &unmarshalledJSON)
	if err != nil {
		return err
	}

	switch value := unmarshalledJSON.(type) {
	case float64:
		d.Duration = time.Duration(value)
	case string:
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid duration: %#v", unmarshalledJSON)
	}

	return nil
}

// MarshalYAML emits the duration as a flat scalar ("5m0s") instead of the
// nested "duration: 5m0s" map that yaml.v2's default reflection produces for
// the embedded time.Duration field.
func (d *Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalYAML parses a duration string ("5m0s") into d.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

var (
	_ json.Marshaler   = (*Duration)(nil)
	_ json.Unmarshaler = (*Duration)(nil)
	_ yaml.Marshaler   = (*Duration)(nil)
	_ yaml.Unmarshaler = (*Duration)(nil)
)
