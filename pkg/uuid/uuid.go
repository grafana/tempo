package uuid

import (
	google_uuid "github.com/google/uuid"
)

type UUID struct {
	google_uuid.UUID
}

func (u *UUID) Size() int {
	return 16
}

func (u *UUID) MarshalTo(data []byte) (n int, err error) {
	b, err := u.UUID.MarshalBinary()
	if err != nil {
		return 0, err
	}

	return copy(data, b), nil
}

func (u *UUID) Unmarshal(data []byte) error {
	err := u.UUID.UnmarshalBinary(data)
	if err != nil {
		return err
	}

	return nil
}
