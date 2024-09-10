package uuid

import (
	"encoding/json"

	google_uuid "github.com/google/uuid"
)

type UUID struct {
	google_uuid.UUID
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
	return json.Marshal(u.UUID.String())
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
