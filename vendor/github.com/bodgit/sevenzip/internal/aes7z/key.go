package aes7z

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	"go4.org/syncutil"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type cacheKey struct {
	password string
	cycles   int
	salt     string // []byte isn't comparable
}

const cacheSize = 10

//nolint:gochecknoglobals
var (
	once  syncutil.Once
	cache *lru.Cache[cacheKey, []byte]
)

func calculateKey(password string, cycles int, salt []byte) ([]byte, error) {
	if err := once.Do(func() (err error) {
		cache, err = lru.New[cacheKey, []byte](cacheSize)

		return
	}); err != nil {
		return nil, fmt.Errorf("aes7z: error creating cache: %w", err)
	}

	ck := cacheKey{
		password: password,
		cycles:   cycles,
		salt:     hex.EncodeToString(salt),
	}

	if key, ok := cache.Get(ck); ok {
		return key, nil
	}

	b := bytes.NewBuffer(salt)

	// Convert password to UTF-16LE
	utf16le := unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	t := transform.NewWriter(b, utf16le.NewEncoder())
	_, _ = t.Write([]byte(password))

	key := make([]byte, sha256.Size)
	if cycles == 0x3f {
		copy(key, b.Bytes())
	} else {
		h := sha256.New()
		for i := uint64(0); i < 1<<cycles; i++ {
			// These will never error
			_, _ = h.Write(b.Bytes())
			_ = binary.Write(h, binary.LittleEndian, i)
		}

		copy(key, h.Sum(nil))
	}

	_ = cache.Add(ck, key)

	return key, nil
}
