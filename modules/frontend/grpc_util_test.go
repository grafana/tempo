package frontend

import (
	"context"
	"testing"

	"github.com/grafana/tempo/pkg/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

func TestHeadersFromGrpcContextCopiesPluginID(t *testing.T) {
	md := metadata.New(map[string]string{
		"authorization":    "Bearer abc",
		"x-scope-orgid":    "tenant-a",
		"x-plugin-id":      "grafana-assistant",
		"x-ignored-header": "nope",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	headers := headersFromGrpcContext(ctx)
	require.Equal(t, "Bearer abc", headers.Get("Authorization"))
	require.Equal(t, "tenant-a", headers.Get("X-Scope-OrgID"))
	require.Equal(t, "grafana-assistant", headers.Get(api.HeaderPluginID))
	require.Empty(t, headers.Get("X-Ignored-Header"))
}

func TestHeadersFromGrpcContextWithoutMetadata(t *testing.T) {
	headers := headersFromGrpcContext(context.Background())
	require.Empty(t, headers)
}
