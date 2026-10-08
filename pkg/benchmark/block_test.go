package benchmark

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/v3/pkg/benchmark/internal/benchtest"
)

func TestLoadLocalBlock(t *testing.T) {
	want, _, bucket := benchtest.Block(t, 20)

	path := filepath.Join(bucket, want.TenantID, want.BlockID.String())
	got, r, err := LoadLocalBlock(context.Background(), path)
	require.NoError(t, err)
	require.NotNil(t, r)

	// meta.json is JSON, so times lose their monotonic reading; compare fields.
	require.Equal(t, want.BlockID, got.BlockID)
	require.Equal(t, want.TenantID, got.TenantID)
	require.Equal(t, want.Version, got.Version)
	require.Equal(t, want.TotalObjects, got.TotalObjects)
	require.Equal(t, want.TotalRecords, got.TotalRecords)
	require.Equal(t, want.Size_, got.Size_)
	require.True(t, want.StartTime.Equal(got.StartTime))
	require.True(t, want.EndTime.Equal(got.EndTime))

	// A trailing separator must address the same block.
	got, _, err = LoadLocalBlock(context.Background(), path+string(filepath.Separator))
	require.NoError(t, err)
	require.Equal(t, want.BlockID, got.BlockID)
}

func TestLoadLocalBlockRejectsBadPaths(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{"block dir is not a UUID", filepath.Join(t.TempDir(), "tenant", "not-a-uuid"), "not a block directory"},
		{"no tenant above the block", "/00000000-0000-0000-0000-000000000000", "no tenant directory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := LoadLocalBlock(ctx, tc.path)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
