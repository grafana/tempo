package model

import (
	"github.com/go-kit/kit/log/level"

	"github.com/cortexproject/cortex/pkg/util/log"
)

type objectCombiner struct{}

var ObjectCombiner = objectCombiner{}

// Combine implements tempodb/encoding/common.ObjectCombiner
func (o objectCombiner) Combine(objA []byte, objB []byte, dataEncoding string) ([]byte, bool) {
	combinedTrace, wasCombined, err := CombineTraceBytes(objA, objB, dataEncoding, dataEncoding)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace, wasCombined
}
