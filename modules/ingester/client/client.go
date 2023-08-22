package client

import (
	"flag"
	"io"
	"time"

	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/middleware"
	ring_client "github.com/grafana/dskit/ring/client"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Config for an ingester client.
type Config struct {
	PoolConfig       ring_client.PoolConfig `yaml:"pool_config,omitempty"`
	RemoteTimeout    time.Duration          `yaml:"remote_timeout,omitempty"`
	GRPCClientConfig grpcclient.Config      `yaml:"grpc_client_config"`
}

type Client struct {
	tempopb.PusherClient
	tempopb.QuerierClient
	grpc_health_v1.HealthClient
	io.Closer
}

// RegisterFlags registers flags.
func (cfg *Config) RegisterFlags(f *flag.FlagSet) {
	cfg.GRPCClientConfig.RegisterFlagsWithPrefix("ingester.client", f)

	f.DurationVar(&cfg.PoolConfig.HealthCheckTimeout, "ingester.client.healthcheck-timeout", 1*time.Second, "Timeout for healthcheck rpcs.")
	f.DurationVar(&cfg.PoolConfig.CheckInterval, "ingester.client.healthcheck-interval", 15*time.Second, "Interval to healthcheck ingesters")
	f.BoolVar(&cfg.PoolConfig.HealthCheckEnabled, "ingester.client.healthcheck-enabled", true, "Healthcheck ingesters.")
	f.DurationVar(&cfg.RemoteTimeout, "ingester.client.timeout", 5*time.Second, "Timeout for ingester client RPCs.")
}

// New returns a new ingester client.
func New(addr string, cfg Config) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
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
		PusherClient:  tempopb.NewPusherClient(conn),
		QuerierClient: tempopb.NewQuerierClient(conn),
		HealthClient:  grpc_health_v1.NewHealthClient(conn),
		Closer:        conn,
	}, nil
}

func instrumentation() ([]grpc.UnaryClientInterceptor, []grpc.StreamClientInterceptor) {
	return []grpc.UnaryClientInterceptor{
			otgrpc.OpenTracingClientInterceptor(opentracing.GlobalTracer()),
			middleware.ClientUserHeaderInterceptor,
		}, []grpc.StreamClientInterceptor{
			otgrpc.OpenTracingStreamClientInterceptor(opentracing.GlobalTracer()),
			middleware.StreamClientUserHeaderInterceptor,
		}
}
