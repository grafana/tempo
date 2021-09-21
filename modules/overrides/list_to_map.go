package overrides

import "encoding/json"

type ListToMap struct {
	m map[string]struct{}
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
