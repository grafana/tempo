package v2

import (
	crand "crypto/rand"
	"testing"

	"github.com/grafana/tempo/v2/tempodb/encoding/common"
	"github.com/stretchr/testify/require"
)

type noopDataWriter struct{}

func (n noopDataWriter) Write(common.ID, []byte) (int, error) { return 10, nil }
func (n noopDataWriter) CutPage() (int, error)                { return 100, nil }
func (n noopDataWriter) Complete() error                      { return nil }

func BenchmarkAppender100(b *testing.B) {
	benchmarkAppender(b, 100)
}

func BenchmarkAppender1000(b *testing.B) {
	benchmarkAppender(b, 1000)
}

func BenchmarkAppender10000(b *testing.B) {
	benchmarkAppender(b, 10000)
}

func BenchmarkAppender200000(b *testing.B) {
	benchmarkAppender(b, 200000)
}

func BenchmarkAppender500000(b *testing.B) {
	benchmarkAppender(b, 500000)
}

func benchmarkAppender(b *testing.B, appendRecords int) {
	for i := 0; i < b.N; i++ {
		appender := NewAppender(noopDataWriter{})

		for j := 0; j < appendRecords; j++ {
			id := make([]byte, 16)
			_, err := crand.Read(id)
			require.NoError(b, err)

			err = appender.Append(id, nil)
			require.NoError(b, err)
		}

		_ = appender.Records()
	}
}
