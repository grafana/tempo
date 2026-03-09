package rardecode

import "errors"

const (
	minWindowSize    = 0x40000
	maxQueuedFilters = 8192
)

var (
	ErrTooManyFilters   = errors.New("rardecode: too many filters")
	ErrInvalidFilter    = errors.New("rardecode: invalid filter")
	ErrMultipleDecoders = errors.New("rardecode: multiple decoders in a single archive not supported")
)

// filter functions take a byte slice, the current output offset and
// returns transformed data.
type filter func(b []byte, offset int64) ([]byte, error)

// filterBlock is a block of data to be processed by a filter.
type filterBlock struct {
	length int    // length of block
	offset int    // bytes to be read before start of block
	filter filter // filter function
}

// decoder is the interface for decoding compressed data
type decoder interface {
	init(r byteReader, reset bool, size int64, ver int) // initialize decoder for current file
	fill(dr *decodeReader) error                        // fill window with decoded data
	version() int                                       // decoder version
}

// decodeReader implements io.Reader for decoding compressed data in RAR archives.
type decodeReader struct {
	tot    int64          // total bytes read from window
	outbuf []byte         // buffered output
	buf    []byte         // filter buffer
	fl     []*filterBlock // list of filters each with offset relative to previous in list
	dec    decoder        // decoder being used to unpack file
	err    error          // current decoder error output
	br     byteReader

	win  []byte // sliding window buffer
	size int    // win length
	r    int    // index in win for reads (beginning)
	w    int    // index in win for writes (end)
}

func (d *decodeReader) init(r byteReader, ver int, size int, reset bool, unPackedSize int64) error {
	d.outbuf = nil
	d.tot = 0
	d.err = nil
	if reset {
		d.fl = nil
	}
	d.br = r

	// initialize window
	size = max(size, minWindowSize)
	if size > len(d.win) {
		b := make([]byte, size)
		if reset {
			d.w = 0
		} else if len(d.win) > 0 {
			n := copy(b, d.win[d.w:])
			n += copy(b[n:], d.win[:d.w])
			d.w = n
		}
		d.win = b
		d.size = size
	} else if reset {
		clear(d.win[:])
		d.w = 0
	}
	d.r = d.w

	// initialize decoder
	if d.dec == nil {
		switch ver {
		case decode29Ver:
			d.dec = new(decoder29)
		case decode50Ver, decode70Ver:
			d.dec = new(decoder50)
		case decode20Ver:
			d.dec = new(decoder20)
		default:
			return ErrUnknownDecoder
		}
	} else if d.dec.version() != ver {
		return ErrMultipleDecoders
	}
	d.dec.init(r, reset, unPackedSize, ver)
	return nil
}

// notFull returns if the window is not full
func (d *decodeReader) notFull() bool { return d.w < d.size }

// writeByte writes c to the end of the window
func (d *decodeReader) writeByte(c byte) {
	d.win[d.w] = c
	d.w++
}

// copyBytes copies len bytes at off distance from the end
// to the end of the window.
func (d *decodeReader) copyBytes(length, offset int) {
	length %= d.size
	if length < 0 {
		length += d.size
	}

	i := (d.w - offset) % d.size
	if i < 0 {
		i += d.size
	}
	iend := i + length
	if i > d.w {
		if iend > d.size {
			iend = d.size
		}
		n := copy(d.win[d.w:], d.win[i:iend])
		d.w += n
		length -= n
		if length == 0 {
			return
		}
		iend = length
		i = 0
	}
	if iend <= d.w {
		n := copy(d.win[d.w:], d.win[i:iend])
		d.w += n
		return
	}
	for length > 0 && d.w < d.size {
		d.win[d.w] = d.win[i]
		d.w++
		i++
		length--
	}
}

// queueFilter adds a filterBlock to the end decodeReader's filters.
func (d *decodeReader) queueFilter(f *filterBlock) error {
	if len(d.fl) >= maxQueuedFilters {
		return ErrTooManyFilters
	}
	// make offset relative to read index (from write index)
	f.offset += d.w - d.r
	// make offset relative to previous filter in list
	for _, fb := range d.fl {
		if f.offset < fb.offset {
			// filter block must not start before previous filter
			return ErrInvalidFilter
		}
		f.offset -= fb.offset
	}
	// offset & length must be < window size
	f.offset %= d.size
	if f.offset < 0 {
		f.offset += d.size
	}
	f.length %= d.size
	if f.length < 0 {
		f.length += d.size
	}
	d.fl = append(d.fl, f)
	return nil
}

func (d *decodeReader) readErr() error {
	err := d.err
	d.err = nil
	return err
}

// fill the decodeReader window
func (d *decodeReader) fill() error {
	if d.err != nil {
		return d.readErr()
	}
	if d.w == d.size {
		// wrap to beginning of buffer
		d.r = 0
		d.w = 0
	}
	d.err = d.dec.fill(d) // fill window using decoder
	if d.w == d.r {
		return d.readErr()
	}
	return nil
}

// bufBytes returns n bytes from the window in a new buffer.
func (d *decodeReader) bufBytes(n int) ([]byte, error) {
	if cap(d.buf) < n {
		d.buf = make([]byte, n)
	}
	// copy into buffer
	ns := 0
	for {
		nn := copy(d.buf[ns:n], d.win[d.r:d.w])
		d.r += nn
		ns += nn
		if ns == n {
			break
		}
		if err := d.fill(); err != nil {
			return nil, err
		}
	}
	return d.buf[:n], nil
}

// processFilters processes any filters valid at the current read index
// and returns the output in outbuf.
func (d *decodeReader) processFilters() ([]byte, error) {
	f := d.fl[0]
	flen := f.length

	// get filter input
	b, err := d.bufBytes(flen)
	if err != nil {
		return nil, err
	}
	for {
		d.fl = d.fl[1:]
		// run filter passing buffer and total bytes read so far
		b, err = f.filter(b, d.tot)
		if err != nil {
			return nil, err
		}
		if len(d.fl) == 0 {
			d.fl = nil
			return b, nil
		}
		// get next filter
		f = d.fl[0]
		if f.offset != 0 {
			// next filter not at current offset
			f.offset -= flen
			return b, nil
		}
		if f.length != len(b) {
			return nil, ErrInvalidFilter
		}
	}
}

// bytes returns a decoded byte slice or an error.
func (d *decodeReader) bytes() ([]byte, error) {
	// fill window if needed
	if d.w == d.r {
		if err := d.fill(); err != nil {
			return nil, err
		}
	}
	n := d.w - d.r

	// return current unread bytes if there are no filters
	if len(d.fl) == 0 {
		b := d.win[d.r:d.w]
		d.r = d.w
		d.tot += int64(n)
		return b, nil
	}

	// check filters
	f := d.fl[0]
	if f.offset < 0 {
		return nil, ErrInvalidFilter
	}
	if f.offset > 0 {
		// filter not at current read index, output bytes before it
		n = min(f.offset, n)
		b := d.win[d.r : d.r+n]
		d.r += n
		f.offset -= n
		d.tot += int64(n)
		return b, nil
	}

	// process filters at current index
	b, err := d.processFilters()
	if cap(b) > cap(d.buf) {
		// filter returned a larger buffer, cache it
		d.buf = b
	}

	d.tot += int64(len(b))
	return b, err
}

// Read decodes data and stores it in p.
func (d *decodeReader) Read(p []byte) (int, error) {
	var err error
	if len(d.outbuf) == 0 {
		d.outbuf, err = d.bytes()
		if err != nil {
			return 0, err
		}
	}
	n := copy(p, d.outbuf)
	d.outbuf = d.outbuf[n:]
	return n, err
}
