package model

import (
	"github.com/go-kit/kit/log/level"

	"github.com/cortexproject/cortex/pkg/util/log"
)

type objectCombiner struct{}

var ObjectCombiner = objectCombiner{}

// Combine implements tempodb/encoding/common.ObjectCombiner
func (o objectCombiner) Combine(objA []byte, objB []byte, dataEncoding string) []byte {
	combinedTrace, _, err := CombineTraceBytes(objA, objB)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace
}

// CombineAcrossEncodings combines the two objects in the passed data encodings. It returns the objects
//  in dataEncodingA
func CombineAcrossEncodings(objA []byte, objB []byte, dataEncodingA string, dataEncodingB string) []byte {
	if dataEncodingA == dataEncodingB {
		return ObjectCombiner.Combine(objA, objB, dataEncodingA)
	}

	// todo(jpe):
	// - marshal both traces to *tempopb.Trace and combine at the proto level
	// - add cross data encoding tests

	combinedTrace, _, err := CombineTraceBytes(objA, objB)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace
}
