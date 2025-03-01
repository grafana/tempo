package client

import (
	"flag"
	"fmt"
	"io"

	"github.com/grafana/dskit/grpcclient"
	"github.com/grafana/dskit/middleware"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/grafana/tempo/pkg/tempopb"
)

// Config for a backendscheduler client.
type Config struct {
	// RemoteTimeout    time.Duration     `yaml:"remote_timeout,omitempty"`
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

	// f.DurationVar(&cfg.RemoteTimeout, "backendscheduler.client.timeout", 5*time.Second, "Timeout for backendscheduler client RPCs.")
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
		BackendSchedulerClient: tempopb.NewBackendSchedulerClient(conn),
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
