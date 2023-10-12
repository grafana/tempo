// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pdatautil // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"

import (
	"encoding/binary"
	"hash"
	"math"
	"sort"
	"sync"

	"github.com/cespare/xxhash/v2"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

var (
	extraByte       = []byte{'\xf3'}
	keyPrefix       = []byte{'\xf4'}
	valEmpty        = []byte{'\xf5'}
	valBytesPrefix  = []byte{'\xf6'}
	valStrPrefix    = []byte{'\xf7'}
	valBoolTrue     = []byte{'\xf8'}
	valBoolFalse    = []byte{'\xf9'}
	valIntPrefix    = []byte{'\xfa'}
	valDoublePrefix = []byte{'\xfb'}
	valMapPrefix    = []byte{'\xfc'}
	valMapSuffix    = []byte{'\xfd'}
	valSlicePrefix  = []byte{'\xfe'}
	valSliceSuffix  = []byte{'\xff'}
)

type hashWriter struct {
	h       hash.Hash
	strBuf  []byte
	keysBuf []string
	sumHash []byte
	numBuf  []byte
}

func newHashWriter() *hashWriter {
	return &hashWriter{
		h:       xxhash.New(),
		strBuf:  make([]byte, 0, 128),
		keysBuf: make([]string, 0, 16),
		sumHash: make([]byte, 0, 16),
		numBuf:  make([]byte, 8),
	}
}

var hashWriterPool = &sync.Pool{
	New: func() interface{} { return newHashWriter() },
}

// MapHash return a hash for the provided map.
// Maps with the same underlying key/value pairs in different order produce the same deterministic hash value.
func MapHash(m pcommon.Map) [16]byte {
	hw := hashWriterPool.Get().(*hashWriter)
	defer hashWriterPool.Put(hw)
	hw.h.Reset()
	hw.writeMapHash(m)
	return hw.hashSum128()
}

// ValueHash return a hash for the provided pcommon.Value.
func ValueHash(v pcommon.Value) [16]byte {
	hw := hashWriterPool.Get().(*hashWriter)
	defer hashWriterPool.Put(hw)
	hw.h.Reset()
	hw.writeValueHash(v)
	return hw.hashSum128()
}

func (hw *hashWriter) writeMapHash(m pcommon.Map) {
	// For each recursive call into this function we want to preserve the previous buffer state
	// while also adding new keys to the buffer. nextIndex is the index of the first new key
	// added to the buffer for this call of the function.
	// This also works for the first non-recursive call of this function because the buffer is always empty
	// on the first call due to it being cleared of any added keys at then end of the function.
	nextIndex := len(hw.keysBuf)

	m.Range(func(k string, v pcommon.Value) bool {
		hw.keysBuf = append(hw.keysBuf, k)
		return true
	})

	// Get only the newly added keys from the buffer by slicing the buffer from nextIndex to the end
	workingKeySet := hw.keysBuf[nextIndex:]

	sort.Strings(workingKeySet)
	for _, k := range workingKeySet {
		v, _ := m.Get(k)
		hw.strBuf = hw.strBuf[:0]
		hw.strBuf = append(hw.strBuf, keyPrefix...)
		hw.strBuf = append(hw.strBuf, k...)
		hw.h.Write(hw.strBuf)
		hw.writeValueHash(v)
	}

	// Remove all keys that were added to the buffer during this call of the function
	hw.keysBuf = hw.keysBuf[:nextIndex]
}

func (hw *hashWriter) writeSliceHash(sl pcommon.Slice) {
	for i := 0; i < sl.Len(); i++ {
		hw.writeValueHash(sl.At(i))
	}
}

func (hw *hashWriter) writeValueHash(v pcommon.Value) {
	switch v.Type() {
	case pcommon.ValueTypeStr:
		hw.strBuf = hw.strBuf[:0]
		hw.strBuf = append(hw.strBuf, valStrPrefix...)
		hw.strBuf = append(hw.strBuf, v.Str()...)
		hw.h.Write(hw.strBuf)
	case pcommon.ValueTypeBool:
		if v.Bool() {
			hw.h.Write(valBoolTrue)
		} else {
			hw.h.Write(valBoolFalse)
		}
	case pcommon.ValueTypeInt:
		hw.h.Write(valIntPrefix)
		binary.LittleEndian.PutUint64(hw.numBuf, uint64(v.Int()))
		hw.h.Write(hw.numBuf)
	case pcommon.ValueTypeDouble:
		hw.h.Write(valDoublePrefix)
		binary.LittleEndian.PutUint64(hw.numBuf, math.Float64bits(v.Double()))
		hw.h.Write(hw.numBuf)
	case pcommon.ValueTypeMap:
		hw.h.Write(valMapPrefix)
		hw.writeMapHash(v.Map())
		hw.h.Write(valMapSuffix)
	case pcommon.ValueTypeSlice:
		hw.h.Write(valSlicePrefix)
		hw.writeSliceHash(v.Slice())
		hw.h.Write(valSliceSuffix)
	case pcommon.ValueTypeBytes:
		hw.h.Write(valBytesPrefix)
		hw.h.Write(v.Bytes().AsRaw())
	case pcommon.ValueTypeEmpty:
		hw.h.Write(valEmpty)
	}
}

// hashSum128 returns a [16]byte hash sum.
func (hw *hashWriter) hashSum128() [16]byte {
	b := hw.sumHash[:0]
	b = hw.h.Sum(b)

	// Append an extra byte to generate another part of the hash sum
	_, _ = hw.h.Write(extraByte)
	b = hw.h.Sum(b)

	res := [16]byte{}
	copy(res[:], b)
	return res
}
