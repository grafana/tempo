package feistel

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)

	encrypted := Encrypt(traceID, false)
	decrypted := Decrypt(encrypted, false)
	require.Equal(t, traceID, decrypted)
}

func TestInPlaceEncryptDecrypt(t *testing.T) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)

	cpyTraceID := make([]byte, 16)
	copy(cpyTraceID, traceID)
	Encrypt(cpyTraceID, true)

	cpyEncrypted := make([]byte, 16)
	copy(cpyEncrypted, cpyTraceID)
	Decrypt(cpyEncrypted, true)
	require.Equal(t, traceID, cpyEncrypted)
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(b, err)

	b.Run("not in-place", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Decrypt(Encrypt(traceID, false), false)
		}
	})

	b.Run("in-place", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Decrypt(Encrypt(traceID, true), true)
		}
	})
}

func BenchmarkEncryptDecrypt_rounds(b *testing.B) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(b, err)

	rounds := []int{1, 2, 4, 8, 16, 32}
	b.Run("not in-place", func(b *testing.B) {
		for _, r := range rounds {
			b.Run(fmt.Sprintf("rounds %d", r), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = decrypt(encrypt(traceID, r), r)
				}
			})
		}
	})

	b.Run("in-place", func(b *testing.B) {
		for _, r := range rounds {
			b.Run(fmt.Sprintf("rounds %d", r), func(b *testing.B) {
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					inPlaceEncrypt(traceID, r)
					inPlaceDecrypt(traceID, r)
				}
			})
		}
	})
}
