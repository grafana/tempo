package client

import (
	"flag"
	"fmt"
	"io"

	"github.com/grafana/dskit/grpcclient"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Config for a backendscheduler client.
type Config struct {
	GRPCClientConfig grpcclient.Config `yaml:"grpc_client_config"`
}

type Client struct {
	tempopb.BackendSchedulerClient
	grpc_health_v1.HealthClient
	io.Closer
}

// RegisterFlags registers flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.GRPCClientConfig.RegisterFlagsWithPrefix("backendscheduler.client", f)
}

// New returns a new backendscheduler client.
func New(addr string, cfg Config) (*Client, error) {
	if addr == "" {
		return nil, fmt.Errorf("backend scheduler address is required")
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	instrumentationOpts, err := cfg.GRPCClientConfig.DialOption(nil, nil)
	if err != nil {
		return nil, err
	}

	opts = append(opts, instrumentationOpts...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		BackendSchedulerClient: tempopb.NewBackendSchedulerClient(conn),
		HealthClient:           grpc_health_v1.NewHealthClient(conn),
		Closer:                 conn,
	}, nil
}
