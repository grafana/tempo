// Package uuid provides a UUID type that can be used in protocol buffer
// messages.  It only wraps the google/uuid package and implements a couple
// helpers to make creating new instances simpler.
package backend

import (
	"encoding/json"

	google_uuid "github.com/google/uuid"
)

var (
	_ json.Unmarshaler = &UUID{}
	_ json.Marshaler   = &UUID{}
)

type UUID google_uuid.UUID

func NewUUID() UUID {
	return UUID(google_uuid.New())
}

func MustParse(s string) UUID {
	return UUID(google_uuid.MustParse(s))
}

func ParseUUID(s string) (UUID, error) {
	u, err := google_uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}

	return UUID(u), nil
}

func (u UUID) String() string {
	return ((google_uuid.UUID)(u)).String()
}

func (u UUID) Marshal() ([]byte, error) {
	return ((google_uuid.UUID)(u)).MarshalBinary()
}

func (u UUID) MarshalTo(data []byte) (n int, err error) {
	return copy(data, u[:]), nil
}

func (u *UUID) Unmarshal(data []byte) error {
	err := ((*google_uuid.UUID)(u)).UnmarshalBinary(data)
	if err != nil {
		return err
	}

	return nil
}

func (u *UUID) Size() int {
	return 16
}

func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(((google_uuid.UUID)(u)).String())
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	uu, err := google_uuid.Parse(s)
	if err != nil {
		return err
	}

	*u = UUID(uu)

	return nil
}
