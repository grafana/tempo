// Copyright (c) 2018 The Jaeger Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"

	"github.com/jaegertracing/jaeger/cmd/collector/app/processor"
	"github.com/jaegertracing/jaeger/proto-gen/api_v2"
)

// GRPCHandler implements gRPC CollectorService.
type GRPCHandler struct {
	logger        *zap.Logger
	spanProcessor processor.SpanProcessor
}

// NewGRPCHandler registers routes for this handler on the given router.
func NewGRPCHandler(logger *zap.Logger, spanProcessor processor.SpanProcessor) *GRPCHandler {
	return &GRPCHandler{
		logger:        logger,
		spanProcessor: spanProcessor,
	}
}

// PostSpans implements gRPC CollectorService.
func (g *GRPCHandler) PostSpans(ctx context.Context, r *api_v2.PostSpansRequest) (*api_v2.PostSpansResponse, error) {
	for _, span := range r.GetBatch().Spans {
		if span.GetProcess() == nil {
			span.Process = r.Batch.Process
		}
	}
	_, err := g.spanProcessor.ProcessSpans(r.GetBatch().Spans, processor.SpansOptions{
		InboundTransport: processor.GRPCTransport,
		SpanFormat:       processor.ProtoSpanFormat,
	})
	if err != nil {
		if err == processor.ErrBusy {
			return nil, status.Errorf(codes.ResourceExhausted, err.Error())
		}
		g.logger.Error("cannot process spans", zap.Error(err))
		return nil, err
	}
	return &api_v2.PostSpansResponse{}, nil
}
