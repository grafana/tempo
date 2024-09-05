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

// Config for a generator client.
type Config struct {
	PoolConfig       ring_client.PoolConfig `yaml:"pool_config,omitempty"`
	RemoteTimeout    time.Duration          `yaml:"remote_timeout,omitempty"`
	GRPCClientConfig grpcclient.Config      `yaml:"grpc_client_config"`
}

type Client struct {
	tempopb.MetricsGeneratorClient
	grpc_health_v1.HealthClient
	io.Closer
}

// RegisterFlags registers flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.GRPCClientConfig.RegisterFlagsWithPrefix("generator.client", f)

	f.DurationVar(&cfg.PoolConfig.HealthCheckTimeout, "generator.client.healthcheck-timeout", 1*time.Second, "Timeout for healthcheck rpcs.")
	f.DurationVar(&cfg.PoolConfig.CheckInterval, "generator.client.healthcheck-interval", 15*time.Second, "Interval to healthcheck generators")
	f.BoolVar(&cfg.PoolConfig.HealthCheckEnabled, "generator.client.healthcheck-enabled", true, "Healthcheck generators.")
	f.DurationVar(&cfg.RemoteTimeout, "generator.client.timeout", 5*time.Second, "Timeout for generator client RPCs.")
}

// New returns a new generator client.
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
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		MetricsGeneratorClient: tempopb.NewMetricsGeneratorClient(conn),
		HealthClient:           grpc_health_v1.NewHealthClient(conn),
		Closer:                 conn,
	}, nil
}

func instrumentation() ([]grpc.UnaryClientInterceptor, []grpc.StreamClientInterceptor) {
	return []grpc.UnaryClientInterceptor{
			middleware.ClientUserHeaderInterceptor,
		}, []grpc.StreamClientInterceptor{
			middleware.StreamClientUserHeaderInterceptor,
		}
}
