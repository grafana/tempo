// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/jaegerexporter"

import (
	"context"
	"fmt"
	"sync"
	"time"

	jaegerproto "github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
)

// newTracesExporter returns a new Jaeger gRPC exporter.
// The exporter name is the name to be used in the observability of the exporter.
// The collectorEndpoint should be of the form "hostname:14250" (a gRPC target).
func newTracesExporter(cfg *Config, set exporter.CreateSettings) (exporter.Traces, error) {
	s := newProtoGRPCSender(cfg, set)
	return exporterhelper.NewTracesExporter(
		context.TODO(), set, cfg, s.pushTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithStart(s.start),
		exporterhelper.WithShutdown(s.shutdown),
		exporterhelper.WithTimeout(cfg.TimeoutSettings),
		exporterhelper.WithRetry(cfg.RetrySettings),
		exporterhelper.WithQueue(cfg.QueueSettings),
	)
}

// protoGRPCSender forwards spans encoded in the jaeger proto
// format, to a grpc server.
type protoGRPCSender struct {
	name         string
	settings     component.TelemetrySettings
	client       jaegerproto.CollectorServiceClient
	metadata     metadata.MD
	waitForReady bool

	conn                      stateReporter
	connStateReporterInterval time.Duration
	stateChangeCallbacks      []func(connectivity.State)

	stopCh         chan struct{}
	stopped        bool
	stopLock       sync.Mutex
	clientSettings *configgrpc.GRPCClientSettings
}

func newProtoGRPCSender(cfg *Config, set exporter.CreateSettings) *protoGRPCSender {
	s := &protoGRPCSender{
		name:                      set.ID.String(),
		settings:                  set.TelemetrySettings,
		metadata:                  metadata.New(nil),
		waitForReady:              cfg.WaitForReady,
		connStateReporterInterval: time.Second,
		stopCh:                    make(chan struct{}),
		clientSettings:            &cfg.GRPCClientSettings,
	}
	for k, v := range cfg.GRPCClientSettings.Headers {
		s.metadata.Set(k, string(v))
	}
	s.AddStateChangeCallback(s.onStateChange)
	return s
}

type stateReporter interface {
	GetState() connectivity.State
}

func (s *protoGRPCSender) pushTraces(
	ctx context.Context,
	td ptrace.Traces,
) error {

	batches, err := jaeger.ProtoFromTraces(td)
	if err != nil {
		return consumererror.NewPermanent(fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err))
	}

	if s.metadata.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, s.metadata)
	}

	for _, batch := range batches {
		_, err = s.client.PostSpans(
			ctx,
			&jaegerproto.PostSpansRequest{Batch: *batch}, grpc.WaitForReady(s.waitForReady))

		if err != nil {
			s.settings.Logger.Debug("failed to push trace data to Jaeger", zap.Error(err))
			return fmt.Errorf("failed to push trace data via Jaeger exporter: %w", err)
		}
	}

	return nil
}

func (s *protoGRPCSender) shutdown(context.Context) error {
	s.stopLock.Lock()
	s.stopped = true
	s.stopLock.Unlock()
	close(s.stopCh)
	return nil
}

func (s *protoGRPCSender) start(ctx context.Context, host component.Host) error {
	if s.clientSettings == nil {
		return fmt.Errorf("client settings not found")
	}
	conn, err := s.clientSettings.ToClientConn(ctx, host, s.settings)
	if err != nil {
		return err
	}

	s.client = jaegerproto.NewCollectorServiceClient(conn)
	s.conn = conn

	go s.startConnectionStatusReporter()
	return nil
}

func (s *protoGRPCSender) startConnectionStatusReporter() {
	connState := s.conn.GetState()
	s.propagateStateChange(connState)

	ticker := time.NewTicker(s.connStateReporterInterval)
	for {
		select {
		case <-ticker.C:
			s.stopLock.Lock()
			if s.stopped {
				s.stopLock.Unlock()
				return
			}

			st := s.conn.GetState()
			if connState != st {
				// state has changed, report it
				connState = st
				s.propagateStateChange(st)
			}
			s.stopLock.Unlock()
		case <-s.stopCh:
			return
		}
	}
}

func (s *protoGRPCSender) propagateStateChange(st connectivity.State) {
	for _, callback := range s.stateChangeCallbacks {
		callback(st)
	}
}

func (s *protoGRPCSender) onStateChange(st connectivity.State) {
	_ = stats.RecordWithTags(context.Background(), []tag.Mutator{tag.Upsert(tag.MustNewKey("exporter_name"), s.name)}, mLastConnectionState.M(int64(st)))
	s.settings.Logger.Info("State of the connection with the Jaeger Collector backend", zap.Stringer("state", st))
}

func (s *protoGRPCSender) AddStateChangeCallback(f func(connectivity.State)) {
	s.stateChangeCallbacks = append(s.stateChangeCallbacks, f)
}
