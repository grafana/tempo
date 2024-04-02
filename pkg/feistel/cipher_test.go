package feistel

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/grafana/tempo/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt(t *testing.T) {
	traceID := make([]byte, 16)
	_, err := rand.Read(traceID)
	require.NoError(t, err)
	fmt.Println("traceID:", util.TraceIDToHexString(traceID))

	cpyTraceID := make([]byte, 16)
	copy(cpyTraceID, traceID)
	encrypted := Encrypt(cpyTraceID)
	fmt.Println("encrypted:", util.TraceIDToHexString(encrypted))

	cpyEncrypted := make([]byte, 16)
	copy(cpyEncrypted, encrypted)
	decrypted := Decrypt(cpyEncrypted)
	fmt.Println("decrypted:", util.TraceIDToHexString(decrypted))
	require.Equal(t, traceID, decrypted)
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
