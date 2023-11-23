// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package zipkinreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver"

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv1"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/zipkin/zipkinv2"
)

const (
	receiverTransportV1Thrift = "http_v1_thrift"
	receiverTransportV1JSON   = "http_v1_json"
	receiverTransportV2JSON   = "http_v2_json"
	receiverTransportV2PROTO  = "http_v2_proto"
)

var errNextConsumerRespBody = []byte(`"Internal Server Error"`)
var errBadRequestRespBody = []byte(`"Bad Request"`)

// zipkinReceiver type is used to handle spans received in the Zipkin format.
type zipkinReceiver struct {
	nextConsumer consumer.Traces

	shutdownWG sync.WaitGroup
	server     *http.Server
	config     *Config

	v1ThriftUnmarshaler      ptrace.Unmarshaler
	v1JSONUnmarshaler        ptrace.Unmarshaler
	jsonUnmarshaler          ptrace.Unmarshaler
	protobufUnmarshaler      ptrace.Unmarshaler
	protobufDebugUnmarshaler ptrace.Unmarshaler

	settings  receiver.CreateSettings
	obsrecvrs map[string]*receiverhelper.ObsReport
}

var _ http.Handler = (*zipkinReceiver)(nil)

// newReceiver creates a new zipkinReceiver reference.
func newReceiver(config *Config, nextConsumer consumer.Traces, settings receiver.CreateSettings) (*zipkinReceiver, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	transports := []string{receiverTransportV1Thrift, receiverTransportV1JSON, receiverTransportV2JSON, receiverTransportV2PROTO}
	obsrecvrs := make(map[string]*receiverhelper.ObsReport)
	for _, transport := range transports {
		obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
			ReceiverID:             settings.ID,
			Transport:              transport,
			ReceiverCreateSettings: settings,
		})
		if err != nil {
			return nil, err
		}
		obsrecvrs[transport] = obsrecv
	}

	zr := &zipkinReceiver{
		nextConsumer:             nextConsumer,
		config:                   config,
		v1ThriftUnmarshaler:      zipkinv1.NewThriftTracesUnmarshaler(),
		v1JSONUnmarshaler:        zipkinv1.NewJSONTracesUnmarshaler(config.ParseStringTags),
		jsonUnmarshaler:          zipkinv2.NewJSONTracesUnmarshaler(config.ParseStringTags),
		protobufUnmarshaler:      zipkinv2.NewProtobufTracesUnmarshaler(false, config.ParseStringTags),
		protobufDebugUnmarshaler: zipkinv2.NewProtobufTracesUnmarshaler(true, config.ParseStringTags),
		settings:                 settings,
		obsrecvrs:                obsrecvrs,
	}
	return zr, nil
}

// Start spins up the receiver's HTTP server and makes the receiver start its processing.
func (zr *zipkinReceiver) Start(_ context.Context, host component.Host) error {
	if host == nil {
		return errors.New("nil host")
	}

	var err error
	zr.server, err = zr.config.HTTPServerSettings.ToServer(host, zr.settings.TelemetrySettings, zr)
	if err != nil {
		return err
	}

	var listener net.Listener
	listener, err = zr.config.HTTPServerSettings.ToListener()
	if err != nil {
		return err
	}
	zr.shutdownWG.Add(1)
	go func() {
		defer zr.shutdownWG.Done()

		if errHTTP := zr.server.Serve(listener); !errors.Is(errHTTP, http.ErrServerClosed) && errHTTP != nil {
			host.ReportFatalError(errHTTP)
		}
	}()

	return nil
}

// v1ToTraceSpans parses Zipkin v1 JSON traces and converts them to OpenCensus Proto spans.
func (zr *zipkinReceiver) v1ToTraceSpans(blob []byte, hdr http.Header) (reqs ptrace.Traces, err error) {
	if hdr.Get("Content-Type") == "application/x-thrift" {
		return zr.v1ThriftUnmarshaler.UnmarshalTraces(blob)
	}
	return zr.v1JSONUnmarshaler.UnmarshalTraces(blob)
}

// v2ToTraceSpans parses Zipkin v2 JSON or Protobuf traces and converts them to OpenCensus Proto spans.
func (zr *zipkinReceiver) v2ToTraceSpans(blob []byte, hdr http.Header) (reqs ptrace.Traces, err error) {
	// This flag's reference is from:
	//      https://github.com/openzipkin/zipkin-go/blob/3793c981d4f621c0e3eb1457acffa2c1cc591384/proto/v2/zipkin.proto#L154
	debugWasSet := hdr.Get("X-B3-Flags") == "1"

	// By default, we'll assume using JSON
	unmarshaler := zr.jsonUnmarshaler

	// Zipkin can send protobuf via http
	if hdr.Get("Content-Type") == "application/x-protobuf" {
		// TODO: (@odeke-em) record the unique types of Content-Type uploads
		if debugWasSet {
			unmarshaler = zr.protobufDebugUnmarshaler
		} else {
			unmarshaler = zr.protobufUnmarshaler
		}
	}

	return unmarshaler.UnmarshalTraces(blob)
}

// Shutdown tells the receiver that should stop reception,
// giving it a chance to perform any necessary clean-up and shutting down
// its HTTP server.
func (zr *zipkinReceiver) Shutdown(context.Context) error {
	var err error
	if zr.server != nil {
		err = zr.server.Close()
	}
	zr.shutdownWG.Wait()
	return err
}

// processBodyIfNecessary checks the "Content-Encoding" HTTP header and if
// a compression such as "gzip", "deflate", "zlib", is found, the body will
// be uncompressed accordingly or return the body untouched if otherwise.
// Clients such as Zipkin-Java do this behavior e.g.
//
//	send "Content-Encoding":"gzip" of the JSON content.
func processBodyIfNecessary(req *http.Request) io.Reader {
	switch req.Header.Get("Content-Encoding") {
	default:
		return req.Body

	case "gzip":
		return gunzippedBodyIfPossible(req.Body)

	case "deflate", "zlib":
		return zlibUncompressedbody(req.Body)
	}
}

func gunzippedBodyIfPossible(r io.Reader) io.Reader {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		// Just return the old body as was
		return r
	}
	return gzr
}

func zlibUncompressedbody(r io.Reader) io.Reader {
	zr, err := zlib.NewReader(r)
	if err != nil {
		// Just return the old body as was
		return r
	}
	return zr
}

const (
	zipkinV1TagValue = "zipkinV1"
	zipkinV2TagValue = "zipkinV2"
)

// The zipkinReceiver receives spans from endpoint /api/v2 as JSON,
// unmarshalls them and sends them along to the nextConsumer.
func (zr *zipkinReceiver) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Now deserialize and process the spans.
	asZipkinv1 := r.URL != nil && strings.Contains(r.URL.Path, "api/v1/spans")

	transportTag := transportType(r, asZipkinv1)
	obsrecv := zr.obsrecvrs[transportTag]
	ctx = obsrecv.StartTracesOp(ctx)

	pr := processBodyIfNecessary(r)
	slurp, _ := io.ReadAll(pr)
	if c, ok := pr.(io.Closer); ok {
		_ = c.Close()
	}
	_ = r.Body.Close()

	var td ptrace.Traces
	var err error
	if asZipkinv1 {
		td, err = zr.v1ToTraceSpans(slurp, r.Header)
	} else {
		td, err = zr.v2ToTraceSpans(slurp, r.Header)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	consumerErr := zr.nextConsumer.ConsumeTraces(ctx, td)

	receiverTagValue := zipkinV2TagValue
	if asZipkinv1 {
		receiverTagValue = zipkinV1TagValue
	}
	obsrecv.EndTracesOp(ctx, receiverTagValue, td.SpanCount(), consumerErr)
	if consumerErr == nil {
		// Send back the response "Accepted" as
		// required at https://zipkin.io/zipkin-api/#/default/post_spans
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if consumererror.IsPermanent(consumerErr) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(errBadRequestRespBody)
	} else {
		// Transient error, due to some internal condition.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(errNextConsumerRespBody)
	}

}

func transportType(r *http.Request, asZipkinv1 bool) string {
	if asZipkinv1 {
		if r.Header.Get("Content-Type") == "application/x-thrift" {
			return receiverTransportV1Thrift
		}
		return receiverTransportV1JSON
	}
	if r.Header.Get("Content-Type") == "application/x-protobuf" {
		return receiverTransportV2PROTO
	}
	return receiverTransportV2JSON
}
