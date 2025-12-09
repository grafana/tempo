package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	mathrand "math/rand/v2"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/e2e"
	"github.com/grafana/tempo/integration/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/blocklist"
	"github.com/grafana/tempo/tempodb/encoding/vparquet3"
)

const (
	tenant = "test"
)

// OwnsEverythingSharder owns everything.
var OwnsEverythingSharder = ownsEverythingSharder{}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func TestPollerOwnership(t *testing.T) {
	storageBackendTestPermutations := []struct {
		name   string
		prefix string
	}{
		{
			name:   "empty-string-prefix",
			prefix: "",
		},
		{
			name: "no-prefix",
		},
		{
			name:   "prefix",
			prefix: "a/b/c/",
		},
		{
			name:   "prefix-no-trailing-slash",
			prefix: "a/b/c",
		},
	}

	logger := log.NewLogfmtLogger(os.Stdout)

	for _, pc := range storageBackendTestPermutations {
		t.Run(pc.name, func(t *testing.T) {
			util.WithTempoHarness(t, util.TestHarnessConfig{
				Components: util.ComponentsObjectStorage,
			}, func(h *util.TempoHarness) {
				// Get the config to determine backend type and settings
				cfg, err := h.GetConfig()
				require.NoError(t, err)

				var rr backend.RawReader
				var ww backend.RawWriter
				var cc backend.Compactor

				listBlockConcurrency := 10

				// Get backend endpoint from the harness
				httpService, ok := h.Backend.(*e2e.HTTPService)
				require.True(t, ok, "backend should be an HTTPService")
				e := httpService.Endpoint(httpService.HTTPPort())

				switch cfg.StorageConfig.Trace.Backend {
				case backend.S3:
					cfg.StorageConfig.Trace.S3.ListBlocksConcurrency = listBlockConcurrency
					cfg.StorageConfig.Trace.S3.Endpoint = e
					cfg.StorageConfig.Trace.S3.Prefix = pc.prefix
					rr, ww, cc, err = s3.New(cfg.StorageConfig.Trace.S3)
				case backend.GCS:
					cfg.StorageConfig.Trace.GCS.ListBlocksConcurrency = listBlockConcurrency
					cfg.StorageConfig.Trace.GCS.Endpoint = e
					cfg.StorageConfig.Trace.GCS.Prefix = pc.prefix
					rr, ww, cc, err = gcs.New(cfg.StorageConfig.Trace.GCS)
				case backend.Azure:
					cfg.StorageConfig.Trace.Azure.Endpoint = e
					cfg.StorageConfig.Trace.Azure.Prefix = pc.prefix
					rr, ww, cc, err = azure.New(cfg.StorageConfig.Trace.Azure)
				}
				require.NoError(t, err)

				r := backend.NewReader(rr)
				w := backend.NewWriter(ww)

				blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
					PollConcurrency:        3,
					TenantPollConcurrency:  3,
					TenantIndexBuilders:    1,
					EmptyTenantDeletionAge: 10 * time.Minute,
				}, OwnsEverythingSharder, r, cc, w, logger)

				// Use the block boundaries in the GCS and S3 implementation
				bb := blockboundary.CreateBlockBoundaries(listBlockConcurrency)

				tenantCount := 250
				tenantExpected := map[string][]uuid.UUID{}

				// Push some data to a few tenants
				for i := 0; i < tenantCount; i++ {
					testTenant := tenant + strconv.Itoa(i)
					tenantExpected[testTenant] = pushBlocksToTenant(t, testTenant, bb, w)

					mmResults, cmResults, listBlocksErr := rr.ListBlocks(context.Background(), testTenant)
					require.NoError(t, listBlocksErr)
					sort.Slice(mmResults, func(i, j int) bool { return mmResults[i].String() < mmResults[j].String() })

					require.Equal(t, tenantExpected[testTenant], mmResults)
					require.Equal(t, len(tenantExpected[testTenant]), len(mmResults))
					require.Equal(t, 0, len(cmResults))
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				l := blocklist.New()
				mm, cm, err := blocklistPoller.Do(ctx, l)
				require.NoError(t, err)

				l.ApplyPollResults(mm, cm)

				for testTenant, expected := range tenantExpected {
					metas := l.Metas(testTenant)

					actual := []uuid.UUID{}
					for _, m := range metas {
						actual = append(actual, (uuid.UUID)(m.BlockID))
					}

					sort.Slice(actual, func(i, j int) bool { return actual[i].String() < actual[j].String() })

					assert.Equal(t, expected, actual)
					assert.Equal(t, len(expected), len(metas))

					for _, e := range expected {
						assert.True(t, found(e, metas))
					}
				}
			})
		})
	}
}

func TestTenantDeletion(t *testing.T) {
	storageBackendTestPermutations := []struct {
		name   string
		prefix string
	}{
		{
			name:   "empty-string-prefix",
			prefix: "",
		},
		{
			name: "no-prefix",
		},
		{
			name:   "prefix",
			prefix: "a/b/c/",
		},
		{
			name:   "prefix-no-trailing-slash",
			prefix: "a/b/c",
		},
	}

	logger := log.NewLogfmtLogger(os.Stdout)

	for _, pc := range storageBackendTestPermutations {
		t.Run(pc.name, func(t *testing.T) {
			util.WithTempoHarness(t, util.TestHarnessConfig{
				Components: util.ComponentsObjectStorage,
			}, func(h *util.TempoHarness) {
				// Get the config to determine backend type and settings
				cfg, err := h.GetConfig()
				require.NoError(t, err)

				var rr backend.RawReader
				var ww backend.RawWriter
				var cc backend.Compactor

				listBlockConcurrency := 3
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Get backend endpoint from the harness
				httpService, ok := h.Backend.(*e2e.HTTPService)
				require.True(t, ok, "backend should be an HTTPService")
				e := httpService.Endpoint(httpService.HTTPPort())

				switch cfg.StorageConfig.Trace.Backend {
				case backend.S3:
					cfg.StorageConfig.Trace.S3.Endpoint = e
					cfg.StorageConfig.Trace.S3.ListBlocksConcurrency = listBlockConcurrency
					cfg.StorageConfig.Trace.S3.Prefix = pc.prefix
					rr, ww, cc, err = s3.New(cfg.StorageConfig.Trace.S3)
				case backend.GCS:
					cfg.StorageConfig.Trace.GCS.Endpoint = e
					cfg.StorageConfig.Trace.GCS.ListBlocksConcurrency = listBlockConcurrency
					cfg.StorageConfig.Trace.GCS.Prefix = pc.prefix
					rr, ww, cc, err = gcs.New(cfg.StorageConfig.Trace.GCS)
				case backend.Azure:
					cfg.StorageConfig.Trace.Azure.Endpoint = e
					cfg.StorageConfig.Trace.Azure.Prefix = pc.prefix
					rr, ww, cc, err = azure.New(cfg.StorageConfig.Trace.Azure)
				}
				require.NoError(t, err)

				r := backend.NewReader(rr)
				w := backend.NewWriter(ww)

				// Tenant deletion is not enabled by default
				blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
					PollConcurrency:        3,
					TenantIndexBuilders:    1,
					EmptyTenantDeletionAge: 100 * time.Millisecond,
					TenantPollConcurrency:  3,
				}, OwnsEverythingSharder, r, cc, w, logger)

				l := blocklist.New()
				mm, cm, err := blocklistPoller.Do(ctx, l)
				require.NoError(t, err)
				t.Logf("mm: %v", mm)
				t.Logf("cm: %v", cm)

				tennants, err := r.Tenants(ctx)
				require.NoError(t, err)
				require.Equal(t, 0, len(tennants))

				writeBadBlockFiles(t, ww, rr, tenant)

				// Now we should have a tenant
				tennants, err = r.Tenants(ctx)
				require.NoError(t, err)
				require.Equal(t, 1, len(tennants))

				time.Sleep(500 * time.Millisecond)

				_, _, err = blocklistPoller.Do(ctx, l)
				require.NoError(t, err)

				tennants, err = r.Tenants(ctx)
				t.Logf("tennants: %v", tennants)
				require.NoError(t, err)
				require.Equal(t, 1, len(tennants))

				// Create a new poller with tenantion deletion enabled
				blocklistPoller = blocklist.NewPoller(&blocklist.PollerConfig{
					PollConcurrency:            3,
					TenantIndexBuilders:        1,
					EmptyTenantDeletionAge:     100 * time.Millisecond,
					EmptyTenantDeletionEnabled: true,
					TenantPollConcurrency:      3,
				}, OwnsEverythingSharder, r, cc, w, logger)

				// Again
				_, _, err = blocklistPoller.Do(ctx, l)
				require.NoError(t, err)

				tennants, err = r.Tenants(ctx)
				require.NoError(t, err)
				require.Equal(t, 0, len(tennants))
			})
		})
	}
}

func found(id uuid.UUID, blockMetas []*backend.BlockMeta) bool {
	for _, b := range blockMetas {
		if (uuid.UUID)(b.BlockID) == id {
			return true
		}
	}

	return false
}

func writeTenantBlocks(t *testing.T, w backend.Writer, tenant string, blockIDs []uuid.UUID) {
	var err error
	for _, b := range blockIDs {
		meta := &backend.BlockMeta{
			BlockID:  backend.UUID(b),
			TenantID: tenant,
		}

		err = w.WriteBlockMeta(context.Background(), meta)
		require.NoError(t, err)
	}
}

func decrementUUIDBytes(uuidBytes []byte) {
	for i := len(uuidBytes) - 1; i >= 0; i-- {
		if uuidBytes[i] > 0 {
			uuidBytes[i]--
			break
		}

		uuidBytes[i] = 255 // Wrap around if the byte is 0
	}
}

func incrementUUIDBytes(uuidBytes []byte) {
	for i := len(uuidBytes) - 1; i >= 0; i-- {
		if uuidBytes[i] < 255 {
			uuidBytes[i]++
			break
		}

		uuidBytes[i] = 0 // Wrap around if the byte is 255
	}
}

func writeBadBlockFiles(t *testing.T, ww backend.RawWriter, rr backend.RawReader, tenant string) {
	t.Logf("writing bad block files")

	ctx := context.Background()

	token := make([]byte, 32)
	_, err := rand.Read(token)
	require.NoError(t, err)

	err = ww.Write(
		ctx,
		vparquet3.DataFileName,
		backend.KeyPath([]string{tenant, uuid.New().String()}),
		bytes.NewReader(token),
		int64(len(token)), nil)

	require.NoError(t, err)

	items, err := rr.List(context.Background(), backend.KeyPath([]string{tenant}))
	require.NoError(t, err)
	t.Logf("items: %v", items)

	var found []string
	f := func(opts backend.FindMatch) {
		found = append(found, opts.Key)
	}

	err = rr.Find(ctx, backend.KeyPath{}, f)
	require.NoError(t, err)
	t.Logf("items: %v", found)
}

func pushBlocksToTenant(t *testing.T, tenant string, bb [][]byte, w backend.Writer) []uuid.UUID {
	// Randomly pick a block boundary
	r := mathrand.IntN(len(bb))

	base := bb[r]
	t.Logf("base: %v", base)
	expected := []uuid.UUID{}

	// Include the min and max in each tenant for testing
	expected = append(expected, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
	expected = append(expected, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"))

	// If we are above zero, then we have room to decrement
	if r > 0 {
		decrementUUIDBytes(base)
		expected = append(expected, uuid.UUID(base))
	}

	// If we are n-1 then we have room to increment
	if r < len(bb)-1 {
		// Grab the one after the boundary
		incrementUUIDBytes(base)
		expected = append(expected, uuid.UUID(base))
	}

	// If we are n-2 then we have room to increment again
	if r < len(bb)-2 {
		// Grab the one after the boundary
		incrementUUIDBytes(base)
		expected = append(expected, uuid.UUID(base))
	}

	// If we are n-3 then we have room to increment again
	if r < len(bb)-3 {
		// Grab the one after the boundary
		incrementUUIDBytes(base)
		expected = append(expected, uuid.UUID(base))
	}

	// Write the blocks using the expectaed block IDs
	writeTenantBlocks(t, w, tenant, expected)

	sort.Slice(expected, func(i, j int) bool { return expected[i].String() < expected[j].String() })
	// t.Logf("expected: %v", expected)

	return expected
}
