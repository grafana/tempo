package feistel

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)

	encrypted := Encrypt(traceID)
	decrypted := Decrypt(encrypted)
	require.Equal(t, traceID, decrypted)
}

func TestInPlaceEncryptDecrypt(t *testing.T) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)

	cpyTraceID := make([]byte, 16)
	copy(cpyTraceID, traceID)
	InPlaceEncrypt(cpyTraceID)

	cpyEncrypted := make([]byte, 16)
	copy(cpyEncrypted, cpyTraceID)
	InPlaceDecrypt(cpyEncrypted)
	require.Equal(t, traceID, cpyEncrypted)
}

func BenchmarkEncryptDecrypt(b *testing.B) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Decrypt(Encrypt(traceID))
	}
}

func BenchmarkInPlaceEncryptDecrypt(b *testing.B) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InPlaceEncrypt(traceID)
		InPlaceDecrypt(traceID)
	}
}
