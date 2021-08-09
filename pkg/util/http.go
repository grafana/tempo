package util

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const (
	TraceIDVar              = "traceID"
	AcceptHeaderKey         = "Accept"
	ProtobufTypeHeaderValue = "application/protobuf"
	JSONTypeHeaderValue     = "application/json"
)

func ParseTraceID(r *http.Request) ([]byte, error) {
	vars := mux.Vars(r)
	traceID, ok := vars[TraceIDVar]
	if !ok {
		return nil, fmt.Errorf("please provide a traceID")
	}

	byteID, err := hexStringToTraceID(traceID)
	if err != nil {
		return nil, err
	}

	return byteID, nil
}

func hexStringToTraceID(id string) ([]byte, error) {
	// The encoding/hex package does not handle non-hex characters.
	// Ensure the ID has only the proper characters
	for pos, idChar := range strings.Split(id, "") {
		if (idChar >= "a" && idChar <= "f") ||
			(idChar >= "A" && idChar <= "F") ||
			(idChar >= "0" && idChar <= "9") {
			continue
		} else {
			return nil, fmt.Errorf("trace IDs can only contain hex characters: invalid character '%s' at position %d", idChar, pos+1)
		}
	}

	// the encoding/hex package does not like odd length strings.
	// just append a bit here
	if len(id)%2 == 1 {
		id = "0" + id
	}

	byteID, err := hex.DecodeString(id)
	if err != nil {
		return nil, err
	}

	size := len(byteID)
	if size > 16 {
		return nil, errors.New("trace IDs can't be larger than 128 bits")
	}
	if size < 16 {
		byteID = append(make([]byte, 16-size), byteID...)
	}

	return byteID, nil
}
