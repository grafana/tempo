package rardecode

type audioVar struct {
	k         [5]int
	d         [4]int
	lastDelta int
	dif       [11]int
	byteCount int
	lastChar  int
}

type audio20Decoder struct {
	chans     int // number of audio channels
	curChan   int // current audio channel
	chanDelta int

	decoders [4]huffmanDecoder
	vars     [4]audioVar

	br *rarBitReader
}

func (d *audio20Decoder) reset() {
	d.chans = 1
	d.curChan = 0
	d.chanDelta = 0

	for i := range d.vars {
		d.vars[i] = audioVar{}
	}
}

func (d *audio20Decoder) init(br *rarBitReader, table []byte) error {
	d.br = br
	n, err := br.readBits(2)
	if err != nil {
		return err
	}
	d.chans = n + 1
	if d.curChan >= d.chans {
		d.curChan = 0
	}
	table = table[:audioSize*d.chans]
	if err = readCodeLengthTable20(br, table); err != nil {
		return err
	}
	for i := 0; i < d.chans; i++ {
		d.decoders[i].init(table[:audioSize])
		table = table[audioSize:]
	}
	return nil
}

func (d *audio20Decoder) decode(delta int) byte {
	v := &d.vars[d.curChan]
	v.byteCount++
	v.d[3] = v.d[2]
	v.d[2] = v.d[1]
	v.d[1] = v.lastDelta - v.d[0]
	v.d[0] = v.lastDelta
	pch := 8*v.lastChar + v.k[0]*v.d[0] + v.k[1]*v.d[1] + v.k[2]*v.d[2] + v.k[3]*v.d[3] + v.k[4]*d.chanDelta
	pch = (pch >> 3) & 0xFF
	ch := pch - delta
	delta <<= 3

	v.dif[0] += abs(delta)
	v.dif[1] += abs(delta - v.d[0])
	v.dif[2] += abs(delta + v.d[0])
	v.dif[3] += abs(delta - v.d[1])
	v.dif[4] += abs(delta + v.d[1])
	v.dif[5] += abs(delta - v.d[2])
	v.dif[6] += abs(delta + v.d[2])
	v.dif[7] += abs(delta - v.d[3])
	v.dif[8] += abs(delta + v.d[3])
	v.dif[9] += abs(delta - d.chanDelta)
	v.dif[10] += abs(delta + d.chanDelta)

	d.chanDelta = ch - v.lastChar
	v.lastDelta = d.chanDelta
	v.lastChar = ch

	if v.byteCount&0x1F != 0 {
		return byte(ch)
	}

	var numMinDif int
	minDif := v.dif[0]
	v.dif[0] = 0
	for i := 1; i < len(v.dif); i++ {
		if v.dif[i] < minDif {
			minDif = v.dif[i]
			numMinDif = i
		}
		v.dif[i] = 0
	}
	if numMinDif > 0 {
		numMinDif--
		i := numMinDif / 2
		if numMinDif%2 == 0 {
			if v.k[i] >= -16 {
				v.k[i]--
			}
		} else if v.k[i] < 16 {
			v.k[i]++
		}
	}
	return byte(ch)
}

func (d *audio20Decoder) fill(dr *decodeReader, size int64) (int64, error) {
	var n int64
	for n < size && dr.notFull() {
		sym, err := d.decoders[d.curChan].readSym(d.br)
		if err != nil {
			return n, err
		}
		if sym == 256 {
			return n, errEndOfBlock
		}
		dr.writeByte(d.decode(sym))
		n++
		d.curChan++
		if d.curChan >= d.chans {
			d.curChan = 0
		}
	}
	return n, nil
}
