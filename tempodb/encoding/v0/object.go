package v0

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/grafana/tempo/tempodb/encoding/common"
)

const (
	uint32Size = 4
)

/*
	|          -- totalLength --                   |
	| total length | id length | id | object bytes |
*/

func MarshalObjectToWriter(id common.ID, b []byte, w io.Writer) (int, error) {
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

func unmarshalObjectFromReader(r io.Reader) (common.ID, []byte, error) {
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

func unmarshalAndAdvanceBuffer(buffer []byte) ([]byte, common.ID, []byte, error) {
	var totalLength uint32

	if len(buffer) == 0 {
		return nil, nil, nil, io.EOF
	}

	if len(buffer) < uint32Size {
		return nil, nil, nil, fmt.Errorf("unable to read totalLength from buffer")
	}
	totalLength = binary.LittleEndian.Uint32(buffer)
	buffer = buffer[uint32Size:]

	var idLength uint32
	if len(buffer) < uint32Size {
		return nil, nil, nil, fmt.Errorf("unable to read idLength from buffer")
	}
	idLength = binary.LittleEndian.Uint32(buffer)
	buffer = buffer[uint32Size:]

	restLength := totalLength - uint32Size*2
	if uint32(len(buffer)) < restLength {
		return nil, nil, nil, fmt.Errorf("unable to read id/object from buffer")
	}

	bytesID := buffer[:idLength]
	bytesObject := buffer[idLength:restLength]

	buffer = buffer[restLength:]

	return buffer, bytesID, bytesObject, nil
}
