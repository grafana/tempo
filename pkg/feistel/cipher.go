package feistel

import (
	"encoding/hex"

	"github.com/segmentio/fasthash/fnv1a"
)

const (
	rounds = 4
)

func Encrypt(data []byte) []byte {
	if len(data)%2 != 0 {
		// TODO: Add padding (?)
		panic("data must be an even number of bytes")
	}
	half := len(data) / 2
	left, right := data[:half], data[half:]
	for i := 0; i < rounds; i++ {
		newRight := xor(left, roundKey(right))
		left, right = right, newRight
	}
	return append(left, right...)
}

func Decrypt(data []byte) []byte {
	if len(data)%2 != 0 {
		panic("data must be an even number of bytes")
	}
	half := len(data) / 2
	left, right := data[:half], data[half:]
	for i := 0; i < rounds; i++ {
		newLeft := xor(right, roundKey(left))
		left, right = newLeft, left
	}
	// TODO: Remove padding (?) if any
	return append(left, right...)
}

func roundKey(data []byte) []byte {
	h := fnv1a.HashBytes64(data)
	// Convert the hash to a byte slice
	hash := make([]byte, 8)
	for i := 0; i < 8; i++ {
		hash[i] = byte(h >> (8 * i))
	}
	hex.EncodeToString(hash)
	return hash
}

// xor performs a bitwise XOR operation on two byte slices.
// The two slices must be of equal length.
func xor(a, b []byte) []byte {
	if len(a) > len(b) {
		panic("xor: a must be shorter than or equal to b")
	}
	res := make([]byte, len(a))
	for i := range a {
		res[i] = a[i] ^ b[i]
	}
	return res
}
