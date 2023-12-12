// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package ocmetrics // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/opencensusreceiver/internal/ocmetrics"

import (
	"context"
	"errors"
	"io"

	commonpb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"

	internaldata "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/opencensus"
)

// Receiver is the type used to handle metrics from OpenCensus exporters.
type Receiver struct {
	agentmetricspb.UnimplementedMetricsServiceServer
	nextConsumer consumer.Metrics
	obsrecv      *receiverhelper.ObsReport
}

// New creates a new ocmetrics.Receiver reference.
func New(nextConsumer consumer.Metrics, set receiver.CreateSettings) (*Receiver, error) {
	if nextConsumer == nil {
		return nil, component.ErrNilNextConsumer
	}
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              receiverTransport,
		LongLivedCtx:           true,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	ocr := &Receiver{
		nextConsumer: nextConsumer,
		obsrecv:      obsrecv,
	}
	return ocr, nil
}

var _ agentmetricspb.MetricsServiceServer = (*Receiver)(nil)

var errMetricsExportProtocolViolation = errors.New("protocol violation: Export's first message must have a Node")

const (
	receiverTransport  = "grpc" // TODO: transport is being hard coded for now, investigate if info is available on context.
	receiverDataFormat = "protobuf"
)

// Export is the gRPC method that receives streamed metrics from
// OpenCensus-metricproto compatible libraries/applications.
func (ocr *Receiver) Export(mes agentmetricspb.MetricsService_ExportServer) error {
	// Retrieve the first message. It MUST have a non-nil Node.
	recv, err := mes.Recv()
	if err != nil {
		return err
	}

	// Check the condition that the first message has a non-nil Node.
	if recv.Node == nil {
		return errMetricsExportProtocolViolation
	}

	var lastNonNilNode *commonpb.Node
	var resource *resourcepb.Resource
	// Now that we've got the first message with a Node, we can start to receive streamed up metrics.
	for {
		lastNonNilNode, resource, err = ocr.processReceivedMsg(
			mes.Context(),
			lastNonNilNode,
			resource,
			recv)
		if err != nil {
			return err
		}

		recv, err = mes.Recv()
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
	recv *agentmetricspb.ExportMetricsServiceRequest,
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

	err := ocr.sendToNextConsumer(longLivedRPCCtx, lastNonNilNode, resource, recv.Metrics)
	return lastNonNilNode, resource, err
}

func (ocr *Receiver) sendToNextConsumer(longLivedRPCCtx context.Context, node *commonpb.Node, resource *resourcepb.Resource, metrics []*ocmetrics.Metric) error {
	ctx := ocr.obsrecv.StartMetricsOp(longLivedRPCCtx)

	numPoints := 0
	// Count number of time series and data points.
	for _, metric := range metrics {
		for _, ts := range metric.GetTimeseries() {
			numPoints += len(ts.GetPoints())
		}
	}

	var consumerErr error
	if len(metrics) > 0 {
		consumerErr = ocr.nextConsumer.ConsumeMetrics(ctx, internaldata.OCToMetrics(node, resource, metrics))
	}

	ocr.obsrecv.EndMetricsOp(
		ctx,
		receiverDataFormat,
		numPoints,
		consumerErr)

	return consumerErr
}
