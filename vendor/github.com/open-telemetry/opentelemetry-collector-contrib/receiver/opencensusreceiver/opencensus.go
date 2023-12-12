// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package opencensusreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver"

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	gatewayruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/cors"
	"github.com/soheilhy/cmux"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/ocmetrics"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/octrace"
)

// ocReceiver is the type that exposes Trace and Metrics reception.
type ocReceiver struct {
	mu                 sync.Mutex
	ln                 net.Listener
	serverGRPC         *grpc.Server
	serverHTTP         *http.Server
	gatewayMux         *gatewayruntime.ServeMux
	corsOrigins        []string
	grpcServerSettings configgrpc.GRPCServerSettings

	traceReceiver   *octrace.Receiver
	metricsReceiver *ocmetrics.Receiver

	traceConsumer   consumer.Traces
	metricsConsumer consumer.Metrics

	startTracesReceiverOnce  sync.Once
	startMetricsReceiverOnce sync.Once

	settings receiver.CreateSettings
}

// newOpenCensusReceiver just creates the OpenCensus receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOpenCensusReceiver(
	transport string,
	addr string,
	tc consumer.Traces,
	mc consumer.Metrics,
	settings receiver.CreateSettings,
	opts ...ocOption,
) (*ocReceiver, error) {
	// TODO: (@odeke-em) use options to enable address binding changes.
	ln, err := net.Listen(transport, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to address %q: %w", addr, err)
	}

	ocr := &ocReceiver{
		ln:              ln,
		corsOrigins:     []string{}, // Disable CORS by default.
		gatewayMux:      gatewayruntime.NewServeMux(),
		traceConsumer:   tc,
		metricsConsumer: mc,
		settings:        settings,
	}

	for _, opt := range opts {
		opt.withReceiver(ocr)
	}

	return ocr, nil
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (ocr *ocReceiver) Start(_ context.Context, host component.Host) error {
	hasConsumer := false
	if ocr.traceConsumer != nil {
		hasConsumer = true
		if err := ocr.registerTraceConsumer(host); err != nil {
			return err
		}
	}

	if ocr.metricsConsumer != nil {
		hasConsumer = true
		if err := ocr.registerMetricsConsumer(host); err != nil {
			return err
		}
	}

	if !hasConsumer {
		return errors.New("cannot start receiver: no consumers were specified")
	}

	if err := ocr.startServer(host); err != nil {
		return err
	}

	// At this point we've successfully started all the services/receivers.
	// Add other start routines here.
	return nil
}

func (ocr *ocReceiver) registerTraceConsumer(host component.Host) error {
	var err error

	ocr.startTracesReceiverOnce.Do(func() {
		ocr.traceReceiver, err = octrace.New(ocr.traceConsumer, ocr.settings)
		if err != nil {
			return
		}

		var srv *grpc.Server
		srv, err = ocr.grpcServer(host)
		if err != nil {
			return
		}

		agenttracepb.RegisterTraceServiceServer(srv, ocr.traceReceiver)

	})

	return err
}

func (ocr *ocReceiver) registerMetricsConsumer(host component.Host) error {
	var err error

	ocr.startMetricsReceiverOnce.Do(func() {
		ocr.metricsReceiver, err = ocmetrics.New(ocr.metricsConsumer, ocr.settings)
		if err != nil {
			return
		}

		var srv *grpc.Server
		srv, err = ocr.grpcServer(host)
		if err != nil {
			return
		}

		agentmetricspb.RegisterMetricsServiceServer(srv, ocr.metricsReceiver)
	})
	return err
}

func (ocr *ocReceiver) grpcServer(host component.Host) (*grpc.Server, error) {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	if ocr.serverGRPC == nil {
		var err error
		ocr.serverGRPC, err = ocr.grpcServerSettings.ToServer(host, ocr.settings.TelemetrySettings)
		if err != nil {
			return nil, err
		}
	}

	return ocr.serverGRPC, nil
}

// Shutdown is a method to turn off receiving.
func (ocr *ocReceiver) Shutdown(context.Context) error {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	var err error
	if ocr.serverHTTP != nil {
		err = ocr.serverHTTP.Close()
	}

	if ocr.ln != nil {
		_ = ocr.ln.Close()
	}

	// TODO: @(odeke-em) investigate what utility invoking (*grpc.Server).Stop()
	// gives us yet we invoke (net.Listener).Close().
	// Sure (*grpc.Server).Stop() enables proper shutdown but imposes
	// a painful and artificial wait time that goes into 20+seconds yet most of our
	// tests and code should be reactive in less than even 1second.
	// ocr.serverGRPC.Stop()

	return err
}

func (ocr *ocReceiver) httpServer() *http.Server {
	ocr.mu.Lock()
	defer ocr.mu.Unlock()

	if ocr.serverHTTP == nil {
		var mux http.Handler = ocr.gatewayMux
		if len(ocr.corsOrigins) > 0 {
			co := cors.Options{AllowedOrigins: ocr.corsOrigins}
			mux = cors.New(co).Handler(mux)
		}
		ocr.serverHTTP = &http.Server{Handler: mux, ReadHeaderTimeout: 20 * time.Second}
	}

	return ocr.serverHTTP
}

func (ocr *ocReceiver) startServer(host component.Host) error {
	// Register the grpc-gateway on the HTTP server mux
	c := context.Background()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := ocr.ln.Addr().String()

	_, ok := ocr.ln.(*net.UnixListener)
	if ok {
		endpoint = "unix:" + endpoint
	}

	if err := agenttracepb.RegisterTraceServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	if err := agentmetricspb.RegisterMetricsServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	// Start the gRPC and HTTP/JSON (grpc-gateway) servers on the same port.
	m := cmux.New(ocr.ln)
	grpcL := m.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"))

	httpL := m.Match(cmux.Any())
	go func() {
		// Check for cmux.ErrServerClosed, because during the shutdown this is not properly close before closing the cmux,
		// see TODO in Shutdown.
		if err := ocr.serverGRPC.Serve(grpcL); !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, cmux.ErrServerClosed) && err != nil {
			host.ReportFatalError(err)
		}
	}()
	go func() {
		if err := ocr.httpServer().Serve(httpL); !errors.Is(err, http.ErrServerClosed) && err != nil {
			host.ReportFatalError(err)
		}
	}()
	go func() {
		if err := m.Serve(); !errors.Is(err, cmux.ErrServerClosed) && err != nil {
			host.ReportFatalError(err)
		}
	}()
	return nil
}
