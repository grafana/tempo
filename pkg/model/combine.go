package model

import (
	"bytes"
	"fmt"

	"github.com/grafana/tempo/tempodb/encoding/common"

	"github.com/grafana/tempo/pkg/model/trace"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/pkg/errors"
)

type objectCombiner struct{}

var ObjectCombiner = objectCombiner{}

var _ common.ObjectCombiner = (*objectCombiner)(nil)

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

	encoding, err := NewDecoder(dataEncoding)
	if err != nil {
		return nil, false, fmt.Errorf("error getting decoder: %w", err)
	}

	combinedBytes, err := encoding.Combine(objs...)
	if err != nil {
		return nil, false, fmt.Errorf("error combining: %w", err)
	}

	return combinedBytes, true, nil
}

// CombineForRead is a convenience method used for combining while reading a trace. Due its
// use of PrepareForRead() it is a costly method and should not be called during any write
// or compaction operations.
func CombineForRead(obj []byte, dataEncoding string, t *tempopb.Trace) (*tempopb.Trace, error) {
	decoder, err := NewDecoder(dataEncoding)
	if err != nil {
		return nil, fmt.Errorf("error getting decoder: %w", err)
	}

	objTrace, err := decoder.PrepareForRead(obj)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling obj (%s): %w", dataEncoding, err)
	}

	combined, _ := trace.CombineTraceProtos(objTrace, t)

	return combined, nil
}
