// SPDX-FileCopyrightText: 2024 Shun Sakai
//
// SPDX-License-Identifier: Apache-2.0 OR MIT

// Package lzip implements reading and writing of lzip format compressed files.
// The package supports version 1 of the specification.
//
// See the following for the specification:
//
//   - https://www.nongnu.org/lzip/manual/lzip_manual.html#File-format
//   - https://datatracker.ietf.org/doc/html/draft-diaz-lzip-09#section-2
package lzip

import (
	"math/bits"

	"github.com/ulikunitz/xz/lzma"
)

const (
	headerSize  = 6
	trailerSize = 20
)

const magic = "LZIP"
const magicSize = 4

type version byte

const (
	version0 version = iota
	version1
)

// MinDictSize is the minimum dictionary size, which is 4 KiB.
const MinDictSize = lzma.MinDictCap

// MaxDictSize is the maximum dictionary size, which is 512 MiB.
const MaxDictSize = 1 << 29

// DefaultDictSize is the default dictionary size, which is 8 MiB.
const DefaultDictSize = 1 << 23

// MaxMemberSize is the maximum member size, which is 2 PiB.
const MaxMemberSize = 1 << 51

type header struct {
	magic [magicSize]byte
	version
	dictSize byte
}

func newHeader(dictSize uint32) *header {
	ds := bits.Len32(dictSize - 1)

	if dictSize > MinDictSize {
		base := 1 << dictSize
		frac := base / 16

		for i := 7; i >= 1; i-- {
			if (base - (i * frac)) >= ds {
				ds |= i << 5
			}
		}
	}

	z := &header{[magicSize]byte([]byte(magic)), version1, byte(ds)}

	return z
}

type trailer struct {
	crc        uint32
	dataSize   uint64
	memberSize uint64
}
