package rardecode

import (
	"io"
)

const audioSize = 257

type decoder20 struct {
	br      *rarBitReader
	size    int64 // unpacked bytes left to be decompressed
	hdrRead bool  // block header has been read
	isAudio bool  // current block is Audio

	codeLength [audioSize * 4]byte

	lz    *lz20Decoder
	audio *audio20Decoder
}

func (d *decoder20) version() int { return decode20Ver }

// init intializes the decoder for decoding a new file.
func (d *decoder20) init(r byteReader, reset bool, size int64, ver int) {
	if d.br == nil {
		d.br = newRarBitReader(r)
	} else {
		d.br.reset(r)
	}
	d.size = size
	if reset {
		d.hdrRead = false
		d.isAudio = false
		if d.audio != nil {
			d.audio.reset()
		}
		clear(d.codeLength[:])
	}
}

func readCodeLengthTable20(br *rarBitReader, table []byte) error {
	var bitlength [19]byte
	for i := 0; i < len(bitlength); i++ {
		n, err := br.readBits(4)
		if err != nil {
			return err
		}
		bitlength[i] = byte(n)
	}

	var bl huffmanDecoder
	bl.init(bitlength[:])

	for i := 0; i < len(table); {
		l, err := bl.readSym(br)
		if err != nil {
			return err
		}
		if l < 16 {
			table[i] = (table[i] + byte(l)) & 0xf
			i++
			continue
		}
		if l == 16 {
			if i == 0 {
				return ErrInvalidLengthTable
			}
			var n int
			n, err = br.readBits(2)
			if err != nil {
				return err
			}
			n += 3
			n = min(i+n, len(table))
			v := table[i-1]
			for i < n {
				table[i] = v
				i++
			}
			continue
		}
		var n int
		if l == 17 {
			n, err = br.readBits(3)
			if err != nil {
				return err
			}
			n += 3
		} else {
			n, err = br.readBits(7)
			if err != nil {
				return err
			}
			n += 11
		}
		n = min(i+n, len(table))
		clear(table[i:n])
		i = n
	}
	return nil
}

func (d *decoder20) readBlockHeader() error {
	n, err := d.br.readBits(1)
	if err != nil {
		return err
	}
	d.isAudio = n > 0
	n, err = d.br.readBits(1)
	if err != nil {
		return err
	}
	if n == 0 {
		clear(d.codeLength[:])
	}
	if d.isAudio {
		if d.audio == nil {
			d.audio = new(audio20Decoder)
		}
		err = d.audio.init(d.br, d.codeLength[:])
	} else {
		if d.lz == nil {
			d.lz = new(lz20Decoder)
		}
		err = d.lz.init(d.br, d.codeLength[:])
	}
	d.hdrRead = true
	return err
}

func (d *decoder20) fill(dr *decodeReader) error {
	for d.size > 0 && dr.notFull() {
		if !d.hdrRead {
			if err := d.readBlockHeader(); err != nil {
				return err
			}
		}
		var n int64
		var err error
		if d.isAudio {
			n, err = d.audio.fill(dr, d.size)
		} else {
			n, err = d.lz.fill(dr, d.size)
		}
		d.size -= n
		switch err {
		case nil:
			continue
		case errEndOfBlock:
			d.hdrRead = false
			continue
		case io.EOF:
			err = ErrDecoderOutOfData
		}
		return err
	}
	if d.size == 0 {
		return io.EOF
	}
	return nil
}
