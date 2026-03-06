package rardecode

type ppm29Decoder struct {
	m   model // ppm model
	esc byte  // escape character
	br  *rarBitReader
}

func (d *ppm29Decoder) init(br *rarBitReader) error {
	maxOrder, err := br.readBits(7)
	if err != nil {
		return err
	}
	reset := maxOrder&0x20 > 0

	// Move any bytes in rarBitReader bit cache back into a byte slice.
	// PPM only reads bytes so it is more efficient to read those bytes
	// directly from byte slices, bypassing the extra bit shifts.
	br.unshiftBytes()
	d.br = br

	var maxMB int
	if reset {
		var c byte
		c, err = d.br.ReadByte()
		if err != nil {
			return err
		}
		maxMB = int(c) + 1
	}

	if maxOrder&0x40 > 0 {
		d.esc, err = d.br.ReadByte()
		if err != nil {
			return err
		}
	}

	maxOrder = (maxOrder & 0x1f) + 1
	if maxOrder > 16 {
		maxOrder = 16 + (maxOrder-16)*3
	}

	return d.m.init(d.br, reset, maxOrder, maxMB)
}

func (d *ppm29Decoder) reset() {
	d.esc = 2
}

func (d *ppm29Decoder) readFilterData() ([]byte, error) {
	c, err := d.m.ReadByte()
	if err != nil {
		return nil, err
	}
	n := int(c&7) + 1
	if n == 7 {
		var b byte
		b, err = d.m.ReadByte()
		if err != nil {
			return nil, err
		}
		n += int(b)
	} else if n == 8 {
		var b byte
		b, err = d.m.ReadByte()
		if err != nil {
			return nil, err
		}
		n = int(b) << 8
		b, err = d.m.ReadByte()
		if err != nil {
			return nil, err
		}
		n |= int(b)
	}

	n++
	buf := make([]byte, n)
	buf[0] = c
	for i := 1; i < n; i++ {
		buf[i], err = d.m.ReadByte()
		if err != nil {
			return nil, err
		}
	}
	return buf, nil
}

// fill window until full, error, filter found or end of block.
func (d *ppm29Decoder) fill(dr *decodeReader) ([]byte, error) {
	for dr.notFull() {
		c, err := d.m.ReadByte()
		if err != nil {
			return nil, err
		}
		if c != d.esc {
			dr.writeByte(c)
			continue
		}
		c, err = d.m.ReadByte()
		if err != nil {
			return nil, err
		}

		switch c {
		case 0:
			return nil, errEndOfBlock
		case 2:
			return nil, errEndOfBlockAndFile
		case 3:
			return d.readFilterData()
		case 4:
			offset := 0
			for i := 0; i < 3; i++ {
				c, err = d.m.ReadByte()
				if err != nil {
					return nil, err
				}
				offset = offset<<8 | int(c)
			}
			len, err := d.m.ReadByte()
			if err != nil {
				return nil, err
			}
			dr.copyBytes(int(len)+32, offset+2)
		case 5:
			len, err := d.m.ReadByte()
			if err != nil {
				return nil, err
			}
			dr.copyBytes(int(len)+4, 1)
		default:
			dr.writeByte(d.esc)
		}
	}
	return nil, nil
}

func newPPM29Decoder() *ppm29Decoder {
	ppm := new(ppm29Decoder)
	ppm.reset()
	ppm.m.maxOrder = 2
	ppm.m.a.init(1)

	return ppm
}
