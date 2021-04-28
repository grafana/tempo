package model

import (
	"github.com/go-kit/kit/log/level"

	"github.com/cortexproject/cortex/pkg/util/log"
)

type objectCombiner struct{}

var ObjectCombiner = objectCombiner{}

func (o objectCombiner) Combine(objA []byte, objB []byte, dataEncoding string) []byte {
	combinedTrace, _, err := CombineTraces(objA, objB)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace
}

func (o objectCombiner) CombineAcrossEncodings(objA []byte, objB []byte, dataEncodingA string, dataEncodingB string) []byte { // jpe actually support this
	combinedTrace, _, err := CombineTraces(objA, objB)
	if err != nil {
		level.Error(log.Logger).Log("msg", "error combining trace protos", "err", err.Error())
	}
	return combinedTrace
}
