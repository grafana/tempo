// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package jaegerreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/jaegerreceiver"

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"sync"

	apacheThrift "github.com/apache/thrift/lib/go/thrift"
	"github.com/gorilla/mux"
	"github.com/jaegertracing/jaeger/cmd/agent/app/configmanager"
	"github.com/jaegertracing/jaeger/cmd/agent/app/httpserver"
	"github.com/jaegertracing/jaeger/cmd/agent/app/processors"
	"github.com/jaegertracing/jaeger/cmd/agent/app/servers"
	"github.com/jaegertracing/jaeger/cmd/agent/app/servers/thriftudp"
	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/pkg/metrics"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
	"github.com/jaegertracing/jaeger/thrift-gen/agent"
	"github.com/jaegertracing/jaeger/thrift-gen/baggage"
	"github.com/jaegertracing/jaeger/thrift-gen/jaeger"
	"github.com/jaegertracing/jaeger/thrift-gen/zipkincore"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/multierr"
	"google.golang.org/grpc"

	jaegertranslator "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/jaeger"
)

// configuration defines the behavior and the ports that
// the Jaeger receiver will use.
type configuration struct {
	CollectorHTTPSettings       confighttp.HTTPServerSettings
	CollectorGRPCServerSettings configgrpc.GRPCServerSettings

	AgentCompactThrift ProtocolUDP
	AgentBinaryThrift  ProtocolUDP
	AgentHTTPEndpoint  string
}

// Receiver type is used to receive spans that were originally intended to be sent to Jaeger.
// This receiver is basically a Jaeger collector.
type jReceiver struct {
	nextConsumer consumer.Traces
	id           component.ID

	config *configuration

	grpc            *grpc.Server
	collectorServer *http.Server

	agentProcessors []processors.Processor
	agentServer     *http.Server

	goroutines sync.WaitGroup

	settings receiver.CreateSettings

	grpcObsrecv *receiverhelper.ObsReport
	httpObsrecv *receiverhelper.ObsReport
}

const (
	agentTransportBinary   = "udp_thrift_binary"
	agentTransportCompact  = "udp_thrift_compact"
	collectorHTTPTransport = "collector_http"
	grpcTransport          = "grpc"

	thriftFormat   = "thrift"
	protobufFormat = "protobuf"
)

var (
	acceptedThriftFormats = map[string]struct{}{
		"application/x-thrift":                 {},
		"application/vnd.apache.thrift.binary": {},
	}
)

// newJaegerReceiver creates a TracesReceiver that receives traffic as a Jaeger collector, and
// also as a Jaeger agent.
func newJaegerReceiver(
	id component.ID,
	config *configuration,
	nextConsumer consumer.Traces,
	set receiver.CreateSettings,
) (*jReceiver, error) {
	grpcObsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             id,
		Transport:              grpcTransport,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	httpObsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             id,
		Transport:              collectorHTTPTransport,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &jReceiver{
		config:       config,
		nextConsumer: nextConsumer,
		id:           id,
		settings:     set,
		grpcObsrecv:  grpcObsrecv,
		httpObsrecv:  httpObsrecv,
	}, nil
}

func (jr *jReceiver) Start(_ context.Context, host component.Host) error {
	if err := jr.startAgent(host); err != nil {
		return err
	}

	return jr.startCollector(host)
}

func (jr *jReceiver) Shutdown(ctx context.Context) error {
	var errs error

	if jr.agentServer != nil {
		if aerr := jr.agentServer.Shutdown(ctx); aerr != nil {
			errs = multierr.Append(errs, aerr)
		}
	}
	for _, processor := range jr.agentProcessors {
		processor.Stop()
	}

	if jr.collectorServer != nil {
		if cerr := jr.collectorServer.Shutdown(ctx); cerr != nil {
			errs = multierr.Append(errs, cerr)
		}
	}
	if jr.grpc != nil {
		jr.grpc.GracefulStop()
	}

	jr.goroutines.Wait()
	return errs
}

func consumeTraces(ctx context.Context, batch *jaeger.Batch, consumer consumer.Traces) (int, error) {
	if batch == nil {
		return 0, nil
	}
	td, err := jaegertranslator.ThriftToTraces(batch)
	if err != nil {
		return 0, err
	}
	return len(batch.Spans), consumer.ConsumeTraces(ctx, td)
}

var _ agent.Agent = (*agentHandler)(nil)
var _ api_v2.CollectorServiceServer = (*jReceiver)(nil)
var _ configmanager.ClientConfigManager = (*notImplementedConfigManager)(nil)

var errNotImplemented = fmt.Errorf("not implemented")

type notImplementedConfigManager struct{}

func (notImplementedConfigManager) GetSamplingStrategy(_ context.Context, _ string) (*api_v2.SamplingStrategyResponse, error) {
	return nil, errNotImplemented
}

func (notImplementedConfigManager) GetBaggageRestrictions(_ context.Context, _ string) ([]*baggage.BaggageRestriction, error) {
	return nil, errNotImplemented
}

type agentHandler struct {
	nextConsumer consumer.Traces
	obsrecv      *receiverhelper.ObsReport
}

// EmitZipkinBatch is unsupported agent's
func (h *agentHandler) EmitZipkinBatch(context.Context, []*zipkincore.Span) (err error) {
	panic("unsupported receiver")
}

// EmitBatch implements thrift-gen/agent/Agent and it forwards
// Jaeger spans received by the Jaeger agent processor.
func (h *agentHandler) EmitBatch(ctx context.Context, batch *jaeger.Batch) error {
	ctx = h.obsrecv.StartTracesOp(ctx)
	numSpans, err := consumeTraces(ctx, batch, h.nextConsumer)
	h.obsrecv.EndTracesOp(ctx, thriftFormat, numSpans, err)
	return err
}

func (jr *jReceiver) PostSpans(ctx context.Context, r *api_v2.PostSpansRequest) (*api_v2.PostSpansResponse, error) {
	ctx = jr.grpcObsrecv.StartTracesOp(ctx)

	batch := r.GetBatch()
	td, err := jaegertranslator.ProtoToTraces([]*model.Batch{&batch})
	if err != nil {
		jr.grpcObsrecv.EndTracesOp(ctx, protobufFormat, len(batch.Spans), err)
		return nil, err
	}

	err = jr.nextConsumer.ConsumeTraces(ctx, td)
	jr.grpcObsrecv.EndTracesOp(ctx, protobufFormat, len(batch.Spans), err)
	if err != nil {
		return nil, err
	}

	return &api_v2.PostSpansResponse{}, nil
}

func (jr *jReceiver) startAgent(host component.Host) error {
	if jr.config == nil {
		return nil
	}

	if jr.config.AgentBinaryThrift.Endpoint != "" {
		obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
			ReceiverID:             jr.id,
			Transport:              agentTransportBinary,
			ReceiverCreateSettings: jr.settings,
		})
		if err != nil {
			return err
		}

		h := &agentHandler{
			nextConsumer: jr.nextConsumer,
			obsrecv:      obsrecv,
		}
		processor, err := jr.buildProcessor(jr.config.AgentBinaryThrift.Endpoint, jr.config.AgentBinaryThrift.ServerConfigUDP, apacheThrift.NewTBinaryProtocolFactoryConf(nil), h)
		if err != nil {
			return err
		}
		jr.agentProcessors = append(jr.agentProcessors, processor)
	}

	if jr.config.AgentCompactThrift.Endpoint != "" {
		obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
			ReceiverID:             jr.id,
			Transport:              agentTransportCompact,
			ReceiverCreateSettings: jr.settings,
		})
		if err != nil {
			return err
		}
		h := &agentHandler{
			nextConsumer: jr.nextConsumer,
			obsrecv:      obsrecv,
		}
		processor, err := jr.buildProcessor(jr.config.AgentCompactThrift.Endpoint, jr.config.AgentCompactThrift.ServerConfigUDP, apacheThrift.NewTCompactProtocolFactoryConf(nil), h)
		if err != nil {
			return err
		}
		jr.agentProcessors = append(jr.agentProcessors, processor)
	}

	jr.goroutines.Add(len(jr.agentProcessors))
	for _, processor := range jr.agentProcessors {
		go func(p processors.Processor) {
			defer jr.goroutines.Done()
			p.Serve()
		}(processor)
	}

	if jr.config.AgentHTTPEndpoint != "" {
		jr.agentServer = httpserver.NewHTTPServer(jr.config.AgentHTTPEndpoint, &notImplementedConfigManager{}, metrics.NullFactory, jr.settings.Logger)

		jr.goroutines.Add(1)
		go func() {
			defer jr.goroutines.Done()
			if err := jr.agentServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) && err != nil {
				host.ReportFatalError(fmt.Errorf("jaeger agent server error: %w", err))
			}
		}()
	}

	return nil
}

func (jr *jReceiver) buildProcessor(address string, cfg ServerConfigUDP, factory apacheThrift.TProtocolFactory, a agent.Agent) (processors.Processor, error) {
	handler := agent.NewAgentProcessor(a)
	transport, err := thriftudp.NewTUDPServerTransport(address)
	if err != nil {
		return nil, err
	}
	if cfg.SocketBufferSize > 0 {
		if err = transport.SetSocketBufferSize(cfg.SocketBufferSize); err != nil {
			return nil, err
		}
	}
	server, err := servers.NewTBufferedServer(transport, cfg.QueueSize, cfg.MaxPacketSize, metrics.NullFactory)
	if err != nil {
		return nil, err
	}
	processor, err := processors.NewThriftProcessor(server, cfg.Workers, metrics.NullFactory, factory, handler, jr.settings.Logger)
	if err != nil {
		return nil, err
	}
	return processor, nil
}

func (jr *jReceiver) decodeThriftHTTPBody(r *http.Request) (*jaeger.Batch, *httpError) {
	bodyBytes, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		return nil, &httpError{
			fmt.Sprintf("Unable to process request body: %v", err),
			http.StatusInternalServerError,
		}
	}

	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, &httpError{
			fmt.Sprintf("Cannot parse content type: %v", err),
			http.StatusBadRequest,
		}
	}
	if _, ok := acceptedThriftFormats[contentType]; !ok {
		return nil, &httpError{
			fmt.Sprintf("Unsupported content type: %v", contentType),
			http.StatusBadRequest,
		}
	}

	tdes := apacheThrift.NewTDeserializer()
	batch := &jaeger.Batch{}
	if err = tdes.Read(r.Context(), batch, bodyBytes); err != nil {
		return nil, &httpError{
			fmt.Sprintf("Unable to process request body: %v", err),
			http.StatusBadRequest,
		}
	}
	return batch, nil
}

// HandleThriftHTTPBatch implements Jaeger HTTP Thrift handler.
func (jr *jReceiver) HandleThriftHTTPBatch(w http.ResponseWriter, r *http.Request) {
	ctx := jr.httpObsrecv.StartTracesOp(r.Context())

	batch, hErr := jr.decodeThriftHTTPBody(r)
	if hErr != nil {
		http.Error(w, html.EscapeString(hErr.msg), hErr.statusCode)
		jr.httpObsrecv.EndTracesOp(ctx, thriftFormat, 0, hErr)
		return
	}

	numSpans, err := consumeTraces(ctx, batch, jr.nextConsumer)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot submit Jaeger batch: %v", err), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
	jr.httpObsrecv.EndTracesOp(ctx, thriftFormat, numSpans, err)
}

func (jr *jReceiver) startCollector(host component.Host) error {
	if jr.config == nil {
		return nil
	}

	if jr.config.CollectorHTTPSettings.Endpoint != "" {
		cln, err := jr.config.CollectorHTTPSettings.ToListener()
		if err != nil {
			return fmt.Errorf("failed to bind to Collector address %q: %w",
				jr.config.CollectorHTTPSettings.Endpoint, err)
		}

		nr := mux.NewRouter()
		nr.HandleFunc("/api/traces", jr.HandleThriftHTTPBatch).Methods(http.MethodPost)
		jr.collectorServer, err = jr.config.CollectorHTTPSettings.ToServer(host, jr.settings.TelemetrySettings, nr)
		if err != nil {
			return err
		}

		jr.goroutines.Add(1)
		go func() {
			defer jr.goroutines.Done()
			if errHTTP := jr.collectorServer.Serve(cln); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
				host.ReportFatalError(errHTTP)
			}
		}()
	}

	if jr.config.CollectorGRPCServerSettings.NetAddr.Endpoint != "" {
		var err error
		jr.grpc, err = jr.config.CollectorGRPCServerSettings.ToServer(host, jr.settings.TelemetrySettings)
		if err != nil {
			return fmt.Errorf("failed to build the options for the Jaeger gRPC Collector: %w", err)
		}

		ln, err := jr.config.CollectorGRPCServerSettings.ToListener()
		if err != nil {
			return fmt.Errorf("failed to bind to gRPC address %q: %w", jr.config.CollectorGRPCServerSettings.NetAddr, err)
		}

		api_v2.RegisterCollectorServiceServer(jr.grpc, jr)

		jr.goroutines.Add(1)
		go func() {
			defer jr.goroutines.Done()
			if errGrpc := jr.grpc.Serve(ln); !errors.Is(errGrpc, grpc.ErrServerStopped) && errGrpc != nil {
				host.ReportFatalError(errGrpc)
			}
		}()
	}

	return nil
}
