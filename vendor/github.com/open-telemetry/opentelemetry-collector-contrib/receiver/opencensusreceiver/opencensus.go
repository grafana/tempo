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
	"go.opentelemetry.io/collector/component/componentstatus"
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
	cfg                *Config
	ln                 net.Listener
	serverGRPC         *grpc.Server
	serverHTTP         *http.Server
	gatewayMux         *gatewayruntime.ServeMux
	corsOrigins        []string
	grpcServerSettings configgrpc.ServerConfig
	cancel             context.CancelFunc

	traceReceiver   *octrace.Receiver
	metricsReceiver *ocmetrics.Receiver

	traceConsumer   consumer.Traces
	metricsConsumer consumer.Metrics

	stopWG sync.WaitGroup

	settings    receiver.Settings
	multiplexer cmux.CMux
}

// newOpenCensusReceiver just creates the OpenCensus receiver services. It is the caller's
// responsibility to invoke the respective Start*Reception methods as well
// as the various Stop*Reception methods to end it.
func newOpenCensusReceiver(
	cfg *Config,
	tc consumer.Traces,
	mc consumer.Metrics,
	settings receiver.Settings,
	opts ...ocOption,
) *ocReceiver {
	ocr := &ocReceiver{
		cfg:             cfg,
		corsOrigins:     []string{}, // Disable CORS by default.
		gatewayMux:      gatewayruntime.NewServeMux(),
		traceConsumer:   tc,
		metricsConsumer: mc,
		settings:        settings,
	}

	for _, opt := range opts {
		opt.withReceiver(ocr)
	}

	return ocr
}

// Start runs the trace receiver on the gRPC server. Currently
// it also enables the metrics receiver too.
func (ocr *ocReceiver) Start(ctx context.Context, host component.Host) error {
	var err error
	ocr.serverGRPC, err = ocr.grpcServerSettings.ToServer(ctx, host, ocr.settings.TelemetrySettings)
	if err != nil {
		return err
	}
	var mux http.Handler = ocr.gatewayMux
	if len(ocr.corsOrigins) > 0 {
		co := cors.Options{AllowedOrigins: ocr.corsOrigins}
		mux = cors.New(co).Handler(mux)
	}
	ocr.serverHTTP = &http.Server{Handler: mux, ReadHeaderTimeout: 20 * time.Second}
	hasConsumer := false
	if ocr.traceConsumer != nil {
		hasConsumer = true
		ocr.traceReceiver, err = octrace.New(ocr.traceConsumer, ocr.settings)
		if err != nil {
			return err
		}
		agenttracepb.RegisterTraceServiceServer(ocr.serverGRPC, ocr.traceReceiver)
	}

	if ocr.metricsConsumer != nil {
		hasConsumer = true
		ocr.metricsReceiver, err = ocmetrics.New(ocr.metricsConsumer, ocr.settings)
		if err != nil {
			return err
		}
		agentmetricspb.RegisterMetricsServiceServer(ocr.serverGRPC, ocr.metricsReceiver)
	}

	if !hasConsumer {
		return errors.New("cannot start receiver: no consumers were specified")
	}

	ocr.ln, err = net.Listen(string(ocr.cfg.NetAddr.Transport), ocr.cfg.NetAddr.Endpoint)
	if err != nil {
		return fmt.Errorf("failed to bind to address %q: %w", ocr.cfg.NetAddr.Endpoint, err)
	}

	// Register the grpc-gateway on the HTTP server mux
	var c context.Context
	c, ocr.cancel = context.WithCancel(context.Background())

	endpoint := ocr.ln.Addr().String()

	_, ok := ocr.ln.(*net.UnixListener)
	if ok {
		endpoint = "unix:" + endpoint
	}

	// Start the gRPC and HTTP/JSON (grpc-gateway) servers on the same port.
	ocr.multiplexer = cmux.New(ocr.ln)
	grpcL := ocr.multiplexer.MatchWithWriters(
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"),
		cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc+proto"))

	httpL := ocr.multiplexer.Match(cmux.Any())
	ocr.stopWG.Add(1)
	startWG := sync.WaitGroup{}
	startWG.Add(3)

	go func() {
		defer ocr.stopWG.Done()
		startWG.Done()
		// Check for cmux.ErrServerClosed, because during the shutdown this is not properly close before closing the cmux,
		if err := ocr.serverGRPC.Serve(grpcL); err != nil && !errors.Is(err, grpc.ErrServerStopped) && !errors.Is(err, cmux.ErrServerClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(err))
		}
	}()
	go func() {
		startWG.Done()
		if err := ocr.serverHTTP.Serve(httpL); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, cmux.ErrServerClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(err))
		}
	}()
	go func() {
		startWG.Done()
		if err := ocr.multiplexer.Serve(); err != nil && !errors.Is(err, net.ErrClosed) {
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(err))
		}
	}()

	startWG.Wait()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := agenttracepb.RegisterTraceServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	if err := agentmetricspb.RegisterMetricsServiceHandlerFromEndpoint(c, ocr.gatewayMux, endpoint, opts); err != nil {
		return err
	}

	if ocr.serverGRPC == nil {
		var err error
		ocr.serverGRPC, err = ocr.grpcServerSettings.ToServer(context.Background(), host, ocr.settings.TelemetrySettings)
		if err != nil {
			return err
		}
	}

	// At this point we've successfully started all the services/receivers.
	// Add other start routines here.
	return nil
}

// Shutdown is a method to turn off receiving.
func (ocr *ocReceiver) Shutdown(context.Context) error {
	if ocr.cancel != nil {
		ocr.cancel()
	}

	if ocr.serverGRPC != nil {
		ocr.serverGRPC.Stop()
		ocr.stopWG.Wait()
	}

	if ocr.serverHTTP != nil {
		_ = ocr.serverHTTP.Close()
	}

	if ocr.ln != nil {
		_ = ocr.ln.Close()
	}

	if ocr.multiplexer != nil {
		ocr.multiplexer.Close()
	}

	ocr.traceConsumer = nil
	ocr.metricsConsumer = nil

	return nil
}
