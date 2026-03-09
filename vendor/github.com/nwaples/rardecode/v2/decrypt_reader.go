package rardecode

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// cipherBlockSliceReader is a sliceReader that users a cipher.BlockMode to decrypt the input.
type cipherBlockSliceReader struct {
	r    sliceReader
	mode cipher.BlockMode
	n    int // bytes encrypted but not read
}

func (c *cipherBlockSliceReader) sizeInBlocks(n int) int {
	bs := c.mode.BlockSize()
	if rem := n % bs; rem > 0 {
		n += bs - rem
	}
	return n
}

func (c *cipherBlockSliceReader) peek(n int) ([]byte, error) {
	bn := c.sizeInBlocks(n)
	b, err := c.r.peek(bn)
	if err != nil {
		if err == io.EOF && len(b) > 0 {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}
	if c.n < bn {
		c.mode.CryptBlocks(b[c.n:], b[c.n:])
		c.n = bn
	}
	return b[:n], nil
}

// readSlice returns the next n bytes of decrypted input.
// If n is not a multiple of the block size, the trailing bytes
// of the last decrypted block will be discarded.
func (c *cipherBlockSliceReader) readSlice(n int) ([]byte, error) {
	bn := c.sizeInBlocks(n)
	b, err := c.r.readSlice(bn)
	if err != nil {
		return nil, err
	}
	if c.n < bn {
		c.mode.CryptBlocks(b[c.n:], b[c.n:])
		c.n = 0
	} else {
		c.n -= bn
	}
	// ignore padding at end of last block
	b = b[:n]
	return b, nil
}

// newAesSliceReader creates a sliceReader that uses AES to decrypt the input
func newAesSliceReader(r sliceReader, key, iv []byte) *cipherBlockSliceReader {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	return &cipherBlockSliceReader{r: r, mode: mode}
}

// cipherBlockReader implements Block Mode decryption of an io.Reader object.
type cipherBlockReader struct {
	r       byteReader
	mode    cipher.BlockMode
	getMode func() (cipher.BlockMode, error)
	inbuf   []byte // raw input blocks not yet decrypted
	outbuf  []byte // output buffer used when output slice < block size
	block   []byte // output buffer for a single block
}

// readBlock returns a single decrypted block.
func (cr *cipherBlockReader) readBlock() ([]byte, error) {
	bs := len(cr.block)
	if len(cr.inbuf) >= bs {
		cr.mode.CryptBlocks(cr.block, cr.inbuf[:bs])
		cr.inbuf = cr.inbuf[bs:]
	} else {
		n := copy(cr.block, cr.inbuf)
		cr.inbuf = nil
		_, err := io.ReadFull(cr.r, cr.block[n:])
		if err != nil {
			return nil, err
		}
		cr.mode.CryptBlocks(cr.block, cr.block)
	}
	return cr.block, nil
}

// Read reads and decrypts data into p.
// If the input is not a multiple of the cipher block size,
// the trailing bytes will be ignored.
func (cr *cipherBlockReader) Read(p []byte) (int, error) {
	if len(cr.outbuf) > 0 {
		n := copy(p, cr.outbuf)
		cr.outbuf = cr.outbuf[n:]
		return n, nil
	}
	// get input blocks
	for len(cr.inbuf) == 0 {
		var err error
		cr.inbuf, err = cr.r.bytes()
		if err != nil {
			return 0, err
		}
	}
	if cr.mode == nil {
		var err error
		cr.mode, err = cr.getMode()
		if err != nil {
			return 0, err
		}
		cr.block = make([]byte, cr.mode.BlockSize())
	}
	bs := cr.mode.BlockSize()
	n := len(cr.inbuf)
	l := len(p)
	if n < bs || l < bs {
		// Next encrypted block spans volumes or Read buffer is too small
		// to fit a single block. Decrypt a single block and store the
		// leftover in outbuf.
		b, err := cr.readBlock()
		if err != nil {
			return 0, err
		}
		n = copy(p, b)
		cr.outbuf = b[n:]
		return n, nil
	}
	// output buffer smaller than input
	n = min(l, n)
	// round down to block size
	n -= n % bs
	cr.mode.CryptBlocks(p[:n], cr.inbuf[:n])
	cr.inbuf = cr.inbuf[n:]
	return n, nil
}

// bytes returns a byte slice of decrypted data.
func (cr *cipherBlockReader) bytes() ([]byte, error) {
	if len(cr.outbuf) > 0 {
		b := cr.outbuf
		cr.outbuf = nil
		return b, nil
	}
	// get more input
	for len(cr.inbuf) == 0 {
		var err error
		cr.inbuf, err = cr.r.bytes()
		if err != nil {
			return nil, err
		}
	}
	if cr.mode == nil {
		var err error
		cr.mode, err = cr.getMode()
		if err != nil {
			return nil, err
		}
		cr.block = make([]byte, cr.mode.BlockSize())
	}
	bs := cr.mode.BlockSize()
	if len(cr.inbuf) < bs {
		// next encrypted block spans volumes
		return cr.readBlock()
	}
	n := len(cr.inbuf)
	n -= n % bs
	// get input buffer and round down to nearest block boundary
	b := cr.inbuf[:n]
	cr.inbuf = cr.inbuf[n:]
	cr.mode.CryptBlocks(b, b)
	return b, nil
}

func newCipherBlockReader(r byteReader, getMode func() (cipher.BlockMode, error)) *cipherBlockReader {
	c := &cipherBlockReader{r: r, getMode: getMode}
	return c
}

// newAesDecryptReader returns a cipherBlockReader that decrypts input from a given io.Reader using AES.
func newAesDecryptReader(r byteReader, h *fileBlockHeader) *cipherBlockReader {
	getMode := func() (cipher.BlockMode, error) {
		err := h.genKeys()
		if err != nil {
			return nil, err
		}
		block, err := aes.NewCipher(h.key)
		if err != nil {
			return nil, err
		}
		return cipher.NewCBCDecrypter(block, h.iv), nil
	}
	return newCipherBlockReader(r, getMode)
}
