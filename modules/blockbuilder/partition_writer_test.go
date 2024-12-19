package blockbuilder

import (
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/tempo/pkg/tempopb"
	"github.com/grafana/tempo/pkg/util/test"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/encoding"
	"github.com/grafana/tempo/tempodb/wal"
	"github.com/stretchr/testify/require"
)

func getPartitionWriter(t *testing.T) *writer {
	logger := log.NewNopLogger()
	endTime := time.Now().Add(2 * time.Minute)
	startTime := time.Now()
	blockCfg := BlockConfig{}
	tmpDir := t.TempDir()
	w, err := wal.New(&wal.Config{
		Filepath:       tmpDir,
		Encoding:       backend.EncNone,
		IngestionSlack: 3 * time.Minute,
		Version:        encoding.DefaultEncoding().Version(),
	})
	require.NoError(t, err)

	return newPartitionSectionWriter(logger, 1, endTime, startTime, blockCfg, &mockOverrides{}, w, encoding.DefaultEncoding())
}

func TestPushBytes(t *testing.T) {
	pw := getPartitionWriter(t)

	tenant := "test-tenant"
	traceID := generateTraceID(t)
	now := time.Now()
	startTime := uint64(now.UnixNano())
	endTime := uint64(now.Add(time.Second).UnixNano())
	req := test.MakePushBytesRequest(t, 1, traceID, startTime, endTime)

	err := pw.pushBytes(tenant, req)
	require.NoError(t, err)
}

func TestPushBytes_UnmarshalError(t *testing.T) {
	pw := getPartitionWriter(t)

	tenant := "test-tenant"
	traceID := []byte{1, 2, 3, 4}
	req := &tempopb.PushBytesRequest{
		Ids:    [][]byte{traceID},
		Traces: []tempopb.PreallocBytes{{Slice: []byte{1, 2}}},
	}

	err := pw.pushBytes(tenant, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal trace")
}
