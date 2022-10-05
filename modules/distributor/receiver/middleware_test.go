package receiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/weaveworks/common/user"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/tempo/pkg/util"
)

type assertFunc func(*testing.T, context.Context)

type testConsumer struct {
	t          *testing.T
	assertFunc assertFunc
}

func (tc *testConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func newAssertingConsumer(t *testing.T, assertFunc assertFunc) consumer.Traces {
	return &testConsumer{
		t:          t,
		assertFunc: assertFunc,
	}
}

func (tc *testConsumer) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	tc.assertFunc(tc.t, ctx)
	return nil
}

func TestFakeTenantMiddleware(t *testing.T) {
	m := FakeTenantMiddleware()

	t.Run("injects org id", func(t *testing.T) {
		consumer := newAssertingConsumer(t, func(t *testing.T, ctx context.Context) {
			orgID, err := user.ExtractOrgID(ctx)
			require.NoError(t, err)
			require.Equal(t, orgID, util.FakeTenantID)
		})

		ctx := context.Background()
		require.NoError(t, m.Wrap(consumer).ConsumeTraces(ctx, ptrace.Traces{}))
	})
}

func TestMultiTenancyMiddleware(t *testing.T) {
	m := MultiTenancyMiddleware()

	t.Run("injects org id", func(t *testing.T) {
		tenantID := "test-tenant-id"

		consumer := newAssertingConsumer(t, func(t *testing.T, ctx context.Context) {
			orgID, err := user.ExtractOrgID(ctx)
			require.NoError(t, err)
			require.Equal(t, orgID, tenantID)
		})

		ctx := metadata.NewIncomingContext(
			context.Background(),
			metadata.Pairs("X-Scope-OrgID", tenantID),
		)
		require.NoError(t, m.Wrap(consumer).ConsumeTraces(ctx, ptrace.Traces{}))
	})

	t.Run("returns error if org id cannot be extracted", func(t *testing.T) {
		// no need to assert anything, because the wrapped function is never called
		consumer := newAssertingConsumer(t, func(t *testing.T, ctx context.Context) {})
		ctx := context.Background()
		require.EqualError(t, m.Wrap(consumer).ConsumeTraces(ctx, ptrace.Traces{}), "no org id")
	})
}
