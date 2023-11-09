package listtomap

import (
	"encoding/json"

	"gopkg.in/yaml.v2"
)

type ListToMap map[string]struct{}

var (
	_ yaml.Marshaler   = (*ListToMap)(nil)
	_ yaml.Unmarshaler = (*ListToMap)(nil)
	_ json.Marshaler   = (*ListToMap)(nil)
	_ json.Unmarshaler = (*ListToMap)(nil)
)

// MarshalYAML implements the Marshal interface of the yaml pkg.
func (l ListToMap) MarshalYAML() (interface{}, error) {
	list := make([]string, 0, len(l))
	for k := range l {
		list = append(list, k)
	}

	if len(list) == 0 {
		return nil, nil
	}
	return list, nil
}

// UnmarshalYAML implements the Unmarshaler interface of the yaml pkg.
func (l *ListToMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	list := make([]string, 0)
	err := unmarshal(&list)
	if err != nil {
		return err
	}

	*l = make(map[string]struct{})
	for _, element := range list {
		(*l)[element] = struct{}{}
	}
	return nil
}

// MarshalJSON implements the Marshal interface of the json pkg.
func (l ListToMap) MarshalJSON() ([]byte, error) {
	list := make([]string, 0, len(l))
	for k := range l {
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

	*l = make(map[string]struct{})
	for _, element := range list {
		(*l)[element] = struct{}{}
	}
	return nil
}

func (l *ListToMap) GetMap() map[string]struct{} {
	if *l == nil {
		*l = map[string]struct{}{}
	}
	return *l
}

func Merge(m1, m2 ListToMap) ListToMap {
	merged := make(ListToMap)
	for k, _ := range m1 {
		merged[k] = struct{}{}
	}
	for k, _ := range m2 {
		merged[k] = struct{}{}
	}
	return merged
}
