package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	uint32Size = 4
)

type IterFunc func(id ID, b []byte) (bool, error)

/*
	|          -- totalLength --                   |
	| total length | id length | id | object bytes |
*/

func MarshalObjectToWriter(id ID, b []byte, w io.Writer) (int, error) {
	idLength := len(id)
	totalLength := len(b) + idLength + uint32Size*2

	err := binary.Write(w, binary.LittleEndian, uint32(totalLength))
	if err != nil {
		return 0, err
	}
	err = binary.Write(w, binary.LittleEndian, uint32(idLength))
	if err != nil {
		return 0, err
	}

	_, err = w.Write(id)
	if err != nil {
		return 0, err
	}
	_, err = w.Write(b)
	if err != nil {
		return 0, err
	}

	return totalLength, err
}

func UnmarshalObjectFromReader(r io.Reader) (ID, []byte, error) {
	var totalLength uint32
	err := binary.Read(r, binary.LittleEndian, &totalLength)
	if err == io.EOF {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	var idLength uint32
	err = binary.Read(r, binary.LittleEndian, &idLength)
	if err != nil {
		return nil, nil, err
	}

	protoLength := totalLength - uint32Size*2
	b := make([]byte, protoLength)
	readLength, err := r.Read(b)
	if err != nil {
		return nil, nil, err
	}
	if uint32(readLength) != protoLength {
		return nil, nil, fmt.Errorf("read %d but expected %d", readLength, protoLength)
	}

	bytesID := b[:idLength]
	bytesObject := b[idLength:]

	return bytesID, bytesObject, nil
}

func IterateObjects(reader io.Reader, fn IterFunc) error {
	for {
		id, b, err := UnmarshalObjectFromReader(reader)
		if err != nil {
			return err
		}
		if id == nil {
			// there are no more objects in the reader
			break
		}

		more, err := fn(id, b)
		if err != nil {
			return err
		}
		if !more {
			// the calling code doesn't need any more objects
			break
		}
	}

	return nil
}
