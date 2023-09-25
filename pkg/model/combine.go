package model

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
)

type objectCombiner struct{}

type ObjectCombiner interface {
	Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error)
}

var StaticCombiner = objectCombiner{}

// Combine implements tempodb/encoding/common.ObjectCombiner
func (o objectCombiner) Combine(dataEncoding string, objs ...[]byte) ([]byte, bool, error) {
	if len(objs) <= 0 {
		return nil, false, errors.New("no objects provided")
	}

	// check to see if we need to combine
	needCombine := false
	for i := 1; i < len(objs); i++ {
		if !bytes.Equal(objs[0], objs[i]) {
			needCombine = true
			break
		}
	}

	if !needCombine {
		return objs[0], false, nil
	}

	encoding, err := NewObjectDecoder(dataEncoding)
	if err != nil {
		return nil, false, fmt.Errorf("error getting decoder: %w", err)
	}

	combinedBytes, err := encoding.Combine(objs...)
	if err != nil {
		return nil, false, fmt.Errorf("error combining: %w", err)
	}

	return combinedBytes, true, nil
}
