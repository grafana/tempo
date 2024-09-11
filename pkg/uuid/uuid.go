// Package uuid provides a UUID type that can be used in protocol buffer
// messages.  It only wraps the google/uuid package and implements a couple
// helpers to make creating new instances simpler.
package uuid

import (
	"encoding/json"

	google_uuid "github.com/google/uuid"
)

var (
	_ json.Unmarshaler = &UUID{}
	_ json.Marshaler   = &UUID{}
)

type UUID struct {
	google_uuid.UUID
}

func New() UUID {
	return UUID{google_uuid.New()}
}

func MustParse(s string) UUID {
	return UUID{google_uuid.MustParse(s)}
}

func Parse(s string) (UUID, error) {
	u, err := google_uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}

	return UUID{u}, nil
}

func From(u google_uuid.UUID) UUID {
	return UUID{u}
}

func (u UUID) Marshal() ([]byte, error) {
	return u.MarshalBinary()
}

func (u *UUID) MarshalTo(data []byte) (n int, err error) {
	b, err := u.MarshalBinary()
	if err != nil {
		return 0, err
	}

	return copy(data, b), nil
}

func (u *UUID) Unmarshal(data []byte) error {
	err := u.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	return nil
}

func (u *UUID) Size() int {
	return 16
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	uu := google_uuid.UUID{}
	err := json.Unmarshal(data, &uu)
	if err != nil {
		return err
	}

	u.UUID = uu

	return nil
}
