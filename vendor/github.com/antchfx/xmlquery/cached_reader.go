package xmlquery

import (
	"bufio"
)

type cachedReader struct {
	buffer  *bufio.Reader
	cache   []byte
	caching bool
}

func newCachedReader(r *bufio.Reader) *cachedReader {
	return &cachedReader{
		buffer:  r,
		cache:   make([]byte, 0, 4096),
		caching: false,
	}
}

func (c *cachedReader) StartCaching() {
	c.cache = c.cache[:0]
	c.caching = true
}

func (c *cachedReader) ReadByte() (b byte, err error) {
	b, err = c.buffer.ReadByte()
	if err != nil {
		return
	}
	if c.caching {
		c.cacheByte(b)
	}
	return
}

func (c *cachedReader) Cache() []byte {
	return c.cache
}

func (c *cachedReader) CacheWithLimit(n int) []byte {
	if n < 1 {
		return nil
	}
	l := len(c.cache)
	if n > l {
		n = l
	}
	return c.cache[:n]
}

func (c *cachedReader) StopCaching() {
	c.caching = false
}

func (c *cachedReader) Read(p []byte) (int, error) {
	n, err := c.buffer.Read(p)
	if err != nil {
		return n, err
	}
	if c.caching {
		for i := 0; i < n; i++ {
			if !c.cacheByte(p[i]) {
				break
			}
		}
	}
	return n, err
}

func (c *cachedReader) cacheByte(b byte) bool {
	n := len(c.cache)
	if n == cap(c.cache) {
		return false
	}
	c.cache = c.cache[:n+1]
	c.cache[n] = b
	return true
}
