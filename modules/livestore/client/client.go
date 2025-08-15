package client

import (
	"flag"
	"io"
	"time"

	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/middleware"
	ring_client "github.com/grafana/dskit/ring/client"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Config for a livestore client.
type Config struct {
	PoolConfig       ring_client.PoolConfig `yaml:"pool_config,omitempty"`
	RemoteTimeout    time.Duration          `yaml:"remote_timeout,omitempty"`
	GRPCClientConfig grpcclient.Config      `yaml:"grpc_client_config"`
}

type Client struct {
	tempopb.QuerierClient
	tempopb.MetricsGeneratorClient
	grpc_health_v1.HealthClient
	io.Closer
}

// RegisterFlags registers flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.GRPCClientConfig.RegisterFlagsWithPrefix("livestore.client", f)

	f.DurationVar(&cfg.PoolConfig.HealthCheckTimeout, "livestore.client.healthcheck-timeout", 1*time.Second, "Timeout for healthcheck rpcs.")
	f.DurationVar(&cfg.PoolConfig.CheckInterval, "livestore.client.healthcheck-interval", 15*time.Second, "Interval to healthcheck livestores")
	f.BoolVar(&cfg.PoolConfig.HealthCheckEnabled, "livestore.client.healthcheck-enabled", true, "Healthcheck livestores.")
	f.DurationVar(&cfg.RemoteTimeout, "livestore.client.timeout", 5*time.Second, "Timeout for livestore client RPCs.")
}

// New returns a new livestore client.
func New(addr string, cfg Config) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	}

	instrumentationOpts, err := cfg.GRPCClientConfig.DialOption(instrumentation())
	if err != nil {
		return nil, err
	}

	opts = append(opts, instrumentationOpts...)
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		QuerierClient:          tempopb.NewQuerierClient(conn),
		MetricsGeneratorClient: tempopb.NewMetricsGeneratorClient(conn),
		HealthClient:           grpc_health_v1.NewHealthClient(conn),
		Closer:                 conn,
	}, nil
}

func instrumentation() ([]grpc.UnaryClientInterceptor, []grpc.StreamClientInterceptor, middleware.InvalidClusterValidationReporter) {
	return []grpc.UnaryClientInterceptor{
			middleware.ClientUserHeaderInterceptor,
		}, []grpc.StreamClientInterceptor{
			middleware.StreamClientUserHeaderInterceptor,
		},
		middleware.NoOpInvalidClusterValidationReporter
}
