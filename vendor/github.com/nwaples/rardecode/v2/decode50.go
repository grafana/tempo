package rardecode

import (
	"errors"
	"io"
	"math/bits"
)

const (
	mainSize5      = 306
	offsetSize5    = 64
	lowoffsetSize5 = 16
	lengthSize5    = 44
	tableSize5     = mainSize5 + offsetSize5 + lowoffsetSize5 + lengthSize5

	offsetSize7 = 80
	tableSize7  = mainSize5 + offsetSize7 + lowoffsetSize5 + lengthSize5
)

var (
	ErrUnknownFilter       = errors.New("rardecode: unknown V5 filter")
	ErrCorruptDecodeHeader = errors.New("rardecode: corrupt decode header")
)

// decoder50 implements the decoder interface for RAR 5 compression.
// Decode input it broken up into 1 or more blocks. Each block starts with
// a header containing block length and optional code length tables to initialize
// the huffman decoders with.
type decoder50 struct {
	br         rar5BitReader // bit reader for current data block
	buf        [tableSize7]byte
	codeLength []byte
	offsetSize int

	lastBlock bool // current block is last block in compressed file

	mainDecoder      huffmanDecoder
	offsetDecoder    huffmanDecoder
	lowoffsetDecoder huffmanDecoder
	lengthDecoder    huffmanDecoder

	offset [4]int
	length int
}

func (d *decoder50) version() int { return decode50Ver }

func (d *decoder50) init(r byteReader, reset bool, size int64, ver int) {
	d.br.reset(r)
	d.lastBlock = false
	if ver == decode70Ver {
		d.codeLength = d.buf[:]
		d.offsetSize = offsetSize7
	} else {
		d.codeLength = d.buf[:tableSize5]
		d.offsetSize = offsetSize5
	}

	if reset {
		clear(d.offset[:])
		d.length = 0
		clear(d.codeLength[:])
	}
}

func (d *decoder50) readBlockHeader() error {
	flags, err := d.br.ReadByte()
	if err != nil {
		return err
	}

	bytecount := (flags>>3)&3 + 1
	if bytecount == 4 {
		return ErrCorruptDecodeHeader
	}

	hsum, err := d.br.ReadByte()
	if err != nil {
		return err
	}

	blockBits := int(flags)&0x07 + 1
	blockBytes := 0
	sum := 0x5a ^ flags
	for i := byte(0); i < bytecount; i++ {
		var n byte
		n, err = d.br.ReadByte()
		if err != nil {
			return err
		}
		sum ^= n
		blockBytes |= int(n) << (i * 8)
	}
	if sum != hsum { // bad header checksum
		return ErrCorruptDecodeHeader
	}
	blockBits += (blockBytes - 1) * 8

	// reset the bits limit
	d.br.setLimit(blockBits)
	d.lastBlock = flags&0x40 > 0

	if flags&0x80 > 0 {
		// read new code length tables and reinitialize huffman decoders
		cl := d.codeLength[:]
		err = readCodeLengthTable(&d.br, cl, false)
		if err != nil {
			return err
		}
		d.mainDecoder.init(cl[:mainSize5])
		cl = cl[mainSize5:]
		d.offsetDecoder.init(cl[:d.offsetSize])
		cl = cl[d.offsetSize:]
		d.lowoffsetDecoder.init(cl[:lowoffsetSize5])
		cl = cl[lowoffsetSize5:]
		d.lengthDecoder.init(cl)
	}
	return nil
}

func slotToLength(br bitReader, n int) (int, error) {
	if n >= 8 {
		bits := uint8(n/4 - 1)
		n = (4 | (n & 3)) << bits
		if bits > 0 {
			b, err := br.readBits(bits)
			if err != nil {
				return 0, err
			}
			n |= b
		}
	}
	n += 2
	return n, nil
}

// readFilter5Data reads an encoded integer used in V5 filters.
func readFilter5Data(br bitReader) (int, error) {
	// TODO: should data really be uint? (for 32bit ints).
	// It will be masked later anyway by decode window mask.
	bytes, err := br.readBits(2)
	if err != nil {
		return 0, err
	}
	bytes++

	var data int
	for i := 0; i < bytes; i++ {
		n, err := br.readBits(8)
		if err != nil {
			return 0, err
		}
		data |= n << (uint(i) * 8)
	}
	return data, nil
}

func (d *decoder50) readFilter(dr *decodeReader) error {
	fb := new(filterBlock)
	var err error

	fb.offset, err = readFilter5Data(&d.br)
	if err != nil {
		return err
	}
	fb.length, err = readFilter5Data(&d.br)
	if err != nil {
		return err
	}
	ftype, err := d.br.readBits(3)
	if err != nil {
		return err
	}
	switch ftype {
	case 0:
		n, err := d.br.readBits(5)
		if err != nil {
			return err
		}
		fb.filter = func(buf []byte, offset int64) ([]byte, error) { return filterDelta(n+1, buf) }
	case 1:
		fb.filter = func(buf []byte, offset int64) ([]byte, error) { return filterE8(0xe8, true, buf, offset) }
	case 2:
		fb.filter = func(buf []byte, offset int64) ([]byte, error) { return filterE8(0xe9, true, buf, offset) }
	case 3:
		fb.filter = filterArm
	default:
		return ErrUnknownFilter
	}
	return dr.queueFilter(fb)
}

func (d *decoder50) decodeLength(dr *decodeReader, i int) error {
	offset := d.offset[i]
	copy(d.offset[1:i+1], d.offset[:i])
	d.offset[0] = offset

	sl, err := d.lengthDecoder.readSym(&d.br)
	if err != nil {
		return err
	}
	d.length, err = slotToLength(&d.br, sl)
	if err == nil {
		dr.copyBytes(d.length, d.offset[0])
	}
	return err
}

func (d *decoder50) decodeOffset(dr *decodeReader, i int) error {
	length, err := slotToLength(&d.br, i)
	if err != nil {
		return err
	}

	offset := 1
	slot, err := d.offsetDecoder.readSym(&d.br)
	if err != nil {
		return err
	}
	if slot < 4 {
		offset += slot
	} else {
		bitCount := uint8(slot/2 - 1)
		offset += (2 | (slot & 1)) << bitCount

		if bitCount >= 4 {
			bitCount -= 4
			if bitCount > 0 {
				if bits.UintSize == 32 {
					// bitReader can only read at most intSize-8 bits.
					// Split read into two parts.
					if bitCount > 24 {
						n, err := d.br.readBits(24)
						if err != nil {
							return err
						}
						bitCount -= 24
						offset += n << (4 + bitCount)
					}
				}
				n, err := d.br.readBits(bitCount)
				if err != nil {
					return err
				}
				offset += n << 4
			}
			n, err := d.lowoffsetDecoder.readSym(&d.br)
			if err != nil {
				return err
			}
			offset += n
		} else {
			n, err := d.br.readBits(bitCount)
			if err != nil {
				return err
			}
			offset += n
		}
	}
	if offset > 0x100 {
		length++
		if offset > 0x2000 {
			length++
			if offset > 0x40000 {
				length++
			}
		}
	}
	copy(d.offset[1:], d.offset[:])
	d.offset[0] = offset
	d.length = length
	dr.copyBytes(d.length, d.offset[0])
	return nil
}

func (d *decoder50) fill(dr *decodeReader) error {
	for dr.notFull() {
		sym, err := d.mainDecoder.readSym(&d.br)
		if err == nil {
			switch {
			case sym < 256:
				// literal
				dr.writeByte(byte(sym))
				continue
			case sym >= 262:
				err = d.decodeOffset(dr, sym-262)
			case sym >= 258:
				err = d.decodeLength(dr, sym-258)
			case sym == 257:
				// use previous offset and length
				dr.copyBytes(d.length, d.offset[0])
				continue
			default: // sym == 256:
				err = d.readFilter(dr)
			}
		} else if err == io.EOF {
			// reached end of the block
			if d.lastBlock {
				return io.EOF
			}
			err = d.readBlockHeader()
		}
		if err != nil {
			if err == io.EOF {
				return ErrDecoderOutOfData
			}
			return err
		}
	}
	return nil
}
