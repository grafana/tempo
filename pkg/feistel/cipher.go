package feistel

import (
	"github.com/segmentio/fasthash/fnv1a"
)

const (
	rounds = 4
)

func Encrypt(data []byte) []byte {
	if len(data)%2 != 0 {
		// TODO: Add padding
		panic("data must be an even number of bytes")
	}
	half := len(data) / 2
	left, right := data[:half], data[half:]
	key := make([]byte, half)
	for i := 0; i < rounds; i++ {
		roundKey(right, key)
		newRight := xor(left, key)
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
	key := make([]byte, half)
	for i := 0; i < rounds; i++ {
		roundKey(left, key)
		newLeft := xor(right, key)
		left, right = newLeft, left
	}
	// TODO: Remove padding (?) if any
	return append(left, right...)
}

func InPlaceEncrypt(data []byte) {
	if len(data)%2 != 0 {
		panic("data must be an even number of bytes")
	}
	half := len(data) / 2
	key := make([]byte, half)
	for i := 0; i < rounds; i++ {
		roundKey(data[half:], key)
		for j := 0; j < half; j++ {
			data[j], data[half+j] = data[half+j], data[j]^key[j]
		}
	}
}

func InPlaceDecrypt(data []byte) {
	if len(data)%2 != 0 {
		panic("data must be an even number of bytes")
	}
	half := len(data) / 2
	key := make([]byte, half)
	for i := 0; i < rounds; i++ {
		roundKey(data[:half], key)
		for j := 0; j < half; j++ {
			data[j], data[half+j] = data[half+j]^key[j], data[j]
		}
	}
}

func roundKey(data []byte, hash []byte) {
	h := fnv1a.HashBytes64(data)
	// Convert the hash to a byte slice
	for i := 0; i < 8; i++ {
		hash[i] = byte(h >> (8 * i))
	}
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
