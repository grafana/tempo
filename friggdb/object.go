package friggdb

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"
)

/*
	|          -- totalLength --                   |
	| total length | id length | id | object bytes |
*/

func marshalObjectToWriter(id ID, p proto.Message, w io.Writer) (int, error) {
	b, err := proto.Marshal(p)
	if err != nil {
		return 0, err
	}

	idLength := len(id)
	totalLength := len(b) + idLength + uint32Size*2

	err = binary.Write(w, binary.LittleEndian, uint32(totalLength))
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

func unmarshalObjectFromReader(out proto.Message, r io.Reader) (ID, bool, error) {
	var totalLength uint32
	err := binary.Read(r, binary.LittleEndian, &totalLength)
	if err == io.EOF {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	var idLength uint32
	err = binary.Read(r, binary.LittleEndian, &idLength)
	if err != nil {
		return nil, false, err
	}

	protoLength := totalLength - uint32Size*2
	b := make([]byte, protoLength)
	readLength, err := r.Read(b)
	if err != nil {
		return nil, false, err
	}
	if uint32(readLength) != protoLength {
		return nil, false, fmt.Errorf("read %d but expected %d", readLength, protoLength)
	}

	bytesID := b[:idLength]
	bytesObject := b[idLength:]

	err = proto.Unmarshal(bytesObject, out)
	if err != nil {
		return nil, false, err
	}

	return bytesID, true, nil
}
