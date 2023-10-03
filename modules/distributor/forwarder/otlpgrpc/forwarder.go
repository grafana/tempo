package otlpgrpc

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/middleware"
	grpcmw "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/multierr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Forwarder struct {
	cfg         Config
	logger      log.Logger
	connections map[string]*grpc.ClientConn
	clients     map[string]ptraceotlp.GRPCClient
	initialized bool
	mu          *sync.RWMutex
}

func NewForwarder(cfg Config, logger log.Logger) (*Forwarder, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate config: %w", err)
	}

	return &Forwarder{
		cfg:         cfg,
		logger:      logger,
		connections: make(map[string]*grpc.ClientConn),
		clients:     make(map[string]ptraceotlp.GRPCClient),
		initialized: false,
		mu:          &sync.RWMutex{},
	}, nil
}

// Dial creates client connections and clients based on config.
// Dial is expected to be called only once.
func (f *Forwarder) Dial(ctx context.Context, opts ...grpc.DialOption) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.initialized {
		return errors.New("already initialized")
	}

	connections := make(map[string]*grpc.ClientConn)
	clients := make(map[string]ptraceotlp.GRPCClient, len(f.cfg.Endpoints))
	for _, endpoint := range f.cfg.Endpoints {
		client, conn, err := f.newTraceOTLPGRPCClientAndConn(ctx, endpoint, f.cfg.TLS, opts...)
		if err != nil {
			return fmt.Errorf("failed to create new trace otlp grpc client: %w", err)
		}

		connections[endpoint] = conn
		clients[endpoint] = client
	}

	f.connections = connections
	f.clients = clients
	f.initialized = true

	return nil
}

func (f *Forwarder) ForwardTraces(ctx context.Context, traces ptrace.Traces) error {
	req := ptraceotlp.NewExportRequestFromTraces(traces)

	var errs []error
	f.mu.RLock()
	for endpoint, client := range f.clients {
		if _, err := client.Export(ctx, req); err != nil {
			errs = append(errs, fmt.Errorf("failed to export trace to endpoint=%s: %w", endpoint, err))
		}
	}
	f.mu.RUnlock()

	return multierr.Combine(errs...)
}

func (f *Forwarder) Shutdown(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var errs []error
	for endpoint, conn := range f.connections {
		if err := conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close grpc connection for endpoint=%s: %w", endpoint, err))
		}

		delete(f.connections, endpoint)
	}

	return multierr.Combine(errs...)
}

func (f *Forwarder) newTraceOTLPGRPCClientAndConn(ctx context.Context, endpoint string, cfg TLSConfig, opts ...grpc.DialOption) (ptraceotlp.GRPCClient, *grpc.ClientConn, error) {
	var creds credentials.TransportCredentials
	if cfg.Insecure {
		creds = insecure.NewCredentials()
	} else {
		var err error
		creds, err = credentials.NewClientTLSFromFile(cfg.CertFile, "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create new server tls from file: %w", err)
		}
	}

	unaryClientInterceptor, streamClientInterceptor := instrumentation()

	opts = append(
		opts,
		grpc.WithTransportCredentials(creds),
		grpc.WithUnaryInterceptor(grpcmw.ChainUnaryClient(unaryClientInterceptor...)),
		grpc.WithStreamInterceptor(grpcmw.ChainStreamClient(streamClientInterceptor...)),
	)

	grpcClientConn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, nil, err
	}

	return ptraceotlp.NewGRPCClient(grpcClientConn), grpcClientConn, nil
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
