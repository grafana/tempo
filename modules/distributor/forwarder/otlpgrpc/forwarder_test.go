package otlpgrpc

import (
	"context"
	"net"
	"testing"

	"github.com/go-kit/log"
	"github.com/grafana/dskit/user"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
	"go.uber.org/atomic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type mockGRPCServer struct {
	ptraceotlp.UnimplementedGRPCServer
	req ptraceotlp.ExportRequest
}

func (m *mockGRPCServer) Export(_ context.Context, req ptraceotlp.ExportRequest) (ptraceotlp.ExportResponse, error) {
	m.req = req
	return ptraceotlp.NewExportResponse(), nil
}

func newForwarder(t *testing.T, cfg Config, logger log.Logger) *Forwarder {
	t.Helper()

	f, err := NewForwarder(cfg, logger)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, f.Shutdown(context.Background()))
	})

	return f
}

func newListener(t *testing.T, srv ptraceotlp.GRPCServer) *bufconn.Listener {
	t.Helper()

	const size = 1024 * 1024
	l := bufconn.Listen(size)
	t.Cleanup(func() {
		err := l.Close()
		require.NoError(t, err)
	})

	s := grpc.NewServer()
	t.Cleanup(func() {
		s.GracefulStop()
	})

	ptraceotlp.RegisterGRPCServer(s, srv)
	go func() {
		require.NoError(t, s.Serve(l))
	}()

	return l
}

type contextDialer func(context.Context, string) (net.Conn, error)

func newContextDialer(l *bufconn.Listener) contextDialer {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		return l.DialContext(ctx)
	}
}

type countingConn struct {
	net.Conn
	closeCount atomic.Int32
	waitClose  chan struct{}
}

func (c *countingConn) Close() error {
	defer func() {
		select {
		case <-c.waitClose:
			return
		default:
		}

		close(c.waitClose)
	}()

	c.closeCount.Add(1)
	return c.Conn.Close()
}

func newContextDialerWithCountingConn(l *bufconn.Listener) (contextDialer, *countingConn) {
	countingConn := &countingConn{
		waitClose: make(chan struct{}),
	}

	return func(ctx context.Context, _ string) (net.Conn, error) {
		conn, err := l.DialContext(ctx)
		if err != nil {
			return nil, err
		}

		countingConn.Conn = conn

		return countingConn, nil
	}, countingConn
}

type fatalError struct{}

func (fatalError) Error() string   { return "context dialer error" }
func (fatalError) Temporary() bool { return false }

func newFailingContextDialer() contextDialer {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return nil, fatalError{}
	}
}

func TestNewForwarder_ReturnsNoErrorAndNonNilForwarderWithValidConfig(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: nil,
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()

	// When
	got, err := NewForwarder(cfg, logger)

	// Then
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestNewForwarder_ReturnsErrorAndNilForwarderWithInvalidConfig(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: nil,
		TLS:       TLSConfig{Insecure: false},
	}
	logger := log.NewNopLogger()

	// When
	got, err := NewForwarder(cfg, logger)

	// Then
	require.Error(t, err)
	require.Nil(t, got)
}

func Test_Forwarder_Dial_ReturnsNoErrorWithWorkingDialer(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	l := newListener(t, nil)
	d := newContextDialer(l)

	// When
	err := f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())

	// Then
	require.NoError(t, err)
}

func Test_Forwarder_Dial_ReturnsErrorWithFailingContextDialer(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	d := newFailingContextDialer()

	// When
	err := f.Dial(
		context.Background(),
		grpc.WithContextDialer(d),
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
	)

	// Then
	require.Error(t, err)
}

func Test_Forwarder_Dial_ReturnsErrorWithCancelledContext(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	l := newListener(t, nil)
	d := newContextDialer(l)
	ctx, cancel := context.WithCancel(context.Background())

	// Ensure the server is ready
	err := f.Dial(ctx, grpc.WithContextDialer(d), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))
	require.NoError(t, err)

	cancel()

	// When
	err = f.Dial(ctx, grpc.WithContextDialer(d), grpc.WithBlock(), grpc.FailOnNonTempDialError(true))

	// Then
	require.Error(t, err)
}

func Test_Forwarder_Dial_ReturnsErrorWhenCalledSecondTime(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	l := newListener(t, nil)
	d := newContextDialer(l)

	// When
	err := f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())

	// Then
	require.NoError(t, err)

	// When
	err = f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())

	// Then
	require.Error(t, err)
}

func Test_Forwarder_ForwardTraces_ReturnsNoErrorAndSentTracesMatchReceivedTraces(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	srv := &mockGRPCServer{}
	l := newListener(t, srv)
	d := newContextDialer(l)
	err := f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())
	require.NoError(t, err)
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")
	ctx := user.InjectOrgID(context.Background(), "123")

	// When
	err = f.ForwardTraces(ctx, traces)

	// Then
	require.NoError(t, err)
	require.Equal(t, traces, srv.req.Traces())
}

func Test_Forwarder_ForwardTraces_ReturnsErrorWithNoOrgIDInContext(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f := newForwarder(t, cfg, logger)
	srv := &mockGRPCServer{}
	l := newListener(t, srv)
	d := newContextDialer(l)
	err := f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())
	require.NoError(t, err)
	traces := ptrace.NewTraces()
	traces.ResourceSpans().AppendEmpty().SetSchemaUrl("testURL")

	// When
	err = f.ForwardTraces(context.Background(), traces)

	// Then
	require.Error(t, err)
}

func Test_Forwarder_Shutdown_CallsCloseOnConnection(t *testing.T) {
	// Given
	cfg := Config{
		Endpoints: []string{"test:1234"},
		TLS:       TLSConfig{Insecure: true},
	}
	logger := log.NewNopLogger()
	f, err := NewForwarder(cfg, logger)
	require.NoError(t, err)
	srv := &mockGRPCServer{}
	l := newListener(t, srv)
	d, conn := newContextDialerWithCountingConn(l)
	err = f.Dial(context.Background(), grpc.WithContextDialer(d), grpc.WithBlock())
	require.NoError(t, err)

	// When
	err = f.Shutdown(context.Background())

	// Then
	<-conn.waitClose
	require.NoError(t, err)
	require.GreaterOrEqual(t, conn.closeCount.Load(), int32(1))
}
