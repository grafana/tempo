// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package octrace // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/octrace"

import (
	"context"
	"errors"
	"io"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agenttracepb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/trace/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/obsreport"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver"

	internaldata "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"
)

const (
	receiverTransport  = "grpc" // TODO: transport is being hard coded for now, investigate if info is available on context.
	receiverDataFormat = "protobuf"
)

// Receiver is the type used to handle spans from OpenCensus exporters.
type Receiver struct {
	agenttracepb.UnimplementedTraceServiceServer
	nextConsumer consumer.Traces
	obsrecv      *obsreport.Receiver
}

// New creates a new opencensus.Receiver reference.
func New(nextConsumer consumer.Traces, set receiver.CreateSettings) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}

	obsrecv, err := obsreport.NewReceiver(obsreport.ReceiverSettings{
		ReceiverID:             set.ID,
		Transport:              receiverTransport,
		LongLivedCtx:           true,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	return &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}, nil
}

var _ agenttracepb.TraceServiceServer = (*Receiver)(nil)

var errUnimplemented = errors.New("unimplemented")

// Config handles configuration messages.
func (ocr *Receiver) Config(agenttracepb.TraceService_ConfigServer) error {
	// TODO: Implement when we define the config receiver/sender.
	return errUnimplemented
}

var errTraceExportProtocolViolation = errors.New("protocol violation: Export's first message must have a Node")

// Export is the gRPC method that receives streamed traces from
// OpenCensus-traceproto compatible libraries/applications.
func (ocr *Receiver) Export(tes agenttracepb.TraceService_ExportServer) error {
	ctx := tes.Context()

	// The first message MUST have a non-nil Node.
	recv, err := tes.Recv()
	if err != nil {
		return err
	}

	// Check the condition that the first message has a non-nil Node.
	if recv.Node == nil {
		return errTraceExportProtocolViolation
	}

	var lastNonNilNode *commonpb.Node
	var resource *resourcepb.Resource
	// Now that we've got the first message with a Node, we can start to receive streamed up spans.
	for {
		lastNonNilNode, resource, err = ocr.processReceivedMsg(
			ctx,
			lastNonNilNode,
			resource,
			recv)
		if err != nil {
			return err
		}

		recv, err = tes.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Do not return EOF as an error so that grpc-gateway calls get an empty
				// response with HTTP status code 200 rather than a 500 error with EOF.
				return nil
			}
			return err
		}
	}
}

func (ocr *Receiver) processReceivedMsg(
	longLivedRPCCtx context.Context,
	lastNonNilNode *commonpb.Node,
	resource *resourcepb.Resource,
	recv *agenttracepb.ExportTraceServiceRequest,
) (*commonpb.Node, *resourcepb.Resource, error) {
	// If a Node has been sent from downstream, save and use it.
	if recv.Node != nil {
		lastNonNilNode = recv.Node
	}

	// TODO(songya): differentiate between unset and nil resource. See
	// https://github.com/census-instrumentation/opencensus-proto/issues/146.
	if recv.Resource != nil {
		resource = recv.Resource
	}

	td := internaldata.OCToTraces(lastNonNilNode, resource, recv.Spans)
	err := ocr.sendToNextConsumer(longLivedRPCCtx, td)
	return lastNonNilNode, resource, err
}

func (ocr *Receiver) sendToNextConsumer(longLivedRPCCtx context.Context, td ptrace.Traces) error {
	ctx := ocr.obsrecv.StartTracesOp(longLivedRPCCtx)

	err := ocr.nextConsumer.ConsumeTraces(ctx, td)
	ocr.obsrecv.EndTracesOp(ctx, receiverDataFormat, td.SpanCount(), err)

	return err
}
