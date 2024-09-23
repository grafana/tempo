package test

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/local"
	"github.com/stretchr/testify/require"
)

func BenchmarkIndexLoad(b *testing.B) {
	var (
		tenant = "benchmark-tenant"
		ctx    = context.Background()
	)

	blockMeta := make([]*backend.BlockMeta, 1000)
	for i := range len(blockMeta) {
		blockMeta[i] = &backend.BlockMeta{
			Version:         "vParquet3",
			BlockID:         backend.NewUUID(),
			TenantID:        tenant,
			StartTime:       time.Now().Add(-50 * time.Minute),
			EndTime:         time.Now().Add(-40 * time.Minute),
			TotalObjects:    10,
			Size_:           12345,
			CompactionLevel: 1,
			Encoding:        backend.EncZstd,
			IndexPageSize:   250000,
			TotalRecords:    124356,
			DataEncoding:    "",
			BloomShardCount: 244,
			FooterSize:      15775,
			DedicatedColumns: backend.DedicatedColumns{
				{Scope: "resource", Name: "namespace", Type: "string"},
				{Scope: "span", Name: "http.method", Type: "string"},
				{Scope: "span", Name: "namespace", Type: "string"},
			},
		}
	}

	rr, rw, _, err := local.New(&local.Config{
		Path: "./test-data",
	})
	require.NoError(b, err)

	w := backend.NewWriter(rw)
	err = w.WriteTenantIndex(ctx, tenant, blockMeta, nil)
	require.NoError(b, err)

	r := backend.NewReader(rr)
	_, _ = r.TenantIndex(ctx, tenant) // read the index once to prime the cache
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = r.TenantIndex(ctx, tenant)
	}
}
