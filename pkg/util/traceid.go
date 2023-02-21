package util

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

func HexStringToTraceID(id string) ([]byte, error) {
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

// TraceIDToHexString converts a trace ID to its string representation and removes any leading zeros.
func TraceIDToHexString(byteID []byte) string {
	id := hex.EncodeToString(byteID)
	// remove leading zeros
	id = strings.TrimLeft(id, "0")
	return id
}

// SpanIDToHexString converts a span ID to its string representation and WITHOUT removing any leading zeros.
// If the id is < 16, left pad with 0s
func SpanIDToHexString(byteID []byte) string {
	id := hex.EncodeToString(byteID)
	id = strings.TrimLeft(id, "0")
	return fmt.Sprintf("%016s", id)
}

// EqualHexStringTraceIDs compares two trace ID strings and compares the
// resulting bytes after padding.  Returns true unless there is a reason not
// to.
func EqualHexStringTraceIDs(a, b string) (bool, error) {
	aa, err := HexStringToTraceID(a)
	if err != nil {
		return false, err
	}
	bb, err := HexStringToTraceID(b)
	if err != nil {
		return false, err
	}

	return bytes.Equal(aa, bb), nil
}

func PadTraceIDTo16Bytes(traceID []byte) []byte {
	if len(traceID) > 16 {
		return traceID[len(traceID)-16:]
	}

	if len(traceID) == 16 {
		return traceID
	}

	padded := make([]byte, 16)
	copy(padded[16-len(traceID):], traceID)

	return padded
}
