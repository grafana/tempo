package rardecode

const (
	main20Size   = 298
	offset20Size = 48
	length20Size = 28
)

type lz20Decoder struct {
	length int    // previous length
	offset [4]int // history of previous offsets

	mainDecoder   huffmanDecoder
	offsetDecoder huffmanDecoder
	lengthDecoder huffmanDecoder

	br *rarBitReader
}

func (d *lz20Decoder) init(br *rarBitReader, table []byte) error {
	d.br = br

	table = table[:main20Size+offset20Size+length20Size]
	if err := readCodeLengthTable20(br, table); err != nil {
		return err
	}
	d.mainDecoder.init(table[:main20Size])
	table = table[main20Size:]
	d.offsetDecoder.init(table[:offset20Size])
	table = table[offset20Size:]
	d.lengthDecoder.init(table)
	return nil
}

func (d *lz20Decoder) decodeOffset(i int) error {
	d.length = lengthBase[i] + 3
	bits := lengthExtraBits[i]
	if bits > 0 {
		n, err := d.br.readBits(bits)
		if err != nil {
			return err
		}
		d.length += n
	}

	var err error
	i, err = d.offsetDecoder.readSym(d.br)
	if err != nil {
		return err
	}
	offset := offsetBase[i] + 1
	bits = offsetExtraBits[i]
	if bits > 0 {
		n, err := d.br.readBits(bits)
		if err != nil {
			return err
		}
		offset += n
	}

	if offset >= 0x2000 {
		d.length++
		if offset >= 0x40000 {
			d.length++
		}
	}
	copy(d.offset[1:], d.offset[:])
	d.offset[0] = offset
	return nil
}

func (d *lz20Decoder) decodeLength(i int) error {
	offset := d.offset[i]
	copy(d.offset[1:], d.offset[:])
	d.offset[0] = offset

	i, err := d.lengthDecoder.readSym(d.br)
	if err != nil {
		return err
	}
	d.length = lengthBase[i] + 2
	bits := lengthExtraBits[i]
	if bits > 0 {
		var n int
		n, err = d.br.readBits(bits)
		if err != nil {
			return err
		}
		d.length += n
	}
	if offset >= 0x101 {
		d.length++
		if offset >= 0x2000 {
			d.length++
			if offset >= 0x40000 {
				d.length++
			}
		}
	}
	return nil
}

func (d *lz20Decoder) decodeShortOffset(i int) error {
	copy(d.offset[1:], d.offset[:])
	offset := shortOffsetBase[i] + 1
	bits := shortOffsetExtraBits[i]
	if bits > 0 {
		n, err := d.br.readBits(bits)
		if err != nil {
			return err
		}
		offset += n
	}
	d.offset[0] = offset
	d.length = 2
	return nil
}

func (d *lz20Decoder) fill(dr *decodeReader, size int64) (int64, error) {
	var n int64
	for n < size && dr.notFull() {
		sym, err := d.mainDecoder.readSym(d.br)
		if err != nil {
			return n, err
		}

		switch {
		case sym < 256: // literal
			dr.writeByte(byte(sym))
			n++
			continue
		case sym > 269:
			err = d.decodeOffset(sym - 270)
		case sym == 269:
			return n, errEndOfBlock
		case sym == 256: // use previous offset and length
			copy(d.offset[1:], d.offset[:])
		case sym < 261:
			err = d.decodeLength(sym - 257)
		default:
			err = d.decodeShortOffset(sym - 261)
		}
		if err != nil {
			return n, err
		}
		dr.copyBytes(d.length, d.offset[0])
		n += int64(d.length)
	}
	return n, nil
}
