package overrides

import (
	"encoding/json"

	"gopkg.in/yaml.v2"
)

type ListToMap struct {
	m map[string]struct{}
}

var _ yaml.Marshaler = (*ListToMap)(nil)
var _ yaml.Unmarshaler = (*ListToMap)(nil)
var _ json.Marshaler = (*ListToMap)(nil)
var _ json.Unmarshaler = (*ListToMap)(nil)

// MarshalYAML implements the Marshal interface of the yaml pkg.
func (l ListToMap) MarshalYAML() (interface{}, error) {
	list := make([]string, 0)
	for k := range l.m {
		list = append(list, k)
	}

	b, err := yaml.Marshal(&list)
	return b, err
}

// UnmarshalYAML implements the Unmarshaler interface of the yaml pkg.
func (l *ListToMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	list := make([]string, 0)
	err := unmarshal(&list)
	if err != nil {
		return err
	}

	l.m = make(map[string]struct{})
	for _, element := range list {
		l.m[element] = struct{}{}
	}
	return nil
}

// MarshalJSON implements the Marshal interface of the json pkg.
func (l ListToMap) MarshalJSON() ([]byte, error) {
	list := make([]string, 0)
	for k := range l.m {
		list = append(list, k)
	}

	return json.Marshal(&list)
}

// UnmarshalJSON implements the Unmarshal interface of the json pkg.
func (l *ListToMap) UnmarshalJSON(b []byte) error {
	list := make([]string, 0)
	err := json.Unmarshal(b, &list)
	if err != nil {
		return err
	}

	l.m = make(map[string]struct{})
	for _, element := range list {
		l.m[element] = struct{}{}
	}
	return nil
}

func (l *ListToMap) GetMap() map[string]struct{} {
	return l.m
}
