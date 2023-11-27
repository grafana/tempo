package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brianvoe/gofakeit/v6"
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

func (d *Duration) Fake(faker *gofakeit.Faker) (any, error) {
	*d = Duration{Duration: time.Duration(faker.Second()) * time.Second}
	return *d, nil
}

var (
	_ json.Marshaler    = (*Duration)(nil)
	_ json.Unmarshaler  = (*Duration)(nil)
	_ gofakeit.Fakeable = (*Duration)(nil)
)
