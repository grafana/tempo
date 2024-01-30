package poller

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/grafana/e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/grafana/tempo/cmd/tempo/app"
	util "github.com/grafana/tempo/integration"
	e2eBackend "github.com/grafana/tempo/integration/e2e/backend"
	"github.com/grafana/tempo/pkg/blockboundary"
	"github.com/grafana/tempo/tempodb/backend"
	"github.com/grafana/tempo/tempodb/backend/azure"
	"github.com/grafana/tempo/tempodb/backend/gcs"
	"github.com/grafana/tempo/tempodb/backend/s3"
	"github.com/grafana/tempo/tempodb/blocklist"
)

const (
	configS3      = "config-s3.yaml"
	configAzurite = "config-azurite.yaml"
	configGCS     = "config-gcs.yaml"

	tenant = "test"
)

// OwnsEverythingSharder owns everything.
var OwnsEverythingSharder = ownsEverythingSharder{}

type ownsEverythingSharder struct{}

func (ownsEverythingSharder) Owns(_ string) bool {
	return true
}

func TestPollerOwnership(t *testing.T) {
	testCompactorOwnershipBackends := []struct {
		name       string
		configFile string
	}{
		{
			name:       "s3",
			configFile: configS3,
		},
		{
			name:       "azure",
			configFile: configAzurite,
		},
		{
			name:       "gcs",
			configFile: configGCS,
		},
	}

	logger := log.NewLogfmtLogger(os.Stdout)
	var hhh *e2e.HTTPService
	t.Parallel()
	for _, tc := range testCompactorOwnershipBackends {
		t.Run(tc.name, func(t *testing.T) {
			s, err := e2e.NewScenario("tempo-integration")
			require.NoError(t, err)
			defer s.Close()

			// set up the backend
			cfg := app.Config{}
			buff, err := os.ReadFile(tc.configFile)
			require.NoError(t, err)
			err = yaml.UnmarshalStrict(buff, &cfg)
			require.NoError(t, err)
			hhh, err = e2eBackend.New(s, cfg)
			require.NoError(t, err)

			err = hhh.WaitReady()
			require.NoError(t, err)

			err = hhh.Ready()
			require.NoError(t, err)

			// Give some time for startup
			time.Sleep(1 * time.Second)

			t.Logf("backend: %s", hhh.Endpoint(hhh.HTTPPort()))

			require.NoError(t, util.CopyFileToSharedDir(s, tc.configFile, "config.yaml"))

			var rr backend.RawReader
			var ww backend.RawWriter
			var cc backend.Compactor

			concurrency := 3

			e := hhh.Endpoint(hhh.HTTPPort())
			switch tc.name {
			case "s3":
				cfg.StorageConfig.Trace.S3.ListBlocksConcurrency = concurrency
				cfg.StorageConfig.Trace.S3.Endpoint = e
				cfg.Overrides.UserConfigurableOverridesConfig.Client.S3.Endpoint = e
				rr, ww, cc, err = s3.New(cfg.StorageConfig.Trace.S3)
			case "gcs":
				cfg.StorageConfig.Trace.GCS.ListBlocksConcurrency = concurrency
				cfg.StorageConfig.Trace.GCS.Endpoint = e
				cfg.Overrides.UserConfigurableOverridesConfig.Client.GCS.Endpoint = e
				rr, ww, cc, err = gcs.New(cfg.StorageConfig.Trace.GCS)
			case "azure":
				cfg.StorageConfig.Trace.Azure.Endpoint = e
				cfg.Overrides.UserConfigurableOverridesConfig.Client.Azure.Endpoint = e
				rr, ww, cc, err = azure.New(cfg.StorageConfig.Trace.Azure)
			}
			require.NoError(t, err)

			r := backend.NewReader(rr)
			w := backend.NewWriter(ww)

			blocklistPoller := blocklist.NewPoller(&blocklist.PollerConfig{
				PollConcurrency:     3,
				TenantIndexBuilders: 1,
			}, OwnsEverythingSharder, r, cc, w, logger)

			// Use the block boundaries in the GCS and S3 implementation
			bb := blockboundary.CreateBlockBoundaries(concurrency)
			// Pick a boundary to use for this test
			base := bb[1]
			expected := []uuid.UUID{}

			expected = append(expected, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
			expected = append(expected, uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"))

			// Grab the one before the boundary
			decrementUUIDBytes(base)
			expected = append(expected, uuid.UUID(base))

			incrementUUIDBytes(base)
			expected = append(expected, uuid.UUID(base))

			incrementUUIDBytes(base)
			expected = append(expected, uuid.UUID(base))

			incrementUUIDBytes(base)
			expected = append(expected, uuid.UUID(base))

			writeTenantBlocks(t, w, tenant, expected)

			sort.Slice(expected, func(i, j int) bool { return expected[i].String() < expected[j].String() })
			t.Logf("expected: %v", expected)

			mmResults, cmResults, err := rr.ListBlocks(context.Background(), tenant)
			require.NoError(t, err)
			sort.Slice(mmResults, func(i, j int) bool { return mmResults[i].String() < mmResults[j].String() })
			t.Logf("mmResults: %s", mmResults)
			t.Logf("cmResults: %s", cmResults)

			assert.Equal(t, expected, mmResults)
			assert.Equal(t, len(expected), len(mmResults))
			assert.Equal(t, 0, len(cmResults))

			l := blocklist.New()
			mm, cm, err := blocklistPoller.Do(l)
			require.NoError(t, err)
			t.Logf("mm: %v", mm)
			t.Logf("cm: %v", cm)

			l.ApplyPollResults(mm, cm)

			metas := l.Metas(tenant)

			actual := []uuid.UUID{}
			for _, m := range metas {
				actual = append(actual, m.BlockID)
			}

			sort.Slice(actual, func(i, j int) bool { return actual[i].String() < actual[j].String() })

			assert.Equal(t, expected, actual)
			assert.Equal(t, len(expected), len(metas))
			t.Logf("actual: %v", actual)

			for _, e := range expected {
				assert.True(t, found(e, metas))
			}
		})
	}
}

func found(id uuid.UUID, blockMetas []*backend.BlockMeta) bool {
	for _, b := range blockMetas {
		if b.BlockID == id {
			return true
		}
	}

	return false
}

func writeTenantBlocks(t *testing.T, w backend.Writer, tenant string, blockIDs []uuid.UUID) {
	var err error
	for _, b := range blockIDs {
		meta := &backend.BlockMeta{
			BlockID:  b,
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
