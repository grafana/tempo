package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkTenantIndexColumnLayout(b *testing.B) {
	for _, tc := range []struct {
		name            string
		blocks, layouts int
	}{{"shared1000", 1000, 10}, {"unique4096", 4096, 4096}, {"none1000", 1000, 0}} {
		idx := tenantIndexWriteFixture(tc.blocks, tc.layouts)
		payload, err := idx.marshalPb()
		require.NoError(b, err)
		reader := NewReader(&MockRawReader{R: payload})
		b.Run(tc.name+"/read", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				got, err := reader.TenantIndex(context.Background(), "test-tenant")
				if err != nil {
					b.Fatal(err)
				}
				_ = got
			}
		})
		b.Run(tc.name+"/readWrite", func(b *testing.B) {
			w := NewWriter(&discardIndexWriter{})
			b.ReportAllocs()
			for b.Loop() {
				got, err := reader.TenantIndex(context.Background(), "test-tenant")
				if err != nil {
					b.Fatal(err)
				}
				_ = got
				if err := w.WriteTenantIndex(context.Background(), "test-tenant", got.Meta, got.CompactedMeta); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(tc.name+"/constructWrite", func(b *testing.B) {
			w := NewWriter(&discardIndexWriter{})
			b.ReportAllocs()
			for b.Loop() {
				got := tenantIndexWriteFixture(tc.blocks, tc.layouts)
				if err := w.WriteTenantIndex(context.Background(), "test-tenant", got.Meta, got.CompactedMeta); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
